package template

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var content embed.FS

func Dist() fs.FS {
	sub, err := fs.Sub(content, "build")
	if err != nil {
		panic(err)
	}
	return sub
}
