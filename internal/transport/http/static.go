package httpapi

import (
	"net/http"
	"path/filepath"
)

func staticHandler(webDir string) http.Handler {
	if webDir == "" {
		webDir = "web/dist"
	}
	files := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || filepath.Ext(r.URL.Path) == "" {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}
