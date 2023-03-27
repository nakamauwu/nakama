package nakama

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/galdor/go-thumbhash"
	"github.com/nicolasparada/go-errs"
)

const (
	ErrUnsupportedMedia       = errs.InvalidArgumentError("unsupported media")
	ErrMediaTooLarge          = errs.InvalidArgumentError("media too large")
	ErrUnsupportedImageFormat = errs.InvalidArgumentError("unsupported image format")
)

const maxMediaSize = 16 << 20 // 16MiB

type Media struct {
	Kind    MediaKind `json:"kind"`
	AsImage *Image    `json:"asImage,omitempty"`
}

type MediaKind string

const (
	MediaKindImage MediaKind = "image"
)

var AllMediaKinds = []MediaKind{
	MediaKindImage,
}

func (m MediaKind) String() string {
	return string(m)
}

func (m MediaKind) OK() bool {
	for _, kind := range AllMediaKinds {
		if m == kind {
			return true
		}
	}
	return false
}

func (m Media) Validate() error {
	if !m.Kind.OK() {
		return ErrUnsupportedMedia
	}

	return nil
}

func (m Media) IsImage() bool {
	return m.Kind == MediaKindImage && m.AsImage != nil
}

type Image struct {
	io.ReadSeeker `json:"-"`
	Path          string `json:"path"`
	Width         uint   `json:"width"`
	Height        uint   `json:"height"`
	ThumbHash     []byte `json:"thumbHash"`
	AltText       string `json:"altText,omitempty"`

	byteSize    uint64 `json:"-"`
	contentType string `json:"-"`
}

func ParseMedia(name string, r io.ReadSeeker) (Media, error) {
	var out Media

	ct, err := detectContentType(r)
	if err != nil {
		return out, err
	}

	switch {
	case strings.HasPrefix(ct, "image/"):
		img, err := parseImage(r)
		if err != nil {
			return out, err
		}

		img.contentType = ct
		img.AltText = name
		out.Kind = MediaKindImage
		out.AsImage = &img

		return out, nil
	default:
		return out, ErrUnsupportedMedia
	}
}

func parseImage(r io.ReadSeeker) (Image, error) {
	var out Image

	var buff bytes.Buffer
	_, err := io.Copy(&buff, r)
	if err != nil {
		return out, fmt.Errorf("decode image: copy to buffer: %w", err)
	}

	byteSize := uint64(buff.Len())
	if byteSize > maxMediaSize {
		return out, ErrMediaTooLarge
	}

	img, format, err := image.Decode(&buff)
	if errors.Is(err, image.ErrFormat) {
		return out, ErrUnsupportedImageFormat
	}

	if err != nil {
		return out, fmt.Errorf("decode image: decode: %w", err)
	}

	// Reset the reader so it can be used again.
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return out, fmt.Errorf("decode image: seek to start: %w", err)
	}

	bounds := img.Bounds()
	now := time.Now().UTC()
	out.ReadSeeker = r
	out.Path = fmt.Sprintf("/%d/%d/%d/%s.%s", now.Year(), now.Month(), now.Day(), genID(), format)
	out.Width = uint(bounds.Dx())
	out.Height = uint(bounds.Dy())
	out.ThumbHash = thumbhash.EncodeImage(img)
	out.byteSize = byteSize

	return out, nil
}

func detectContentType(r io.ReadSeeker) (string, error) {
	// http.DetectContentType uses at most 512 bytes to make its decision.
	h := make([]byte, 512)
	_, err := r.Read(h)
	if err != nil {
		return "", fmt.Errorf("detect content type: read head: %w", err)
	}

	// Reset the reader so it can be used again.
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("detect content type: seek to start: %w", err)
	}

	mt, _, err := mime.ParseMediaType(http.DetectContentType(h))
	if err != nil {
		return "", fmt.Errorf("detect content type: %w", err)
	}

	return mt, nil
}

// Resize with the specified dimensions
// to achieve the correct aspect ratio without stretching.
// The final data is a JPEG encoded image.
func (i *Image) Resize(w, h uint) error {
	img, err := imaging.Decode(i, imaging.AutoOrientation(true))
	if errors.Is(err, image.ErrFormat) {
		return ErrUnsupportedImageFormat
	}

	if err != nil {
		return fmt.Errorf("resize image: decode: %w", err)
	}

	// Reset the reader so it can be used again.
	_, err = i.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("resize image: seek to start: %w", err)
	}

	resized := imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)

	var buff bytes.Buffer
	err = imaging.Encode(&buff, resized, imaging.JPEG)
	if err != nil {
		return fmt.Errorf("resize image: jpeg encode: %w", err)
	}

	if ext := filepath.Ext(i.Path); ext != ".jpeg" {
		i.Path = strings.TrimSuffix(i.Path, ext) + ".jpeg"
	}

	i.ReadSeeker = bytes.NewReader(buff.Bytes())
	i.Width = w
	i.Height = h
	i.ThumbHash = thumbhash.EncodeImage(resized)
	i.byteSize = uint64(buff.Len())
	i.contentType = "image/jpeg"

	return nil
}
