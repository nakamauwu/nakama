package media

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/nakamauwu/nakama/media/crawler"
	"mvdan.cc/xurls/v2"
)

var reURL = xurls.Relaxed()

type Media struct {
	Kind       Kind
	DirectLink DirectLink
	Preview    crawler.Metadata
}

func (m Media) IsDirectLink() bool {
	return m.Kind == KindDirectLink
}

func (m Media) IsPreview() bool {
	return m.Kind == KindPreview
}

type Kind string

const (
	KindDirectLink Kind = "direct_link"
	KindPreview    Kind = "preview"
)

type DirectLink struct {
	Kind   DirectLinkKind
	Source string
}

func (l DirectLink) IsImage() bool {
	return l.Kind == DirectLinkKindImage
}

func (l DirectLink) IsVideo() bool {
	return l.Kind == DirectLinkKindVideo
}

func (l DirectLink) IsAudio() bool {
	return l.Kind == DirectLinkKindAudio
}

type DirectLinkKind string

const (
	DirectLinkKindImage DirectLinkKind = "image"
	DirectLinkKindVideo DirectLinkKind = "video"
	DirectLinkKindAudio DirectLinkKind = "audio"
)

type Extractor struct {
	Cache *lru.Cache
}

func (e *Extractor) Extract(ctx context.Context, s string) []Media {
	var out []Media

	var g sync.WaitGroup

	for _, link := range collectLinks(s) {
		got, ok := e.Cache.Get(link)
		if ok {
			media, ok := got.(Media)
			if ok {
				out = append(out, media)
			}
			continue
		}

		g.Add(1)
		go func(link string) {
			defer g.Done()

			media, ok := e.extract(ctx, link)
			if !ok {
				return
			}

			e.Cache.Add(link, media)
			out = append(out, media)
		}(link)
	}

	g.Wait()

	return out
}

func (e Extractor) extract(ctx context.Context, link string) (Media, bool) {
	var out Media

	ct, err := DetectContentType(ctx, link)
	if err != nil {
		return out, false
	}

	if strings.HasPrefix(ct, "image/") {
		return Media{
			Kind: KindDirectLink,
			DirectLink: DirectLink{
				Kind:   DirectLinkKindImage,
				Source: link,
			},
		}, true
	}

	if strings.HasPrefix(ct, "video/") {
		return Media{
			Kind: KindDirectLink,
			DirectLink: DirectLink{
				Kind:   DirectLinkKindVideo,
				Source: link,
			},
		}, true
	}

	if strings.HasPrefix(ct, "audio/") {
		return Media{
			Kind: KindDirectLink,
			DirectLink: DirectLink{
				Kind:   DirectLinkKindAudio,
				Source: link,
			},
		}, true
	}

	if ct == "text/html" {
		preview, err := crawler.Crawl(ctx, link)
		if err != nil {
			return out, false
		}

		return Media{
			Kind:    KindPreview,
			Preview: preview,
		}, true
	}

	return out, false
}

func DetectContentType(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("User-Agent", "nakama")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 512))
	if err != nil {
		return "", fmt.Errorf("read limited reader: %w", err)
	}

	ct := http.DetectContentType(b)
	if ct == "application/octet-stream" {
		s := resp.Header.Get("Content-Type")
		if mt, _, err := mime.ParseMediaType(s); err == nil {
			return mt, nil
		}
	} else if mt, _, err := mime.ParseMediaType(ct); err == nil {
		return mt, nil
	}

	return ct, nil
}

func collectLinks(s string) []string {
	return reURL.FindAllString(s, -1)
}
