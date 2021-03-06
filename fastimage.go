package fastimage

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FastImage instance needs to be initialized before use
type FastImage struct {
	Client *http.Client
}

//DefaultFastImage returns default FastImage client
func DefaultFastImage(timeout int) *FastImage {
	return &FastImage{
		Client: &http.Client{
			Transport: &http.Transport{
				Dial:            (&net.Dialer{Timeout: time.Duration(timeout) * time.Second}).Dial,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

type decoder struct {
	reader io.ReaderAt
}

//Detect image type and size
func (f *FastImage) Detect(uri string) (ImageType, *ImageSize, error) {
	//start := time.Now().UnixNano()
	u, err := url.Parse(uri)
	if err != nil {
		return Unknown, nil, err
	}

	header := make(http.Header)
	header.Set("Referer", u.Scheme+"://"+u.Host)
	header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.87 Safari/537.36")

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Host:       u.Host,
	}

	resp, err2 := f.Client.Do(req)
	if err2 != nil {
		return Unknown, nil, err2
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Unknown, nil, fmt.Errorf(resp.Status)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "image") {
		return Unknown, nil, fmt.Errorf("%v is not image", uri)
	}

	d := &decoder{
		reader: newReaderAt(resp.Body),
	}

	var t ImageType
	var s *ImageSize
	var e error

	typebuf := make([]byte, 2)
	if _, err := d.reader.ReadAt(typebuf, 0); err != nil {
		return Unknown, nil, err
	}

	switch {
	case string(typebuf) == "BM":
		t = BMP
		s, e = d.getBMPImageSize()
	case bytes.Equal(typebuf, []byte{0x47, 0x49}):
		t = GIF
		s, e = d.getGIFImageSize()
	case bytes.Equal(typebuf, []byte{0xFF, 0xD8}):
		t = JPEG
		s, e = d.getJPEGImageSize()
	case bytes.Equal(typebuf, []byte{0x89, 0x50}):
		t = PNG
		s, e = d.getPNGImageSize()
	case string(typebuf) == "II" || string(typebuf) == "MM":
		t = TIFF
		s, e = d.getTIFFImageSize()
	case string(typebuf) == "RI":
		t = WEBP
		s, e = d.getWEBPImageSize()
	default:
		t = Unknown
		e = fmt.Errorf("Unkown image type[%v]", typebuf)
	}
	//stop := time.Now().UnixNano()
	//if stop-start > 500000000 {
	//	fmt.Printf("[%v]%v\n", stop-start, f.Url)
	//}
	return t, s, e
}

//GetImageSize create a default fastimage instance to detect image type and size
func GetImageSize(url string) (ImageType, *ImageSize, error) {
	return DefaultFastImage(2).Detect(url)
}
