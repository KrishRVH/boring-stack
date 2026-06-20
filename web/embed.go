package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets
var embedded embed.FS

func Assets() http.FileSystem {
	sub, err := fs.Sub(embedded, "assets")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
