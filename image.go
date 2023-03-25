package nakama

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/disintegration/imaging"
	"github.com/nicolasparada/go-errs"
)

const ErrUnsupportedImageFormat = errs.InvalidArgumentError("unsupported image format")

type Image struct {
	Path      string `json:"path"`
	Width     uint   `json:"width"`
	Height    uint   `json:"height"`
	ThumbHash []byte `json:"thumbHash"`
}

// fillJPEG image with the specified dimensions
// to achieve the correct aspect ratio without stretching.
func fillJPEG(r io.Reader, w, h uint) ([]byte, error) {
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if errors.Is(err, image.ErrFormat) {
		return nil, ErrUnsupportedImageFormat
	}

	if err != nil {
		return nil, fmt.Errorf("fill image: decode: %w", err)
	}

	resized := imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)

	var out bytes.Buffer
	err = imaging.Encode(&out, resized, imaging.JPEG)
	if err != nil {
		return nil, fmt.Errorf("fill image: jpeg encode: %w", err)
	}

	return out.Bytes(), nil
}
