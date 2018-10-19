package image

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"godinary/storage"
	bimg "gopkg.in/h2non/bimg.v1"
)

// Image contains image attributes
type Image struct {
	Width       int
	Height      int
	Quality     int
	AspectRatio float32
	Content     *bimg.Image
	RawContent  []byte
	Hash        string
	URL         string
	Format      bimg.ImageType
}

// Load charges content from bytestring
func (img *Image) Load(r io.Reader) {
	body, _ := ioutil.ReadAll(r)
	img.Content = bimg.NewImage(body)
}

// Download retrieves url into io.Reader
func (img *Image) Download() ([]byte, error) {

	c := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   2 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   2 * time.Second,
			ResponseHeaderTimeout: 2 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	if img.URL == "" {
		return nil, fmt.Errorf("sourceURL not found in image")
	}

	resp, err := c.Get(img.URL)
	if err != nil || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cannot download image %s: %v", img.URL, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	img.Content = bimg.NewImage(body)
	return body, err
}

// ExtractInfo stores dimensions into object
func (img *Image) ExtractInfo() error {
	size, err := img.Content.Size()
	if err != nil {
		return fmt.Errorf("can't extract dimensions: %v", err)
	}
	img.Height = size.Height
	img.Width = size.Width
	img.AspectRatio = float32(img.Width) / float32(img.Height)
	return nil
}

// Process resizes and convert image
func (img *Image) Process(source Image, sd storage.Driver) error {
	var err error
	options := bimg.Options{
		Width:   img.Width,
		Height:  img.Height,
		Quality: img.Quality,
		Type:    img.Format,
	}

	if img.RawContent, err = source.Content.Process(options); err != nil {
		return err
	}
	if sd != nil {
		go sd.Write(img.RawContent, img.Hash, "derived/")
	}
	return nil
}
