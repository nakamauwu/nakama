package web

import "embed"

//go:embed dist/*
var StaticFiles embed.FS

//go:embed template/*
var TemplateFiles embed.FS
