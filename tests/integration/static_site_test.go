package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"chess_clone/internal/config"
	httpapi "chess_clone/internal/transport/http"
)

func TestHTTPServerServesBuiltReactEntryAndAssets(t *testing.T) {
	webDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<main>Online Chess</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte("window.chess = true"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := httpapi.NewHandler(config.Config{WebDir: webDir})
	server := httptest.NewServer(handler)
	defer server.Close()

	for path, want := range map[string]string{
		"/":              "<main>Online Chess</main>",
		"/join":          "<main>Online Chess</main>",
		"/assets/app.js": "window.chess = true",
	} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if response.StatusCode != http.StatusOK || string(body) != want {
			t.Fatalf("GET %s status=%d body=%q want=%q", path, response.StatusCode, body, want)
		}
	}
}
