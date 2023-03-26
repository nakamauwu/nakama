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

// resizeImage with the specified dimensions
// to achieve the correct aspect ratio without stretching.
func resizeImage(r io.Reader, w, h uint) ([]byte, error) {
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if errors.Is(err, image.ErrFormat) {
		return nil, ErrUnsupportedImageFormat
	}

	if err != nil {
		return nil, fmt.Errorf("resize image: decode: %w", err)
	}

	resized := imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)

	var out bytes.Buffer
	err = imaging.Encode(&out, resized, imaging.JPEG)
	if err != nil {
		return nil, fmt.Errorf("resize image: jpeg encode: %w", err)
	}

	return out.Bytes(), nil
}
