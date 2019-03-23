package line

import (
	"fmt"
	"log"
	"math"
	"path/filepath"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/u3paka/oteumi/gazo"
)

type Service struct {
	*linebot.Client
}

func NewService(channelSecret, channelToken string) (*Service, error) {
	c, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatal(err)
		return &Service{c}, err
	}
	return &Service{c}, nil
}

//func (s *Service) replyConfirm(replyToken, ask, Llabel, Lans, Rlabel, Rans string) error {
//	template := linebot.NewConfirmTemplate(
//		ask,
//		linebot.NewMessageTemplateAction(Llabel, Lans),
//		linebot.NewMessageTemplateAction(Rlabel, Rans),
//	)
//	_, err := s.Client.ReplyMessage(
//		replyToken,
//		linebot.NewTemplateMessage("Confirm alt text", template),
//	).Do()
//	return err
//}

func (s *Service) ConvertImages(baseurl, imgdir, replyToken string, imgs ...string) ([]linebot.Message, error) {
	Msgs := make([]linebot.Message, 0)
	var err error
	imgdir, err = filepath.Abs(imgdir)
	if err != nil {
		return Msgs, err
	}

	for _, img := range imgs[:int(math.Min(float64(len(imgs)), 4.0))] {
		RelPath, _ := filepath.Rel(imgdir, img)
		OriginalURL, err := BaseURLPlus(baseurl, "img", RelPath)
		if err != nil {
			log.Fatal(err)
			continue
		}
		PreviewURL := OriginalURL
		PreviewPath := filepath.Join(imgdir, "tmp", filepath.Base(img))
		imp := new(gazo.ImageProcessor)
		err = imp.Open(img).Adjust(240, 240).ToJPEG().SetQuality(80).Save(PreviewPath)
		if err != nil {
			log.Fatal(err)
			continue
		}
		ok := Retry(Exist, PreviewPath)
		if ok {
			PreviewPathRel, _ := filepath.Rel(imgdir, PreviewPath)
			PreviewURL, err = BaseURLPlus(baseurl, "img", PreviewPathRel)
		}
		fmt.Println(baseurl, OriginalURL, PreviewURL)
		Msgs = append(Msgs, linebot.NewImageMessage(OriginalURL, PreviewURL))
	}
	return Msgs, nil
}
