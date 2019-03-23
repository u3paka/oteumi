package gazo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	_ "image/jpeg" //jpeg module
	_ "image/png"  //png module
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/graphics-go/graphics"
	"github.com/BurntSushi/graphics-go/graphics/interp"
	"github.com/jmcvetta/randutil"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
	"github.com/soniakeys/quant/median"
)

type ImageFrame struct {
	Bottom    float64 `json:"bottom"`
	Right     float64 `json:"right"`
	Width     float64 `json:"width"`
	Left      float64 `json:"left"`
	Height    float64 `json:"height"`
	Top       float64 `json:"top"`
	Thickness float64 `json:"thickness"`
	Scale     float64 `json:"scale"`
}

type ImageProcessor struct {
	Path  string
	Image image.Image
}

type Predict struct {
	Status string     `json:"status"`
	Name   string     `json:"name"`
	Score  float64    `json:"score"`
	Path   string     `json:"path"`
	Frame  ImageFrame `json:"frame"`
}

type VisionResponse struct {
	Status string    `json:"status"`
	Result []Predict `json:"result"`
}

type FrameResponse struct {
	Status string       `json:"status"`
	Result []ImageFrame `json:"result"`
}

type VisionContext struct {
	Tmpl     string
	Predicts []Predict
	Event    string
}

//func VisionAPI(p, uri string)(v VisionResponse, err error){
//	req, err := newRequest("POST", uri, map[string]string{
//		"filepath":p,
//	}, map[string]string{
//		"image":p,
//	})
//	cli := &http.Client{}
//	res, err := cli.Do(req)
//	if err != nil {
//		return
//	}
//	defer res.Body.Close()
//	d := json.NewDecoder(res.Body)
//	//b, err := ioutil.ReadAll(res.Body)
//	//if err != nil{
//	//	return
//	//}
//	//err = json.Unmarshal(b, &v)
//	err = d.Decode(&v)
//	if err != nil {
//		return
//	}
//	return
//}

func (ctx *VisionContext) Template() string {
	//rtmpl := ctx.Event
	funcMap := template.FuncMap{
		"rand": func(cs ...string) string {
			rand.Seed(time.Now().Unix())
			return cs[rand.Intn(len(cs))]
		},
		"join": func(cs ...string) string {
			if cs[0] == "" {
				return ""
			}
			return strings.Join(cs, "")
		},
		"multiplyfloat": func(v, v2 float64) float64 {
			return v * v2
		},
		"formatfloat": func(v float64, prec int) string {
			return strconv.FormatFloat(v, 'f', prec, 64)
		},
	}
	tpl := template.Must(template.New("main").Funcs(funcMap).Parse(ctx.Tmpl))
	var doc bytes.Buffer
	if err := tpl.ExecuteTemplate(&doc, ctx.Event, ctx); err != nil {
		fmt.Println(err)
		return ""
	}
	return strings.TrimSpace(doc.String())
}

func (gm *ImageProcessor) Open(path string) *ImageProcessor {
	gm.Path = path
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	src, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	} else {
		gm.Image = src
	}
	f.Close()
	return gm
}

func (gm *ImageProcessor) Adjust(width, height uint) *ImageProcessor {
	x := float64(gm.Image.Bounds().Dx())
	y := float64(gm.Image.Bounds().Dy())
	switch {
	case x >= y:
		r := float64(width) / x
		gm.Image = resize.Resize(width, uint(y*r), gm.Image, resize.Lanczos3)

	case x < y:
		r := float64(height) / y
		gm.Image = resize.Resize(uint(x*r), height, gm.Image, resize.Lanczos3)
	}
	//gm.Image = resize.Resize(uint(math.Min(float64(width), float64(x))), uint(math.Min(float64(height), float64(gm.Image.Bounds().Dy()))), gm.Image, resize.Lanczos3)
	return gm
}

func (gm *ImageProcessor) SizeDown(width, height uint) *ImageProcessor {
	gm.Image = resize.Resize(uint(math.Min(float64(width), float64(gm.Image.Bounds().Dx()))), uint(math.Min(float64(height), float64(gm.Image.Bounds().Dy()))), gm.Image, resize.Lanczos3)
	return gm
}

type JpegService struct {
	SrcPath string
	Src     image.Image
	DstPath string
	Dst     image.Image
	Quality int
}

func (imp *ImageProcessor) ToJPEG() *JpegService {
	s := new(JpegService)
	s.SrcPath = imp.Path
	s.Src = imp.Image
	s.Quality = 100
	return s
}
func (s *JpegService) SetQuality(q int) *JpegService {
	s.Quality = q
	return s
}

func (s *JpegService) Save(dstpath string) error {
	s.DstPath = dstpath
	f, err := os.Create(s.DstPath)
	defer f.Close()
	if err != nil {
		switch e := err.(type) {
		case *os.PathError:
			d, _ := path.Split(e.Path)
			if err := os.MkdirAll(d, 0777); err != nil {
				return err
			}
			f, err = os.Create(dstpath)
		default:
			log.Fatal(err)
			if f != nil {
				f.Close()
			}
			return err
		}
	}
	opts := &jpeg.Options{Quality: s.Quality}
	return jpeg.Encode(f, s.Src, opts)
}

type GifService struct {
	SrcPath string
	Src     image.Image
	DstPath string
	Dst     *gif.GIF
	bg      string
}

func (imp *ImageProcessor) ToGIF() *GifService {
	s := new(GifService)
	s.Src = imp.Image
	s.SrcPath = imp.Path
	s.Dst = &gif.GIF{
		Image:     []*image.Paletted{},
		Delay:     []int{},
		LoopCount: 0,
	}
	s.bg = "#FFFFFF"
	return s
}

func (s *GifService) BGColor(color string) *GifService {
	s.bg = color
	return s
}

func (s *GifService) Save(dstpath string) error {
	s.DstPath = dstpath
	f, err := os.Create(s.DstPath)
	defer f.Close()
	if err != nil {
		switch e := err.(type) {
		case *os.PathError:
			d, _ := path.Split(e.Path)
			if err := os.MkdirAll(d, 0777); err != nil {
				return err
			}
			f, err = os.Create(dstpath)
		default:
			log.Fatal(err)
			if f != nil {
				f.Close()
			}
			return err
		}
	}
	return gif.EncodeAll(f, s.Dst)
}

func (s *GifService) Rotate(speed float64, delay int, zoom, reverse bool) *GifService {
	c, err := colorful.Hex(s.bg)
	if err != nil {
		log.Fatal(err)
	}
	var base float64 = (math.Pi * 2) * 10 * speed / 360
	if reverse {
		base *= -1
	}
	limit := int(360 / 10 / speed)
	q := median.Quantizer(256)
	p := q.Quantize(make(color.Palette, 0, 256), s.Src)
	for i := 0; i < limit; i++ {
		dst := image.NewPaletted(s.Src.Bounds(), p)
		draw.Draw(dst, s.Src.Bounds(), &image.Uniform{c}, image.ZP, draw.Src)
		err = graphics.Rotate(dst, s.Src, &graphics.RotateOptions{Angle: base * float64(i)})
		if err != nil {
			log.Fatal(err)
		}
		if zoom {
			w, h := float64(s.Src.Bounds().Dx()), float64(s.Src.Bounds().Dy())
			tmp := image.NewPaletted(s.Src.Bounds(), p)
			draw.Draw(tmp, s.Src.Bounds(), &image.Uniform{c}, image.ZP, draw.Src)
			z := float64(0.5 + float64(i)/30.0)
			graphics.I.
				Scale(z, z).
				Translate((w-w*z)/2, (h-h*z)/2).
				Transform(tmp, dst, interp.Bilinear)
			dst = tmp
		}
		s.Dst.Image = append(s.Dst.Image, dst)
		s.Dst.Delay = append(s.Dst.Delay, delay)
	}
	return s
}

func GetRandomImages(d string, cnt int) []string {
	choices := make([]randutil.Choice, 0)
	filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
		switch filepath.Ext(path) {
		case ".jpg", ".jpeg", ".png":
			choices = append(choices, randutil.Choice{Weight: 1, Item: path})
		}
		return nil
	})
	fs := make([]string, cnt)
	if len(choices) == 0 {
		return fs
	}
	for i := 0; i < cnt; i++ {
		result, _ := randutil.WeightedChoice(choices)
		fs[i] = result.Item.(string)
	}
	return fs
}

func ImgPath(url string, dir string, info ...string) string {
	_, fname := path.Split(url)
	return ModExt(fname, dir, info...)
}

func ModExt(fname string, dir string, info ...string) string {
	ext := filepath.Ext(fname)
	switch ext {
	case ".jpeg", ".jpeg:orig", ".jpg:orig":
		fname = strings.Replace(fname, ext, ".jpg", -1)
	case ".png:orig":
		fname = strings.Replace(fname, ext, ".png", -1)
	case "":
		fname += ".tmp.jpg"
	}
	return filepath.Join(dir, filepath.Join(info...), fname)
}

func DownloadImage(url string, dir string, info ...string) (string, error) {
	var savepath string
	response, err := http.Get(url)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return savepath, err
	}
	savepath = ImgPath(url, dir, info...)
	return SaveBinary(response.Body, savepath)
}

func SaveBinary(bin io.ReadCloser, savepath string) (string, error) {
	file, err := os.Create(savepath)
	// ERR handling
	if err != nil {
		switch e := err.(type) {
		case *os.PathError:
			d, _ := path.Split(e.Path)
			if err := os.MkdirAll(d, 0777); err != nil {
				return savepath, err
			}
			file, err = os.Create(savepath)
		default:
			if file != nil {
				file.Close()
			}
			return savepath, err
		}
	}
	fmt.Println(io.Copy(file, bin))
	return filepath.Abs(savepath)
}
