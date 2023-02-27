package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Masterminds/sprig/v3"
	"golang.org/x/exp/slog"
	"mvdan.cc/xurls/v2"
)

//go:embed template/include/*.tmpl template/*.tmpl
var templateFS embed.FS

var tmplFuncs = template.FuncMap{
	"linkify": linkify,
}

var reURL = xurls.Relaxed()

func parsePage(name string) *template.Template {
	tmpl := template.New(name).Funcs(sprig.FuncMap()).Funcs(tmplFuncs)
	tmpl = template.Must(tmpl.ParseFS(templateFS, "template/include/*.tmpl"))
	return template.Must(tmpl.ParseFS(templateFS, "template/"+name))
}

func parseInclude(name string) *template.Template {
	tmpl := template.New(name).Funcs(sprig.FuncMap()).Funcs(tmplFuncs)
	tmpl = template.Must(tmpl.ParseFS(templateFS, "template/include/*.tmpl"))
	return template.Must(tmpl.ParseFS(templateFS, "template/include/"+name))
}

func (h *Handler) render(w http.ResponseWriter, tmpl *template.Template, data any, statusCode int) {
	var buff bytes.Buffer
	err := tmpl.Execute(&buff, data)
	if err != nil {
		h.Logger.Error("render", err, slog.String("tmpl", tmpl.Name()))
		http.Error(w, fmt.Sprintf("could not render %q", tmpl.Name()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err = buff.WriteTo(w)
	if err != nil {
		h.Logger.Error("http write", err, slog.String("tmpl", tmpl.Name()))
	}
}

// linkify transforms URLs in the given text to HTML anchor tags.
func linkify(s string) template.HTML {
	s = template.HTMLEscapeString(s)
	s = reURL.ReplaceAllString(s,
		`<a href="$0" target="_blank" rel="noopener noreferrer">$0</a>`,
	)
	return template.HTML(s)
}
