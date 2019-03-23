package main

import (
	"fmt"
	"time"
)

type Session struct {
	UserId   string
	MsgType  string
	UserType string

	Text   string
	Footer string

	ByName   string
	FromName string
	ToName   string

	Token    string
	ImgFiles []string

	Duration string

	StartTime time.Time
}

func (sess Session) handleText() Session {
	fmt.Println("handle text ____")
	return sess
}
