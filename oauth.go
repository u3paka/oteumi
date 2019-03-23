package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/garyburd/go-oauth/oauth"
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
)

const (
	refreshTokenURL  = "https://api.twitter.com/oauth/request_token"
	authorizationURL = "https://api.twitter.com/oauth/authenticate"
	accessTokenURL   = "https://api.twitter.com/oauth/access_token"
	accountURL       = "https://api.twitter.com/1.1/account/verify_credentials.json"
)

type TwitterClient struct {
	*oauth.Client
}

func NewTwitterClient(key, secret string) TwitterClient {
	return TwitterClient{
		&oauth.Client{
			TemporaryCredentialRequestURI: refreshTokenURL,
			ResourceOwnerAuthorizationURI: authorizationURL,
			TokenRequestURI:               accessTokenURL,
			Credentials: oauth.Credentials{
				Token:  key,
				Secret: secret,
			},
		},
	}
}

func (ap App) NewTwitterClient() TwitterClient {
	return NewTwitterClient(ap.Redis.Get("system:auth:twtr_key").Val(), ap.Redis.Get("system:auth:twtr_secret").Val())
}

func (c TwitterClient) GetAccessToken(rt *oauth.Credentials, oauthVerifier string) (int, *oauth.Credentials, error) {
	at, _, err := c.RequestToken(nil, rt, oauthVerifier)
	if err != nil {
		err := errors.Wrap(err, "Failed to get access token.")
		return http.StatusBadRequest, nil, err
	}
	return http.StatusOK, at, nil
}

func (c TwitterClient) GetMe(at *oauth.Credentials, user interface{}) (int, error) {
	resp, err := c.Get(nil, at, accountURL, nil)
	if err != nil {
		err = errors.Wrap(err, "Failed to send twitter request.")
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		err = errors.New("Twitter is unavailable")
		return http.StatusInternalServerError, err
	}

	if resp.StatusCode >= 400 {
		err = errors.New("Twitter request is invalid")
		return http.StatusBadRequest, err
	}

	err = json.NewDecoder(resp.Body).Decode(user)
	if err != nil {
		err = errors.Wrap(err, "Failed to decode user account response.")
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil

}

var (
	store *sessions.CookieStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))
)

const (
	SessionName       = "session-name"
	ContextSessionKey = "session"
)

func applicationHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// セッションの取得
		session, err := store.Get(r, SessionName)
		if err != nil {
			// 不正なセッションだった場合は作り直す
			session, err = store.New(r, SessionName)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		context.Set(r, ContextSessionKey, session)
		// 個別のハンドラー呼び出し
		fn(w, r)
	}
}

func getSession(r *http.Request) (*sessions.Session, error) {
	if v := context.Get(r, ContextSessionKey); v != nil {
		return v.(*sessions.Session), nil
	}
	return nil, errors.New("failed to get session")
}

func (ap App) LoginByTwitter(w http.ResponseWriter, r *http.Request) {
	oc := ap.NewTwitterClient()
	rt, err := oc.RequestTemporaryCredentials(nil, ap.Conf.GetString("url.callback"), nil)
	if err != nil {
		pp.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := getSession(r)
	if err != nil {
		pp.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// session.Values
	//ltok := c.QueryParam("utype")
	//if ltok == LineUser{
	//	session.Set("line_token", c.QueryParam("uname"))
	//}

	session.Values["request_token"] = rt.Token
	session.Values["request_token_secret"] = rt.Secret
	session.Save(r, w)
	url := oc.AuthorizationURL(rt, nil)
	pp.Println("rediret:", url)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
	return
}

func (ap App) TwitterCallback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tok := r.PostFormValue("oauth_token")
	if tok == "" {
		pp.Println("no tok")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ov := r.PostFormValue("oauth_verifier")
	if ov == "" {
		pp.Println("no ov")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session, err := getSession(r)
	v, ok := session.Values["request_token"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rt, ok := v.(string)
	if tok != rt || !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	v, ok = session.Values["request_token_secret"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rts, ok := v.(string)
	if rts == "" || !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	twc := ap.NewTwitterClient()
	code, at, err := twc.GetAccessToken(&oauth.Credentials{Token: rt, Secret: rts}, ov)
	if err != nil {
		pp.Println(code, err)
		w.WriteHeader(code)
		return
	}
	account := struct {
		ID         string `json:"id_str"`
		ScreenName string `json:"screen_name"`
	}{}

	pp.Println(account.ScreenName)
	code, err = twc.GetMe(at, &account)
	//
	if err != nil {
		pp.Println(code, err)
		w.WriteHeader(code)
		return
	}
	session.Values["id"] = account.ID
	session.Values["screen_name"] = account.ScreenName
	if at != nil {
		func() {
			ap.Redis.SAdd(SET_TwitterAuth, account.ScreenName).Val()
			kk := fmt.Sprintf(Index_App, account.ScreenName)
			ap.Redis.HMSet(kk, map[string]string{
				"id":                  account.ID,
				"screen_name":         account.ScreenName,
				"access_token":        at.Token,
				"access_token_secret": at.Secret,
			})

			//TwitterとLINEの連携について
			v, ok := session.Values["line_token"]
			if !ok {
				return
			}
			ltok, ok := v.(string)
			if !ok {
			}
			session.Values["line_token"] = ltok
			pp.Println("line_token:", ltok)
			uname, err := ap.Redis.Get("auth:session:" + ltok).Result()
			if err != nil {
				return
			}
			k := fmt.Sprintf(Index_App, uname)
			ap.Redis.HMSet(k, map[string]string{
				"id":               uname,
				"twtr_screen_name": account.ScreenName,
				"twtr_id":          account.ID,
			})
			ap.Redis.BgSave()
			return
		}()
	}

	session.Save(r, w)
	http.Redirect(w, r, ap.Conf.GetString("url.domain")+"/user", http.StatusMovedPermanently)
	return
}
