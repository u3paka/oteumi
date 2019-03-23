package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli"
)

func main() {
	runtime.GOMAXPROCS(2)
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   "register keys",
			Action: func(clc *cli.Context) error {
				ap := NewApp()
				ap.RedisCon(ap.Conf.GetString("address.redis"))
				var input, bname, cs, ct string
				fmt.Println("What's your bot name?")
				fmt.Scanln(&input)
				fmt.Printf("Hello, %s!\n", input)
				bname = input

				fmt.Println("What's your bot type? [line/twitter]")
				fmt.Scanln(&input)
				if strings.Contains(input, "line") {
					fmt.Println("What's your LINE_CHANNEL_SECRET?")
					fmt.Scanln(&cs)
					fmt.Println("What's your LINE_CHANNEL_TOKEN?")
					fmt.Scanln(&ct)
					ap.Redis.HMSet(fmt.Sprintf("app:%s", bname), map[string]string{
						"line_channel_secret": cs,
						"line_channel_token":  ct,
					})
					ap.Redis.SAdd(apps_t_wait, input)
				}
				if strings.Contains(input, "tw") {
					fmt.Println("What's your TWTR_CONSUMER_SECRET?")
					fmt.Scanln(&cs)
					fmt.Println("What's your TWTR_CONSUMER_KEY?")
					fmt.Scanln(&ct)
					ap.Redis.HMSet("app:twtr", map[string]string{
						"twtr_consumer_secret": cs,
						"twtr_consumer_key":    ct,
					})
				}
				fmt.Printf("OK!!, DONE!!")
				return nil
			},
		},
		{
			Name:    "unlock",
			Aliases: []string{"u"},
			Usage:   "init redis streaming lock",
			Action: func(clc *cli.Context) error {
				ap := NewApp()
				ap.RedisCon(ap.Conf.GetString("address.redis"))
				bs, err := ap.Redis.SMembers(apps_t_streaming).Result()
				if err != nil {
					fmt.Println(err)
					return err
				}
				for _, b := range bs {
					fmt.Println(b)
					err := ap.Redis.SMove(apps_t_streaming, apps_t_wait, b).Err()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}

				bs, err = ap.Redis.SMembers(apps_l_streaming).Result()
				if err != nil {
					fmt.Println(err)
					return err
				}
				for _, b := range bs {
					fmt.Println(b)
					err := ap.Redis.SMove(apps_l_streaming, apps_l_wait, b).Err()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}
				return nil
			},
		},
		{
			Name:    "lock",
			Aliases: []string{"l"},
			Usage:   "init redis streaming lock",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "linebot",
					Value: "",
					Usage: "line botname",
				},
				cli.StringFlag{
					Name:  "twtrbot",
					Value: "",
					Usage: "twitter botname",
				},
			},
			Action: func(clc *cli.Context) error {
				ap := NewApp()
				ap.RedisCon(ap.Conf.GetString("address.redis"))
				for _, b := range strings.Split(clc.String("linebot"), ",") {
					fmt.Println(b)
					err := ap.Redis.SMove(apps_l_wait, apps_l_streaming, b).Err()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}
				for _, b := range strings.Split(clc.String("twtrbot"), ",") {
					fmt.Println(b)
					err := ap.Redis.SMove(apps_t_wait, apps_t_streaming, b).Err()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}
				return nil
			},
		},
		{
			Name:    "serve",
			Aliases: []string{"s"},
			Usage:   "serve http app",
			Action: func(clc *cli.Context) error {
				http.HandleFunc("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("data/public/img"))).ServeHTTP)
				http.HandleFunc("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
				http.HandleFunc("/node_modules/", http.StripPrefix("/node_modules/", http.FileServer(http.Dir("node_modules"))).ServeHTTP)
				t := template.Must(template.ParseGlob("templates/index.html"))
				http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
					t.ExecuteTemplate(w, "test", "")
				})
				http.ListenAndServe(":8089", nil)
				select {}
			},
		},
		{
			Name:    "chat",
			Aliases: []string{"c"},
			Usage:   "chat",
			Action: func(clc *cli.Context) error {
				ap := NewApp()
				ap.RedisCon(ap.Conf.GetString("address.redis"))
				ap.jmgCon()
				ap.SQLCon(ap.Conf.GetString("fpath.markov"))
				ap.RinchatCon()
				ap.Conf.OnConfigChange(func(e fsnotify.Event) {
					fmt.Println("changed setting file:", e.Name)
					ap.RinchatCon()
				})

				s := bufio.NewScanner(os.Stdin)
				fmt.Println("Lets chat now!")
				for s.Scan() {
					tx := s.Text()
					switch tx {
					case "exit":
						os.Exit(1)
					default:
						out := ap.Chat.LoadContext("user").MSet(map[string]interface{}{
							"session_userid":   "userchat",
							"session_text":     tx,
							"session_sendto":   "",
							"session_msgtype":  "line",
							"session_duration": "0s",
						}).Do(tx)
						fmt.Println(">>", out.Out)
					}
				}
				if s.Err() != nil {
					// non-EOF error.
					log.Fatal(s.Err())
				}
				select {}
			},
		},
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "run all components",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "linebot",
					Value: &cli.StringSlice{},
					Usage: "line botname",
				},
				cli.StringSliceFlag{
					Name:  "twtrbot",
					Value: &cli.StringSlice{},
					Usage: "twitter botname",
				},
			},
			Action: func(clc *cli.Context) error {
				ap := NewApp()
				ap.RedisCon(ap.Conf.GetString("address.redis"))
				ap.jmgCon()
				ap.SQLCon(ap.Conf.GetString("fpath.markov"))
				ap.RinchatCon()
				ap.Conf.OnConfigChange(func(e fsnotify.Event) {
					fmt.Println("changed setting file:", e.Name)
					ap.RinchatCon()
				})
				ap.CronFuncs()

				var bs []string
				bs = clc.StringSlice("linebot")
				// line
				if len(bs) == 0 {
					var err error
					bs, err = ap.Redis.SMembers(apps_l_wait).Result()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}
				for _, b := range bs {
					fmt.Println(b + " starts LineService")
					ap.Redis.SMove(apps_l_wait, apps_l_streaming, b).Err()
					l, err := ap.NewLineService(
						ap.Redis.HGet(fmt.Sprintf("app:%s", b), "line_channel_secret").Val(),
						ap.Redis.HGet(fmt.Sprintf("app:%s", b), "line_channel_token").Val(),
					)
					if err != nil {
						log.Fatal(err)
						return err
					}
					http.HandleFunc(fmt.Sprintf("/callback/%s", b), l.Callback)
				}
				dir, err := filepath.Abs(ap.Conf.GetString("dir.pubroot"))
				if err != nil {
					fmt.Print(err)
					return err
				}
				imgdir := filepath.Join(dir, ap.Conf.GetString("dir.img"))
				imgdirurl := fmt.Sprintf("/%s/", ap.Conf.GetString("dir.img"))
				http.HandleFunc(imgdirurl, http.StripPrefix(imgdirurl, http.FileServer(http.Dir(imgdir))).ServeHTTP)
				// http.HandleFunc("/node_modules/", http.StripPrefix("/node_modules/", http.FileServer(http.Dir("node_modules"))).ServeHTTP)
				// t := template.Must(template.ParseGlob("templates/index.html"))
				// http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
				// 	t.ExecuteTemplate(w, "test", "")
				// })
				fmt.Println(ap.Conf.GetString("port"), "HTTP SERVING")
				go http.ListenAndServe(ap.Conf.GetString("port"), nil)

				// twitter
				ts := clc.StringSlice("twtrbot")
				if len(ts) == 0 {
					var err error
					ts, err = ap.Redis.SMembers(apps_t_wait).Result()
					if err != nil {
						fmt.Println(err)
						return err
					}
				}
				for _, b := range ts {
					tws, err := ap.NewTwtrService(b)
					if err != nil {
						fmt.Println(err)
						return err
					}
					fmt.Println(b)
					go tws.Stream(time.Minute * 5)
				}
				select {}
			},
		},
	}
	app.Run(os.Args)
}
