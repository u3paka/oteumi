package crawler

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/k0kubun/pp"
	_ "github.com/mattn/go-sqlite3"
)

func TestAmznSwitch(t *testing.T) {
	url := "https://www.amazon.co.jp/dp/B01NCXFWIZ/"
	pp.Println(AmznCheck(url))
	// pv, err := strconv.Atoi(pvs)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(pv, 32378-pv)
}

func TestCrawl(t *testing.T) {
	var dbfile string = "./test.db"

	//db, err := sql.Open("sqlite3", ":memory:")
	//os.Remove(dbfile)
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Println("sql opened")
	//
	//_, err = db.Exec( `CREATE TABLE "ss" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" VARCHAR(255), "content" VARCHAR(255), "emotion" VARCHAR(255), "original" VARCHAR(255))` )
	//if err != nil { panic(err) }

	re := regexp.MustCompile(`(?P<name>[\p{Hiragana}\p{Katakana}\p{Han}]+)[「「『\(（](?P<content>.+)[」」』\)）]\s?(?P<emotion>[\p{Hiragana}\p{Katakana}\p{Han}]*)`)
	n1 := re.SubexpNames()
	for i := 975; i < 10000; i++ {
		si := strconv.Itoa(i)
		fn := fmt.Sprintf("./ss/%s.html", si)
		//SaveHtml(fmt.Sprintf("http://www.lovelive-ss.com/?p=%s", si), fn)
		doc, err := ReadHtml(fn)
		if err != nil {
			fmt.Printf("read. Exec error=%s", err)
			return
		}
		rea := re.Copy()
		doc.Find(".t_b").Each(func(_ int, s *goquery.Selection) {
			ls := strings.Split(s.Text(), "  ")
			fmt.Println(ls)
			tx, err := db.Begin()
			if err != nil {
				fmt.Printf("begin. Exec error=%s", err)
				return
			}
			defer tx.Commit()
			for _, l := range ls {
				l := strings.TrimSpace(l)
				mps := rea.FindAllStringSubmatch(l, -1)
				if len(mps) == 0 {
					return
				}
				//変数取得
				for _, mp := range mps {
					mmap := make(map[string]string, len(mps))
					for i, k := range mp {
						mmap[n1[i]] = k
					}
					fmt.Println(mmap)
					if _, err = tx.Exec(`INSERT INTO "ss" ("name", "content", "emotion", "original") VALUES (?, ?, ?, ?) `, mmap["name"], mmap["content"], mmap["emotion"], mmap[""]); err != nil {
						panic(err)
					}
					time.Sleep(time.Millisecond * 5)
				}
			}
		})
	}
}
