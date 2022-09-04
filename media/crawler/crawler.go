package crawler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type MetaTag struct {
	Property string
	Content  string
}

func (t MetaTag) Empty() bool {
	return t.Property == "" || t.Content == ""
}

func NewMetaTagFromHTML(h *colly.HTMLElement) MetaTag {
	return MetaTag{
		Property: strings.ToLower(strings.TrimSpace(h.Attr("property"))),
		Content:  strings.TrimSpace(h.Attr("content")),
	}
}

type Metadata struct {
	Site        string
	Title       string
	Description string
	Images      []ImageMetadata
}

type ImageMetadata struct {
	Source string
	Alt    string
	Width  uint
	Height uint
}

func Crawl(ctx context.Context, link string) (Metadata, error) {
	var metadata Metadata
	c := colly.NewCollector(
		colly.StdlibContext(ctx),
		// TODO: change and review
		// colly.UserAgent("Nakama"),
		// colly.IgnoreRobotsTxt(),
	)

	var title, description string
	var metaTags []MetaTag

	c.OnHTML(`head title`, func(h *colly.HTMLElement) {
		title = strings.TrimSpace(h.Text)
	})

	c.OnHTML(`head meta[name="description"][content]`, func(h *colly.HTMLElement) {
		description = strings.TrimSpace(h.Attr("content"))
	})

	c.OnHTML("head meta[property][content]", func(h *colly.HTMLElement) {
		if t := NewMetaTagFromHTML(h); !t.Empty() {
			metaTags = append(metaTags, t)
		}
	})

	c.OnScraped(func(r *colly.Response) {
		metadata.Title = title
		metadata.Description = description

		// TODO: support array of images
		var image ImageMetadata

		for _, tag := range metaTags {
			// fmt.Printf("%s=%s\n", tag.Property, tag.Content)

			switch tag.Property {
			case "og:site_name", "twitter:site":
				if metadata.Site == "" {
					metadata.Site = tag.Content
				}
			case "og:title", "twitter:title":
				if metadata.Title == "" {
					metadata.Title = tag.Content
				}
			case "description", "og:description", "twitter:description":
				if metadata.Description == "" {
					metadata.Description = tag.Content
				}
			case "og:image:secure_url", "og:image", "twitter:image:src", "twitter:image":
				if image.Source == "" {
					image.Source = tag.Content
				}
			case "og:image:alt":
				if image.Alt == "" {
					image.Alt = tag.Content
				}
			case "og:image:width":
				v, err := strconv.ParseUint(tag.Content, 10, 32)
				if err == nil && image.Width == 0 {
					image.Width = uint(v)
				}
			case "og:image:height":
				v, err := strconv.ParseUint(tag.Content, 10, 32)
				if err == nil && image.Height == 0 {
					image.Height = uint(v)
				}
			}
		}

		if image.Source != "" {
			metadata.Images = append(metadata.Images, image)
		}
	})

	err := c.Visit(link)
	if err != nil {
		return metadata, fmt.Errorf("scrape: %w", err)
	}

	return metadata, nil
}
