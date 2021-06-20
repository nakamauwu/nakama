package web

import "embed"

//go:embed static/*
var StaticFiles embed.FS

//go:embed template/*
var TemplateFiles embed.FS
