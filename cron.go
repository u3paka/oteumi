package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/k0kubun/pp"
	"github.com/u3paka/markov"
	"github.com/u3paka/oteumi/crawler"
	"github.com/u3paka/oteumi/gazo"
	"github.com/u3paka/oteumi/reditool"
	"github.com/robfig/cron"
)

func (ap *App) CronFuncs() {
	ap.Cron = cron.New()
	ap.Cron.AddFunc("@every 1m", func() {
		fmt.Println("Redis:last backup: ", time.Unix(ap.Redis.LastSave().Val(), 0))
		ap.Redis.BgSave()
	})

	ap.Cron.AddFunc("@every 10m", func() {
		watch := ap.Conf.GetStringMapStringSlice("watch.amzn")
		urls := watch["url"]
		u := "umi0315ote"
		datas := make([]map[string]string, 0)
		for _, url := range urls {
			data, err := crawler.AmznCheck(url)
			pp.Println(data)
			if err != nil {
				continue
			}
			datas = append(datas, data)
		}
		tpl, err := template.New("main").Delims("<", ">").Parse(watch["txt"][0])
		if err != nil {
			fmt.Println(err)
			return
		}
		var buf bytes.Buffer
		tpl.Execute(&buf, datas)
		t := strings.TrimSpace(buf.String())
		if t == "" {
			return
		}
		pp.Println(t)
		tws, err := ap.NewTwtrService(u)
		if err != nil {
			fmt.Println(err)
			return
		}
		tws.replySession(&Session{
			ByName: u,
			Text:   t,
		})
	})

	//ap.Cron.AddFunc("@every 5m", func(){
	//	//TODO SET SRTR
	//	b := "kemofriends"
	//	tskey := fmt.Sprintf(ZKey_TimeStamp, b)
	//	//5分間接触のない場合、
	//	us := ap.Redis.ZRangeByScore(tskey, redis.ZRangeBy{
	//		Min: "-inf",
	//		Max: strconv.Itoa(int(time.Now().Add(-5*time.Minute).UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)))),
	//		Offset: 0,
	//		Count: -1,
	//	}).Val()
	//
	//	var ul = make([]interface{}, len(us))
	//	for i, u := range us{
	//		cxtKey := fmt.Sprintf(Index_BotUser, TwUser, u, b)
	//		//コンテクストの初期化
	//		ap.Redis.InitContext(cxtKey)
	//		ul[i] = u
	//	}
	//	rk := fmt.Sprintf("game:room:srtr:%s", b)
	//	//Roomから除外
	//	ap.Redis.SRem(rk, ul...)
	//	//TimeStampから除外
	//	ap.Redis.ZRem(tskey, ul...)
	//})
	// 315の海未ちゃんたいむ
	ap.Cron.AddFunc("00 10 03,15 * * *", func() {
		us, _ := ap.Redis.SInter(SET_TwitterAuth, apps_cron_315umi).Result()
		d := filepath.Join(ap.Conf.GetString("dir.pubroot"), ap.Conf.GetString("dir.img"), "categories", "umi")
		for _, u := range us {
			fmt.Println(u)
			s := &Session{
				ByName:  u,
				MsgType: "tweet",
				Footer:  "",
			}
			s.Text = "あと5分で #315の海未ちゃんたいむ ですよ！ みなさん準備はよろしいですか？"
			fmt.Println(s)
			s.ImgFiles = gazo.GetRandomImages(d, 1)
			tws, err := ap.NewTwtrService(u)
			if err != nil {
				fmt.Println(err)
				continue
			}
			tws.replySession(s)
		}
	})

	// 315の海未ちゃんたいむ
	ap.Cron.AddFunc("58 14 03,15 * * *", func() {
		us, _ := ap.Redis.SInter(SET_TwitterAuth, apps_cron_315umi).Result()
		d := filepath.Join(ap.Conf.GetString("dir.pubroot"), ap.Conf.GetString("dir.img"), "categories", "umi")
		for _, u := range us {
			fmt.Println(u)
			s := &Session{
				ByName:  u,
				MsgType: "tweet",
				Footer:  "",
			}
			k := fmt.Sprintf(Index_App, u)
			p, _ := reditool.HGetOrSet(ap.Redis, k, "char", default_char)
			mc := markov.NewTalkService(ap.DB, p)
			s.Text = mc.TrigramMarkovChain().String()
			fmt.Println(s)
			s.Footer = " #自動 #315の海未ちゃんたいむ"
			s.ImgFiles = gazo.GetRandomImages(d, 2)
			tws, err := ap.NewTwtrService(u)
			if err != nil {
				fmt.Println(err)
				continue
			}
			tws.replySession(s)
		}
	})

	////Generate定期
	ap.Cron.AddFunc("@every 20m", func() {
		us, _ := ap.Redis.SInter(SET_TwitterAuth, apps_cron_generate).Result()
		for _, u := range us {
			fmt.Println("GENERATE ", u)
			s := &Session{
				ByName:  u,
				MsgType: "tweet",
				Footer:  "",
			}
			k := fmt.Sprintf(Index_App, u)
			p, _ := reditool.HGetOrSet(ap.Redis, k, "char", default_char)
			mc := markov.NewTalkService(ap.DB, p)
			s.Text = mc.TrigramMarkovChain().String()
			fmt.Println(s)

			tws, err := ap.NewTwtrService(u)
			if err != nil {
				fmt.Println(err)
				continue
			}
			tws.replySession(s)

			rand.Seed(time.Now().UnixNano())
			time.Sleep(time.Millisecond * time.Duration(int64(rand.Intn(10000))))
		}
	})

	////334
	ap.Cron.AddFunc("00 34 03 * * *", func() {
		us, _ := ap.Redis.SInter(SET_TwitterAuth, apps_cron_334).Result()
		for _, u := range us {
			s := &Session{
				ByName:  u,
				MsgType: "tweet",
				Footer:  "",
			}
			s.Text = "334"
			s.Footer = ""
			tws, err := ap.NewTwtrService(u)
			if err != nil {
				fmt.Println(err)
				continue
			}
			tws.replySession(s)
		}
	})

	//// 遡りフォロバ
	//ap.Cron.AddFunc("@every 1h", func() {
	//	un := "kemofriends"
	//	v := url.Values{}
	//	v.Set("screen_name", un)
	//	api := ap.TwMap[un].Client
	//	pages := api.GetFollowersListAll(v)
	//	//follow_num := 10
	//	for {
	//		select {
	//		case <-time.After(time.Second * 10):
	//			pp.Print("follow back timeout")
	//			return
	//		case page := <-pages:
	//		//Print the current page of followers
	//			if page.Error != nil {
	//				fmt.Println(page)
	//			}
	//			//for _, user := range page.Followers {
	//			//	fmt.Println(user.ScreenName, user.Name, aa.IsSendFollowReq(user))
	//			//	//if follow_num > 0 && aa.IsSendFollowReq(user) {
	//			//	//	aa.api.FollowUser(user.ScreenName)
	//			//	//	follow_num -= 1
	//			//	//}
	//			//	time.Sleep(20*time.Second)
	//			//}
	//		}
	//	}
	//})
	ap.Cron.Start()
}
