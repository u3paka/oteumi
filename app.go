package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"text/template"
	"time"

	redis "gopkg.in/redis.v5"

	"github.com/alfredxing/calc/compute"
	"github.com/fsnotify/fsnotify"
	"github.com/k0kubun/pp"
	"github.com/robfig/cron"
	"github.com/spf13/viper"

	"github.com/u3paka/jumangok/jmg"
	"github.com/u3paka/markov"
	"github.com/u3paka/rinchat"
	"github.com/u3paka/rinchat/srtr"
)

type App struct {
	Redis  *redis.Client
	JmgCli *jmg.Client
	DB     *sql.DB
	Chat   *rinchat.Service
	Cron   *cron.Cron
	Conf   *viper.Viper
}

func NewApp() *App {
	v := viper.New()
	v.SetConfigName("config") // name of config file (without extension)
	v.AddConfigPath(".")      // optionally look for config in the working directory
	v.AddConfigPath("./data")
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("changed setting file:", e.Name)
	})
	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	v.AutomaticEnv()
	pp.Println(v.GetString("address.jumangok"))
	return &App{
		Conf: v,
	}

}

func (ap *App) RedisCon(address string) *App {
	ap.Redis = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return ap
}

func (ap *App) jmgCon() (*App, error) {
	ap.JmgCli = jmg.NewClient(ap.Conf.GetString("address.jumangok"))
	return ap, nil
}

func (ap *App) SQLCon(address string) *App {
	db, err := sql.Open("sqlite3", address)
	if err != nil {
		log.Fatal(err)
		defer db.Close()
	}
	ap.DB = db
	return ap
}

func (ap *App) RinchatCon() (*App, error) {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]bool{})
	gob.Register(jmg.Word{})
	gob.Register(jmg.Meta{})
	gob.Register([]*jmg.Word{})
	gob.Register(srtr.Context{})

	cf := &rinchat.Config{ap.Conf}
	ap.Chat = rinchat.NewService(cf)
	// Replacer
	repmap := make([]string, 0)
	for k, v := range ap.Conf.GetStringMap("filter.in") {
		vv, ok := v.(string)
		if ok {
			repmap = append(repmap, k, vv)
		}
	}
	ri := strings.NewReplacer(repmap...)
	ap.Chat.InFilterFunc = func(ctx *rinchat.Context) {
		ctx.In = ri.Replace(ctx.In)
	}
	// Replacer
	repmapo := make([]string, 0)
	for k, v := range ap.Conf.GetStringMap("filter.out") {
		vv, ok := v.(string)
		if ok {
			repmapo = append(repmapo, k, vv)
		}
	}

	ro := strings.NewReplacer(repmapo...)
	ap.Chat.OutFilterFunc = func(ctx *rinchat.Context) {
		ctx.Out = ro.Replace(ctx.Out)
	}

	ap.Chat.LoadFunc = func(ctx *rinchat.Context) {
		d, err := ap.Redis.Get(fmt.Sprintf("sess:%s", ctx.UID)).Result()
		if err != nil {
			fmt.Println(err)
			return
		}
		dec := gob.NewDecoder(bytes.NewBuffer([]byte(d)))
		v := make(map[string]interface{}, 0)
		dec.Decode(&v)
		ctx.Lock()
		defer ctx.Unlock()
		ctx.Vars = v
		return
	}

	ap.Chat.SaveFunc = func(ctx *rinchat.Context) {
		ctx.FlushWithout("session")
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		ctx.RLock()
		defer ctx.RUnlock()
		err := enc.Encode(ctx.Vars)
		if err != nil {
			fmt.Println(err)
			return
		}
		key := fmt.Sprintf("sess:%s", ctx.UID)
		ap.Redis.Set(key, buf.Bytes(), ap.Conf.GetDuration("ttl.session"))
		return
	}

	ap.Chat.PreservedFuncs = func(ctx *rinchat.Context) template.FuncMap {
		return template.FuncMap{
			"gen": func(ts ...string) string {
				p, okc := ctx.Get("session_chara")
				ps, ok := p.(string)
				if !ok {
					ps = ap.Conf.GetString("default_chara")
					if !okc {
						go ctx.Set("session_chara", ps)
					}
				}
				mc := markov.NewTalkService(ap.DB, ps)
				return mc.TrigramMarkovChain(ts...).ThinkingTime(ap.Conf.GetDuration("limit.thinking")).String()
			},
			"ex": func(ts ...string) string {
				l := make([]string, 0)
				for _, t := range ts {
					res, err := ap.Redis.SMembers(fmt.Sprintf("dialog:%s", t)).Result()
					if err != nil {
						fmt.Println(err)
						continue
					}
					if len(res) == 0 {
						break
					}
					l = append(l, res...)
				}
				res := ap.Chat.GetKnowledge("expression", ts...)
				l = append(l, res...)
				if len(l) == 0 {
					return ""
				}
				rand.Seed(time.Now().Unix())
				return l[rand.Intn(len(l)-1)]
			},
			"sismember": func(k, t string) bool {
				return ap.Redis.SIsMember(k, t).Val()
			},
			"smembers": func(k string) []string {
				return ap.Redis.SMembers(k).Val()
			},
			"srand": func(k string) string {
				return ap.Redis.SRandMember(k).Val()
			},
			"sadd": func(k, t string) string {
				ap.Redis.SAdd(k, t).Val()
				return ""
			},
			"hset": func(k, f, v string) string {
				ap.Redis.HSet(k, f, v).Val()
				return ""
			},
			"srem": func(k, t string) string {
				ap.Redis.SRem(k, t).Val()
				return ""
			},
			"lpush": func(k, v string) string {
				err := ap.Redis.LPush(k, v).Err()
				if err != nil {
					fmt.Println(err)
				}
				return ""
			},

			"scard": func(k string) string {
				cnt64, err := ap.Redis.SCard(k).Result()
				if err != nil {
					fmt.Println(err)
				}
				return strconv.FormatInt(cnt64, 10)
			},
			"calc": func(t string) string {
				res, err := compute.Evaluate(t)
				if err != nil {
					fmt.Println(err)
				}
				return strconv.FormatFloat(res, 'f', 4, 64)
			},
			"rubi": func(w *jmg.Word) string {
				// TODO カタカナに統一するひつようがある
				if w == nil {
					return ""
				}
				switch w.Sound {
				case "":
					return ""
				case w.Surface:
					return w.Surface
				default:
					return w.Surface + "(" + w.Sound + ")"
				}
			},

			"nlp": func(t string) []*jmg.Word {
				ws, err := ap.JmgCli.Jumanpp(context.Background(), t)
				if err != nil {
					fmt.Println(err)
				}
				return ws
			},

			"extpos": func(ws []*jmg.Word, poss ...string) []*jmg.Word {
				return jmg.Extract(ws, func(w *jmg.Word) bool {
					for _, p := range poss {
						if w.Pos == p {
							return true
						}
					}
					return false
				})
			},

			"extdomain": func(ws []*jmg.Word, ds ...string) []*jmg.Word {
				return jmg.Extract(ws, func(w *jmg.Word) bool {
					for _, d := range ds {
						if w.HasDomain(d) {
							return true
						}
					}
					return false
				})
			},

			"srtrctx": func() *srtr.Context {
				const skey = "session_srtr"
				vs := ctx.GetAll()
				f := srtr.LoadContext(vs, skey)
				ctx.Set(skey, f)
				// pp.Println(ctx.Vars)
				return f
			},
			"srtrai": srtr.Ai(ap.Redis),
		}
	}
	return ap, nil
}
