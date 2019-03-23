package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/u3paka/oteumi/gazo"
)

func TestInt(t *testing.T) {
	type test struct {
		ii int64
	}
	tt := test{}
	t.Log(tt)
	t.Log(tt.ii)
	if tt.ii != int64(0) {
		t.Failed()
	}
}

func TestAddFooter(t *testing.T) {
	fmt.Println(addFooter("んひー", "(パクツイ便乗)", 140))
	x := addFooter(`ああああああああああああああああああああああああああ
	あああああああああああああああああああああああああああああああああああああああああ
	あああああああああああああああああああああああああああああああああああ
	ああああああああああああああああああああああああああああああああ
	あああああああああああああ
	あああああああああ
	あああああああああ`, "(end)", 140)
	fmt.Println(len([]rune(x)))
	pp.Print(x)
}

func TestTstream(t *testing.T) {
	ap := NewApp()
	ap.RedisCon(ap.Conf.GetString("address.redis"))
	//tws, err := ap.NewTwtrService("umi0315ote")
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	ap.RinchatCon()
	us, _ := ap.Redis.SInter(SET_TwitterAuth, apps_cron_315umi).Result()
	d := filepath.Join(ap.Conf.GetString("dir.pubroot"), ap.Conf.GetString("dir.img"), "categories", "umi")
	for _, u := range us {
		fmt.Println(u)
		s := &Session{
			ByName:  u,
			MsgType: "tweet",
			Footer:  "",
		}
		s.Text = "お昼のうみちゃんたいむ！"
		s.Footer = "#自動 #315の海未ちゃんたいむ"
		s.ImgFiles = gazo.GetRandomImages(d, 1)
		fmt.Println(s)
		tws, err := ap.NewTwtrService(u)
		if err != nil {
			fmt.Println(err)
			continue
		}
		tws.replySession(s)
	}

	//tws.Stream(time.Minute*5)
	//s := bufio.NewScanner(os.Stdin)
	//for s.Scan() {
	//	tx := s.Text()
	//	switch tx{
	//	case "exit":
	//		tws.CoolAidC <- true
	//		//for uname, tws := range ap.TwMap{
	//		//	tws <- true
	//		//	fmt.Println("cleaning...", uname)
	//		//	<- tws.CoolAidC
	//		//}
	//		os.Exit(1)
	//	default:
	//		//log.Print(tx)
	//	}
	//}
	//if s.Err() != nil {
	//	// non-EOF error.
	//	log.Fatal(s.Err())
	//}
	select {}
}
