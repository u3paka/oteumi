package twtr

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

func newRequest(method, uri string, forms map[string]string, files map[string]string) (*http.Request, error) {
	var err error
	var file *os.File
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range files {
		file, err = os.Open(v)
		if err != nil {
			continue
		}
		part, err := writer.CreateFormFile(k, v)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, file)
	}

	for key, val := range forms {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}
