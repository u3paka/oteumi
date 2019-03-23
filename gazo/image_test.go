package gazo

import (
	"fmt"
	"testing"
)

func TestJpegService_Save(t *testing.T) {
	img := "test_img.jpg"
	ppa := "save_place.jpg"
	imp := new(ImageProcessor)
	err := imp.Open(img).SizeDown(240, 240).ToJPEG().SetQuality(80).Save(ppa)
	if err != nil {
		fmt.Println(err)
	}
}
