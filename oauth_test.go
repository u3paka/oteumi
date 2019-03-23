package main

import (
	"log"
	"net/http"
	"testing"
)

func TestOauth(t *testing.T) {
	ap := NewApp()
	ap.RedisCon(ap.Conf.GetString("address.redis"))

	http.HandleFunc("/login/", applicationHandler(ap.LoginByTwitter))
	http.HandleFunc("/callback/", applicationHandler(ap.TwitterCallback))
	log.Fatal(http.ListenAndServe(":8089", nil))
}
