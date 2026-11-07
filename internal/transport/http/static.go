package httpapi

import "net/http"

func staticHandler(webDir string) http.Handler {
	if webDir == "" {
		webDir = "web/dist"
	}
	return http.FileServer(http.Dir(webDir))
}
