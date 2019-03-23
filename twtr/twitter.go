package twtr

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/dustin/gojson"
)

type Client struct {
	*twitter.Client
	HttpClient *http.Client
}

const (
	post                     = "POST"
	get                      = "GET"
	twitter_media_upload_url = "https://upload.twitter.com/1.1/media/upload.json"
	updateIconUrl            = "https://api.twitter.com/1.1/account/update_profile_image.json"
	updateProfileUrl         = "https://api.twitter.com/1.1/account/update_profile.json"
)

func (cli *Client) UploadMedias(srcs []string) (ids []int64, err error) {
	err = nil
	if len(srcs) == 0 {
		return
	}
	for i, src := range srcs {
		if i > 3 {
			break
		}
		if _, err := os.Stat(src); err != nil {
			continue
		}
		func() {
			req, err := newRequest(post, twitter_media_upload_url, map[string]string{}, map[string]string{
				"media": src,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			resp, err := cli.HttpClient.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()
			d := json.NewDecoder(resp.Body)
			var v map[string]interface{}
			err = d.Decode(&v)
			if err != nil {
				return
			}
			//{\"media_id\":832555156990414848,\"media_id_string\":\"832555156990414848\",
			// \"size\":56060,\"expires_after_secs\":86400,\"image\":{
			// \"image_type\":\"image\\/jpeg\"	,\"w\":400,\"h\":400}}"
			r, err := strconv.Atoi(v["media_id_string"].(string))
			if err != nil {
				return
			}
			ids = append(ids, int64(r))
		}()
	}
	return
}

func NewTwtrClient(ck, cks, at, ats string) *Client {
	config := oauth1.NewConfig(ck, cks)
	token := oauth1.NewToken(at, ats)
	hcli := config.Client(oauth1.NoContext, token)
	tcli := twitter.NewClient(hcli)
	return &Client{
		tcli,
		hcli,
	}
}

type Profile struct {
	Name        string
	Description string
	Location    string
	Icon        string
	Banner      string
}

func (cli *Client) UpdateIcon(src string) (u twitter.User, err error) {
	_, err = os.Stat(src)
	if err != nil {
		return
	}
	req, err := newRequest(post, updateIconUrl, map[string]string{}, map[string]string{
		"image": src,
	})
	if err != nil {
		return
	}
	resp, err := cli.HttpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&u)
	if err != nil {
		return
	}
	return
}

func (cli *Client) UpdateProfile(name, description, location string) (u twitter.User, err error) {
	v := url.Values{}
	if name != "" {
		v.Set("name", name)
	}
	if description != "" {
		v.Set("description", description)
	}
	if location != "" {
		v.Set("location", location)
	}
	resp, _ := cli.HttpClient.PostForm(updateProfileUrl, v)
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&u)
	return
}
