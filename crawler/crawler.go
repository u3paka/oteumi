package crawler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func SaveHtml(url, saveto string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		fmt.Print("url scarapping failed", err)
		return
	}
	res, err := doc.Find("body").Html()
	if err != nil {
		fmt.Print("dom get failed", err)
		return
	}
	errw := ioutil.WriteFile(saveto, []byte(res), os.ModePerm)
	if errw != nil {
		fmt.Print("failed to write", err)
		return
	}
	fmt.Println("ok", url, saveto)
	return
}

func ReadHtml(file string) (*goquery.Document, error) {
	fileInfos, _ := ioutil.ReadFile(file)
	stringReader := strings.NewReader(string(fileInfos))
	return goquery.NewDocumentFromReader(stringReader)
}

func AmznCheck(url string) (data map[string]string, err error) {
	data = make(map[string]string, 0)
	const key = "この商品は、Amazon.co.jp が販売、発送します。"
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return
	}
	data["name"] = strings.TrimSpace(doc.Find("#productTitle").Text())
	data["price"] = doc.Find("#priceblock_ourprice").Text()
	data["url"] = url
	mi := doc.Find("#merchant-info").Text()
	if mi == "" {
		err = errors.New("no merchatn-info")
		return
	}
	if !strings.Contains(mi, key) {
		err = errors.New("not official")

	}
	return
}
