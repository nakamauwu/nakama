package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed template/include/*.tmpl template/*.tmpl
var templateFS embed.FS

func parseTmpl(name string) *template.Template {
	tmpl := template.New(name)
	tmpl = template.Must(tmpl.ParseFS(templateFS, "template/include/*.tmpl"))
	return template.Must(tmpl.ParseFS(templateFS, "template/"+name))
}

func (h *Handler) renderTmpl(w http.ResponseWriter, tmpl *template.Template, data any, statusCode int) {
	var buff bytes.Buffer
	err := tmpl.Execute(&buff, data)
	if err != nil {
		h.Logger.Output(2, fmt.Sprintf("could not render %q: %v\n", tmpl.Name(), err))
		http.Error(w, fmt.Sprintf("could not render %q", tmpl.Name()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err = buff.WriteTo(w)
	if err != nil {
		h.Logger.Output(2, fmt.Sprintf("could not send %q: %v\n", tmpl.Name(), err))
	}
}
