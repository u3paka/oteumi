package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/jmcvetta/randutil"
	"github.com/k0kubun/pp"
	"github.com/u3paka/jumangok/jmg"
	"github.com/u3paka/oteumi/gazo"
	"github.com/u3paka/oteumi/reditool"
	"github.com/u3paka/oteumi/twtr"
)

const (
	probPubTrap = "prob.pubtrap"
	probAgrv    = "prob.agrv"
	lockAgrv    = "lock:agrv:"
	lockAgrvKey = "ttl.lock.argv"
)

type TwtrService struct {
	*App
	Client   *twtr.Client
	UserName string
	Persona  string
	CoolAidC chan interface{}
}

func (ap *App) NewTwtrService(uname string) (*TwtrService, error) {
	ck := ap.Redis.HGet("app:twtr", "twtr_consumer_key").Val()
	cks := ap.Redis.HGet("app:twtr", "twtr_consumer_secret").Val()
	k := fmt.Sprintf(Index_App, uname)
	hash := ap.Redis.HGetAll(k).Val()
	at := hash["access_token"]
	ats := hash["access_token_secret"]
	if at == "" || ats == "" {
		fmt.Println("not TwitterAuthorized...", uname)
		return &TwtrService{}, errors.New("not TwitterAuthorized..." + uname)
	}
	char, _ := reditool.HGetOrSet(ap.Redis, k, "char", default_char)
	twc := twtr.NewTwtrClient(ck, cks, at, ats)
	return &TwtrService{
		ap,
		twc,
		uname,
		char,
		make(chan interface{})}, nil
}

func (s *TwtrService) Stream(d time.Duration) {
	if s.Redis.SIsMember(apps_t_streaming, s.UserName).Val() {
		fmt.Println(s.UserName + " is already streaming")
		return
	}
	s.Redis.SMove(apps_t_wait, apps_t_streaming, s.UserName).Err()
	defer s.Redis.SMove(apps_t_streaming, apps_t_wait, s.UserName).Err()

	stream, err := s.Client.Streams.User(&twitter.StreamUserParams{
		With:          "followings",
		StallWarnings: twitter.Bool(true),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	demux := twitter.NewSwitchDemux()

	demux.StatusDeletion = func(deletion *twitter.StatusDeletion) {
		// fmt.Println("Deletion: " + deletion.UserIDStr)
	}

	demux.Warning = func(warning *twitter.StallWarning) {
		fmt.Println(warning)
	}

	demux.Tweet = func(tweet *twitter.Tweet) {
		go s.handleTweet(tweet)
		if s.Redis != nil {
			pipe := s.Redis.Pipeline()
			pipe.HMSet(fmt.Sprintf(keyUid, TwUser, tweet.User.ScreenName), map[string]string{
				"id":          tweet.User.IDStr,
				"name":        tweet.User.Name,
				"screen_name": tweet.User.ScreenName,
			})
			pipe.RPush(fmt.Sprintf(Index_StatusIds, TwUser, tweet.User.ScreenName), tweet.IDStr).Err()
			pipe.Exec()
		}
	}

	demux.DM = func(dm *twitter.DirectMessage) {
		if dm.SenderScreenName == s.UserName {
			return
		}
		fmt.Println(dm.Text)
		sess := &Session{
			ByName:   s.UserName,
			FromName: dm.SenderScreenName,
			ToName:   dm.SenderScreenName,
			Token:    dm.IDStr,
			MsgType:  "dm",
		}
		out := s.Chat.LoadContext(dm.SenderScreenName).MSet(map[string]interface{}{
			"session_userid":   dm.SenderScreenName,
			"session_text":     dm.Text,
			"session_sendto":   dm.SenderScreenName,
			"session_msgtype":  "dm",
			"session_duration": "0s",
		}).Do(dm.Text)

		sess.Text = out.Out
		d, _ := out.Get("session_duration")
		var ok bool
		sess.Duration, ok = d.(string)
		if !ok {
			sess.Duration = "0s"
		}
		s.replySession(sess)
	}

	demux.Event = func(event *twitter.Event) {
		fmt.Printf("%#v\n", event)
		switch event.Event {
		case "follow":
			if event.Source.ScreenName == s.UserName {
				return
			}
			sess := &Session{
				ByName:   s.UserName,
				FromName: event.Source.ScreenName,
				ToName:   event.Source.ScreenName,
				Token:    "",
				MsgType:  "twtr",
			}
			out := s.Chat.LoadContext(event.Source.ScreenName).MSet(map[string]interface{}{
				"session_userid":   event.Source.ScreenName,
				"session_text":     "",
				"session_sendto":   event.Source.ScreenName,
				"session_username": event.Source.Name,
				"session_msgtype":  "twtr",
				"session_duration": "0s",
			}).Express("event.follow")

			sess.Text = out.Out
			st, _ := out.Get("session_sendto")
			var ok bool
			sess.ToName, ok = st.(string)
			if !ok {
				sess.ToName = event.Source.ScreenName
			}

			d, _ := out.Get("session_duration")
			sess.Duration, ok = d.(string)
			if !ok {
				sess.Duration = "0s"
			}
			s.replySession(sess)
		case "list_member_added":
			if event.Source.ScreenName == s.UserName {
				return
			}
			sess := &Session{
				ByName:   s.UserName,
				FromName: event.Source.ScreenName,
				ToName:   event.Source.ScreenName,
				Token:    "",
				MsgType:  "twtr",
			}
			out := s.Chat.LoadContext(event.Source.ScreenName).MSet(map[string]interface{}{
				"session_userid":   event.Source.ScreenName,
				"session_text":     "",
				"session_sendto":   "",
				"session_username": event.Source.Name,
				"session_msgtype":  "twtr",
				"session_duration": "0s",
			}).Express("event.list_member_added")
			sess.Text = out.Out
			st, _ := out.Get("session_sendto")
			var ok bool
			sess.ToName, ok = st.(string)
			if !ok {
				sess.ToName = event.Source.ScreenName
			}

			d, _ := out.Get("session_duration")
			sess.Duration, ok = d.(string)
			if !ok {
				sess.Duration = "0s"
			}
			s.replySession(sess)

		case "unfollow":
			s.Client.Friendships.Destroy(&twitter.FriendshipDestroyParams{
				ScreenName: event.Source.ScreenName,
			})

		case "favorite":
			if event.Source.ScreenName == s.UserName {
				return
			}
			utype := TwUser
			s.Redis.ZIncrBy(fmt.Sprintf(ZKey_Fav, utype, s.UserName), 1.0, event.Source.ScreenName)
			rand.Seed(time.Now().UnixNano())
			rtmpl := s.Redis.SRandMember(fmt.Sprintf(Index_FIXED_DIALOG, event.Event, s.UserName)).Val()
			fmt.Println(rtmpl)
			if rtmpl != "" && rand.Intn(100) < 5 {
				s.Client.Statuses.Update(fmt.Sprintf(rtmpl, event.Source.Name), &twitter.StatusUpdateParams{
					Status: fmt.Sprintf(rtmpl, event.Source.Name),
				})
			}
			return
		case "unfavorite":
			if event.Source.ScreenName == s.UserName {
				return
			}
			s.Redis.ZIncrBy(fmt.Sprintf(ZKey_Fav, TwUser, s.UserName), -1.0, event.Source.ScreenName)
		}
	}
	demux.FriendsList = func(fl *twitter.FriendsList) {
		fmt.Println("FRIENDS", fl)
	}

	fmt.Println(s.UserName, "Starting Stream...")
	go demux.HandleChan(stream.Messages)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
		fmt.Println("Stopping Stream...")
		stream.Stop()
		s.Redis.SRem(apps_t_streaming, s.UserName)
	case <-s.CoolAidC:
		fmt.Println("Stopping Stream...by CoolAid")
		stream.Stop()
		s.Redis.SRem(apps_t_streaming, s.UserName)
	}
	os.Exit(1)
	return
}

func (s *TwtrService) ReactFilter(tweet *twitter.Tweet) (*twitter.Tweet, bool) {
	if strings.HasSuffix(tweet.Source, s.Conf.GetString("via")+"</a>") {
		return tweet, false
	}

	// RTは除外
	if strings.HasPrefix(tweet.Text, "RT") || tweet.Retweeted {
		return tweet, false
	}

	//// 空ツイートは除外
	if tweet.Text == "" {
		return tweet, false
	}

	//自分へのメンションは反応
	if strings.Contains(tweet.Text, s.UserName) || tweet.InReplyToScreenName == s.UserName {
		tweet.Text = cleanTweet(tweet.Text)
		return tweet, true
	}
	tweet.Text = cleanTweet(tweet.Text)

	// 自ツイートへの反応 :から始まるときのみ反応する
	const CommandMark = ":"
	force := strings.HasPrefix(tweet.Text, CommandMark)
	if force {
		tweet.Text = strings.TrimPrefix(tweet.Text, CommandMark)
	}

	switch {
	//自ツイートへの反応 :から始まるときのみ反応する。 e.g. :kusoripu
	case tweet.User.ScreenName == s.UserName && force:
		return tweet, force

	case tweet.InReplyToScreenName != "":
		return tweet, false

	// TTL Lock期間は反応しない
	case s.Redis.PTTL(lockAgrv+tweet.User.ScreenName).Val() > 0:
		fmt.Println("LOCKed:AGRV REACT", tweet.User.ScreenName)
		return tweet, false

	// 自発絡み禁止
	case s.Redis.SIsMember(usersAgrvNG, tweet.User.ScreenName).Val():
		return tweet, false

	// クソリプ検知
	case Kusoripu(tweet.Text):
		// out := s.Chat.LoadContext(s.UserName).MSet(map[string]interface{}{
		// 	"keyword": s.UserName,
		// }).Express(fmt.Sprintf("trap.%s", kind))
		return tweet, false
	}
	// TLへのアグレッシブリアクション　設定単語に反応してランダム返答する。
	for k, vs := range s.Conf.GetStringMapStringSlice("react") {
		for _, kk := range strings.Split(k, ",") {
			kk = strings.TrimSpace(kk)
			if strings.Contains(tweet.Text, kk) {
				goto agrv
			}
		}
		continue
	agrv:
		v, err := randutil.ChoiceString(vs)
		if err != nil {
			return tweet, false
		}
		s.replySession(&Session{
			Text:     v,
			ByName:   s.UserName,
			FromName: s.UserName,
			Token:    tweet.IDStr,
			ToName:   tweet.User.ScreenName,
		})
		// TTL Lock
		reditool.TTLLock(s.Redis, lockAgrv+tweet.User.ScreenName, s.Conf.GetDuration(lockAgrvKey))
		pp.Println("LOCK", lockAgrv+tweet.User.ScreenName)
		return tweet, false
	}

	// TRAP TL上のツイートを取得 -> 名詞抽出し、Publicツイート
	// if s.Redis.SIsMember(apps_trapping, s.UserName).Val() {
	// 	var next bool
	// 	tweet, next = s.handleTrap(tweet)
	// 	if !next {
	// 		return tweet, false
	// 	}
	// }

	// rand.Seed(time.Now().UnixNano())
	// if rand.Intn(100) < s.Conf.GetInt(probAgrv) {
	// 	return tweet, true
	// }
	return tweet, false
}

func (s *TwtrService) handleTrap(status *twitter.Tweet) (*twitter.Tweet, bool) {
	key := fmt.Sprintf(Index_TrapLs, s.UserName)
	kind := s.Redis.LPop(key).Val()
	if kind == "" {
		return status, true
	}

	sess := &Session{
		ByName:   s.UserName,
		FromName: status.User.ScreenName,
		Token:    status.IDStr,
		ToName:   status.User.ScreenName,
	}

	ws, err := s.JmgCli.Jumanpp(context.Background(), status.Text)
	if err != nil {
		fmt.Println(err)
		return status, true
	}
	ks := jmg.Extract(ws, func(w *jmg.Word) bool {
		if w.Pos == "名詞" {
			return true
		}
		return false
	})
	if len(ks) == 0 {
		return status, true
	}
	out := s.Chat.LoadContext(s.UserName).MSet(map[string]interface{}{
		"keyword": ks[0],
	}).Express(fmt.Sprintf("trap.%s", kind))

	if out.Out == "" {
		s.Redis.LPush(key, kind).Val()
		return status, true
	}
	sess.Text = out.Out
	// 指定確率でpublicツイートになる。
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(100) < s.Conf.GetInt(probPubTrap) {
		sess.ToName = ""
		//パブリックツイートなら、ファボする。
		s.SafetyFav(status.IDStr)
	}
	s.replySession(sess)
	//TTLLock
	reditool.TTLLock(s.Redis, lockAgrv+status.User.ScreenName, s.Conf.GetDuration(lockAgrvKey))
	pp.Println("LOCK", lockAgrv+status.User.ScreenName)
	return status, false
}

func (s *TwtrService) handleTweet(status *twitter.Tweet) error {
	// recover
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}
			fmt.Println(err)
		}
	}()

	// 時限式詫びFav
	favTimer := time.AfterFunc(time.Second*3, func() {
		s.SafetyFav(status.IDStr)
	})
	defer func() {
		// stop Fav timer
		if favTimer != nil {
			if !favTimer.Stop() {
				<-favTimer.C
			}
		}
	}()

	sess := &Session{
		ByName:   s.UserName,
		FromName: status.User.ScreenName,
		ToName:   status.User.ScreenName,
		Token:    status.IDStr,
	}
	var ok bool
	status, ok = s.ReactFilter(status)
	if !ok {
		return nil
	}

	imgs := s.GetImages(status)

	out := s.Chat.LoadContext(status.User.ScreenName).MSet(map[string]interface{}{
		"session_userid":   status.User.ScreenName,
		"session_text":     status.Text,
		"session_sendto":   status.User.ScreenName,
		"session_msgtype":  "twtr",
		"session_duration": "0s",
	}).Do(status.Text)

	sess.Text = out.Out
	st, _ := out.Get("session_sendto")

	sess.ToName, ok = st.(string)
	if !ok {
		sess.ToName = ""
	}

	d, _ := out.Get("session_duration")
	sess.Duration, ok = d.(string)
	if !ok {
		sess.Duration = "0s"
	}
	if len(imgs) > 0 {
		// s.handleImages(sess, imgs)
		// TODO
	}
	// SEND
	s.replySession(sess)
	return nil
}

func (s *TwtrService) GetImages(status *twitter.Tweet) []string {
	if status.ExtendedEntities == nil {
		return []string{}
	}
	if status.ExtendedEntities.Media == nil {
		return []string{}
	}
	imgfiles := make([]string, len(status.ExtendedEntities.Media))
	// Download Image
	for k, media := range status.ExtendedEntities.Media {
		AbsImgPath, err := gazo.DownloadImage(media.MediaURL+":orig", s.Conf.GetString("dir.img"), "twitter", status.User.ScreenName)
		if err == nil {
			imgfiles[k] = AbsImgPath
		}
	}
	return imgfiles
}

func (s *TwtrService) SafetyFav(id string) {
	if !s.Redis.SIsMember(apps_fav, s.UserName).Val() {
		return
	}
	k := fmt.Sprintf(Index_App, s.UserName) + ":fav"
	v := s.Redis.IncrBy(k, 1).Val()
	if v == 1 {
		s.Redis.PExpire(k, time.Minute*30)
	}
	lim, err := s.Redis.HGet(fmt.Sprintf(Index_App, s.UserName), "favlimit").Int64()
	if err != nil {
		fmt.Println(err)
		return
	}
	if v > lim {
		du := s.Redis.PTTL(k).Val()
		time.AfterFunc(du, func() {
			s.SafetyFav(id)
		})
		return
	}
	sid, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println(err)
		return
	}
	s.Client.Favorites.Create(&twitter.FavoriteCreateParams{
		ID: int64(sid),
	})
	return
}

func (s *TwtrService) replySession(x *Session) error {
	delay, _ := time.ParseDuration(x.Duration)
	if x.Text == "" {
		return errors.New("reply session is failed. empty text")
	}
	switch x.MsgType {
	default:
		// リプライ規制
		appKey := fmt.Sprintf(Index_App, x.ByName)
		cntKey := appKey + ":cnt"
		cnt := s.Redis.IncrBy(cntKey, 1).Val()

		repLimitCnt := s.Conf.GetInt64("limit.reply")
		twLimitCnt := s.Conf.GetInt64("limit.tweet")
		switch {
		case cnt == 1:
			s.Redis.PExpire(cntKey, time.Minute*30)

		case cnt == repLimitCnt:
			fmt.Println("start self-restriction tweet")
			du := s.Redis.PTTL(cntKey).Val()
			time.AfterFunc(du, func() {
				const layout = `15:04:05`
				//TODO Template
				out := s.Chat.LoadContext("").MSet(map[string]interface{}{
					"nowstr": time.Now().Format(layout),
				}).Express("self-restriction.end")
				// text := fmt.Sprintf(s.Redis.SRandMember(fmt.Sprintf(Index_FIXED_DIALOG, x.ByName, "self-restriction")).Val(), nowstr)
				_, _, err := s.Client.Statuses.Update(out.Out, nil)
				if err != nil {
					fmt.Println(err)
				}
			})
			out := s.Chat.LoadContext("").MSet(map[string]interface{}{
				"resttime": du.String(),
			}).Express("self-restriction.begin")
			fmt.Println(out.Out)
			// text := fmt.Sprintf(s.Redis.SRandMember(fmt.Sprintf(Index_FIXED_DIALOG, x.ByName, "kisei")).Val(), du.String())
			_, _, err := s.Client.Statuses.Update(out.Out, nil)
			if err != nil {
				fmt.Println(err)
			}
			return nil
		case cnt > repLimitCnt && x.ToName != "", cnt > twLimitCnt:
			fmt.Println("self-restriction tweet", repLimitCnt)
			s.SafetyFav(x.Token)
			return nil
		}

		if strings.HasPrefix(x.ToName, "@") {
			x.ToName = strings.Trim(x.ToName, "@")
		}
		if strings.HasPrefix(x.Text, "!") {
			x.Text = strings.Trim(x.Text, "!")
		}

		// 送信対象と命令者が異なり、フォローチェックが必要な場合
		if x.ToName != "" && x.ToName != x.FromName {
			err := s.Redis.SUnionStore("tmp:friends:"+x.ByName+"-"+x.FromName, fmt.Sprintf(keyUidFriends, TwUser, x.ByName), fmt.Sprintf(keyUidFriends, TwUser, x.FromName)).Err()
			if err != nil {
				fmt.Println(err)
				return err
			}
			if !s.Redis.SIsMember("tmp:friends:"+x.ByName+"-"+x.FromName, x.ToName).Val() {
				u, _, err := s.Client.Users.Show(&twitter.UserShowParams{
					ScreenName: x.ToName,
				})
				if err != nil {
					fmt.Println(err)
					return err
				}
				if !u.Following {
					go s.SafetyFav(x.Token)
					x.Text = " [RESTRICTED]\n cause: " + x.ToName + " is NOT our common friend..."
					x.ToName = x.FromName
				}
			}
		}
		tf := func() {
			text, mids, sid64, err := s.replyContents(x)
			if err != nil {
				fmt.Println(err)
				return
			}
			_, _, err = s.Client.Statuses.Update(text, &twitter.StatusUpdateParams{
				InReplyToStatusID: sid64,
				MediaIds:          mids,
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		}
		time.AfterFunc(delay, tf)
	case "dm":
		x.Text = addFooter(x.Text, "...", 10000)
		tf := func() {
			_, _, err := s.Client.DirectMessages.New(&twitter.DirectMessageNewParams{
				ScreenName: x.ToName,
				Text:       x.Text,
			})
			if err != nil {
				fmt.Println(err)
			}
		}
		time.AfterFunc(delay, tf)
	}
	return nil
}

func (tws *TwtrService) replyContents(x *Session) (t string, mids []int64, sid int64, err error) {
	// text => @USERID text...
	if x.ToName != "" {
		if !strings.HasPrefix(x.ToName, "@") {
			x.Text = fmt.Sprintf("@%s %s", x.ToName, x.Text)
		} else {
			x.Text = fmt.Sprintf("%s %s", x.ToName, x.Text)
		}
	}
	if x.Text == "" {
		err = errors.New("empty tweet body")
		return
	}
	// 140 chars limit
	t = addFooter(x.Text, x.Footer, 140)
	mids, _ = tws.Client.UploadMedias(x.ImgFiles)

	if x.Token == "" {
		return
	}

	token, err := strconv.Atoi(x.Token)
	sid = int64(token)
	return
}

func addFooter(msg, footer string, max int) string {
	rt := []rune(msg)
	ft := []rune(footer)
	lrt := len(rt)
	lft := len(ft)
	if lrt > max-lft {
		msg = string(rt[:max-lft])
	}
	msg += footer
	return msg
}

func cleanTweet(text string) string {
	if strings.HasPrefix(text, "@") {
		lst := strings.SplitN(text, " ", 2)
		if len(lst) < 2 {
			text = ""
		} else {
			text = lst[1]
		}
	}
	if ls := strings.Split(text, "http"); len(ls) > 1 {
		r := make([]string, 1)
		r[0] = ls[0]
		for _, tx := range ls[1:] {
			txls := strings.SplitN(tx, " ", 2)
			r = append(r, txls[1:]...)
		}
		text = strings.Join(r, "")
	}
	return text
}
