package main

import (
	"net/http"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/manveru/gotour/asset"
	"github.com/skratchdot/open-golang/open"
)

func main() {
	http.Handle("/",
		http.FileServer(
			&assetfs.AssetFS{Asset: asset.Asset, AssetDir: asset.AssetDir, Prefix: "/"}))
	println("Serving at http://127.0.0.1:7879")
	go func() {
		<-time.After(3 * time.Second)
		open.Run("http://localhost:7879")
	}()
	http.ListenAndServe((":7879"), nil)
}
