package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/u3paka/oteumi/gazo"
	"github.com/u3paka/oteumi/line"
	"github.com/pkg/errors"
)

type LineService struct {
	*App
	*line.Service
}

func (ap *App) NewLineService(channelSecret, channelToken string) (*LineService, error) {
	c, err := line.NewService(channelSecret, channelToken)
	return &LineService{ap, c}, err
}

func (s *LineService) Callback(w http.ResponseWriter, r *http.Request) {
	events, err := s.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println(err)
		return
	}
	d := line.NewSwitchDemux()
	d.PreText = func(m *linebot.TextMessage, e *linebot.Event) (*linebot.TextMessage, *linebot.Event, error) {
		if len(m.Text) > 10000 {
			_, err := s.ReplyMessage(
				e.ReplyToken,
				linebot.NewTextMessage("too long text"),
			).Do()
			if err != nil {
				fmt.Println(err)
				return m, e, err
			}
			return m, e, errors.New("too long text")
		}
		return m, e, nil
	}
	d.UserText = func(m *linebot.TextMessage, e *linebot.Event) {
		ctx := s.Chat.LoadContext(e.Source.UserID).MSet(map[string]interface{}{
			"session_userid":   e.Source.UserID,
			"session_text":     m.Text,
			"session_sendto":   "",
			"session_msgtype":  "line",
			"session_duration": "0s",
		}).Do(m.Text)
		if ctx.Out == "" {
			return
		}
		_, err := s.ReplyMessage(
			e.ReplyToken,
			linebot.NewTextMessage(ctx.Out),
		).Do()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	d.PreImage = func(m *linebot.ImageMessage, e *linebot.Event) (*linebot.ImageMessage, *linebot.Event, error) {
		return m, e, nil
	}
	d.UserImage = func(m *linebot.ImageMessage, e *linebot.Event) {
		dir, err := filepath.Abs(s.Conf.GetString("dir.pubroot"))
		if err != nil {
			fmt.Print(err)
			return
		}
		dir = filepath.Join(dir, s.Conf.GetString("dir.img"))

		fmt.Print(dir)
		content, err := s.Client.GetMessageContent(m.ID).Do()
		if err != nil {
			fmt.Print(err)
		}
		path := filepath.Join(dir, "line", e.Source.UserID, m.ID+".jpg")
		//tmppath := filepath.Join(dir, relpath)
		savepath, err := gazo.SaveBinary(content.Content, path)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash := s.Redis.HGetAll(fmt.Sprintf(keyUid, "LU", e.Source.UserID)).Val()
		//if utype != LineUser{
		//	switch{
		//	case hash["status"] != "waiting":
		//		return nil
		//	}
		//}
		fmt.Println(hash)
		msgs, err := s.ConvertImages(s.Conf.GetString("url.domain"), dir, e.ReplyToken, savepath)

		if err != nil {
			fmt.Println(err)
			return
		}
		ctx := s.Chat.LoadContext(e.Source.UserID).Do("img come in")
		if ctx.Out != "" {
			t := linebot.NewTextMessage(ctx.Out)
			msgs = append([]linebot.Message{t}, msgs...)
		}
		_, err2 := s.ReplyMessage(
			e.ReplyToken,
			msgs...,
		).Do()
		if err2 != nil {
			fmt.Println(err2)
			return
		}

		return
	}
	d.HandleEvents(events)
	w.WriteHeader(http.StatusOK)
	return
}

func (s *LineService) replySession(sess *Session) error {
	fmt.Println(sess)
	switch {
	// case len(sess.ImgFiles) > 0:
	// 	msgs, err := s.ConvertImages(s.Conf.GetString("url.domain"), s.Conf.GetString("dir.pubroot"), sess.Token, sess.ImgFiles...)

	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return err
	// 	}
	// 	ctx := s.Chat.NewContext(sess.).Do("img come in")
	// 	t := linebot.NewTextMessage(ctx.Out)
	// 	msgs = append([]linebot.Message{t}, msgs...)
	// 	_, err2 := s.ReplyMessage(
	// 		sess.Token,
	// 		msgs[:int(math.Min(float64(len(msgs)), 4.0))]...,
	// 	).Do()
	// 	if err2 != nil {
	// 		fmt.Println(err2)
	// 		return err2
	// 	}
	case sess.Text != "":
		_, err := s.ReplyMessage(
			sess.Token,
			linebot.NewTextMessage(sess.Text),
		).Do()
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}
