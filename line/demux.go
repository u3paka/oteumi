package line

import (
	"github.com/line/line-bot-sdk-go/linebot"
)

type SwitchDemux struct {
	All        func(message interface{})
	PreText    func(m *linebot.TextMessage, event *linebot.Event) (*linebot.TextMessage, *linebot.Event, error)
	UserText   func(m *linebot.TextMessage, event *linebot.Event)
	GroupText  func(m *linebot.TextMessage, event *linebot.Event)
	RoomText   func(m *linebot.TextMessage, event *linebot.Event)
	PreImage   func(m *linebot.ImageMessage, event *linebot.Event) (*linebot.ImageMessage, *linebot.Event, error)
	UserImage  func(m *linebot.ImageMessage, event *linebot.Event)
	GroupImage func(m *linebot.ImageMessage, event *linebot.Event)
	RoomImage  func(m *linebot.ImageMessage, event *linebot.Event)
	Others     func(message interface{})
}

func NewSwitchDemux() SwitchDemux {
	return SwitchDemux{
		All: func(message interface{}) {},
		PreText: func(m *linebot.TextMessage, event *linebot.Event) (*linebot.TextMessage, *linebot.Event, error) {
			return m, event, nil
		},
		UserText:  func(m *linebot.TextMessage, event *linebot.Event) {},
		GroupText: func(m *linebot.TextMessage, event *linebot.Event) {},
		RoomText:  func(m *linebot.TextMessage, event *linebot.Event) {},
		PreImage: func(m *linebot.ImageMessage, event *linebot.Event) (*linebot.ImageMessage, *linebot.Event, error) {
			return m, event, nil
		},
		UserImage:  func(m *linebot.ImageMessage, event *linebot.Event) {},
		GroupImage: func(m *linebot.ImageMessage, event *linebot.Event) {},
		RoomImage:  func(m *linebot.ImageMessage, event *linebot.Event) {},
		Others:     func(message interface{}) {},
	}
}

func (d SwitchDemux) Handle(event *linebot.Event) {
	d.All(event)
	msg := event.Type
	switch msg {
	case linebot.EventTypeMessage:
		switch message := event.Message.(type) {
		case *linebot.TextMessage:
			var err error
			message, event, err = d.PreText(message, event)
			if err != nil {
				return
			}
			utype := event.Source.Type
			switch utype {
			case "user":
				d.UserText(message, event)
			case "group":
				d.GroupText(message, event)
			case "room":
				d.RoomText(message, event)
			}
		case *linebot.ImageMessage:
			var err error
			message, event, err = d.PreImage(message, event)
			if err != nil {
				return
			}
			utype := event.Source.Type
			switch utype {
			case "user":
				d.UserImage(message, event)
			case "group":
				d.GroupImage(message, event)
			case "room":
				d.RoomImage(message, event)
			}
		default:
			d.Others(msg)
		}
	default:
		d.Others(msg)
	}
}

func (d SwitchDemux) HandleEvents(events []*linebot.Event) {
	for _, event := range events {
		d.Handle(event)
	}
}
