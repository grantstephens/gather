package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var frontendFiles embed.FS

func frontendFS() (fs.FS, error) {
	return fs.Sub(frontendFiles, "frontend/dist")
}
