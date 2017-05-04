//go:generate go-bindata -pkg static -ignore .DS_Store -ignore src/ -ignore .../*.go -o files.go ./...

package static

import (
	"log"
	"net/http"
	"os"

	"github.com/elazarl/go-bindata-assetfs"
)

// all static/ files embedded as a Go library
func Handler() http.Handler {
	var h http.Handler
	if info, err := os.Stat("static/"); err == nil && info.IsDir() {
		log.Printf("Use local static files")
		h = http.FileServer(http.Dir("static/"))
	} else {
		h = http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo})
	}
	return h
}
