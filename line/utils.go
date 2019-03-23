package line

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"time"
)

func BaseURLPlus(baseURL string, elem ...string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(elem...)
	return u.String(), nil
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func Retry(fn func(string) bool, arg string) bool {
	wait := time.Millisecond
	select {
	case <-time.After(time.Second * 5):
		return true
	default:
		for i := 0; i < 10 && wait < time.Second*3; i++ {
			if fn(arg) {
				return true
			}
			switch {
			case i == 0:
				fmt.Println("start monitoring...")
				fallthrough
			case i < 3:
				wait += time.Millisecond * 100
			default:
				wait += wait
			}
			<-time.After(wait)
		}
	}
	return false
}
