package launcher

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func smallIcon(t *testing.T) []byte {
	t.Helper()
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	header := make([]byte, 22)
	header[2], header[4], header[6], header[7], header[10], header[12] = 1, 1, 16, 16, 1, 32
	binary.LittleEndian.PutUint32(header[14:], uint32(pngData.Len()))
	binary.LittleEndian.PutUint32(header[18:], 22)
	return append(header, pngData.Bytes()...)
}

func TestApplicationIconServingAndRefresh(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "with-icon", "title='Icon App'\ntype='GUI'\nentry='app.exe'")
	icon := smallIcon(t)
	path := filepath.Join(dir, "app.ico")
	if err := os.WriteFile(path, icon, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	app := scan(t, store).Apps[0]
	if app.IconURL == "" {
		t.Fatal("icon URL missing")
	}
	handler := store.IconHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(418) }))
	request := httptest.NewRequest("GET", app.IconURL, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || response.Header().Get("Content-Type") != "image/x-icon" || !bytes.Equal(response.Body.Bytes(), icon) {
		t.Fatalf("icon response: %d %v", response.Code, response.Header())
	}
	cached := httptest.NewRecorder()
	request.Header.Set("If-None-Match", response.Header().Get("ETag"))
	handler.ServeHTTP(cached, request)
	if cached.Code != 304 {
		t.Fatalf("expected conditional response, got %d", cached.Code)
	}
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest("HEAD", app.IconURL, nil))
	if head.Code != 200 || head.Body.Len() != 0 {
		t.Fatal("HEAD must not send icon bytes")
	}
	now := time.Now().Add(time.Second)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatal(err)
	}
	refreshed := scan(t, store).Apps[0]
	if refreshed.IconURL == app.IconURL {
		t.Fatal("refresh did not invalidate icon URL")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if scan(t, store).Apps[0].IconURL != "" {
		t.Fatal("removed icon still advertised")
	}
	for _, route := range []string{app.IconURL, "/app-icons/unknown", "/app-icons/../../app.toml"} {
		got := httptest.NewRecorder()
		handler.ServeHTTP(got, httptest.NewRequest("GET", route, nil))
		if got.Code != 404 {
			t.Fatalf("unexpected response for %s: %d", route, got.Code)
		}
	}
	fallback := httptest.NewRecorder()
	handler.ServeHTTP(fallback, httptest.NewRequest("GET", "/index.html", nil))
	if fallback.Code != 418 {
		t.Fatal("normal assets did not reach the next handler")
	}
}

func TestInvalidIconIsOptional(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "bad-icon", "title='Bad Icon'\ntype='GUI'\nentry='app.exe'")
	path := filepath.Join(dir, "app.ico")
	store := NewStore(root)
	if scan(t, store).Apps[0].IconURL != "" {
		t.Fatal("missing icon must be optional")
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), 32), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot := scan(t, store)
	if len(snapshot.Warnings) != 0 {
		t.Fatal("bad icon must not block library loading")
	}
	handler := store.IconHandler(http.NotFoundHandler())
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, httptest.NewRequest("GET", snapshot.Apps[0].IconURL, nil))
	if got.Code != 404 {
		t.Fatalf("invalid ICO served: %d", got.Code)
	}
	if err := os.WriteFile(path, make([]byte, maxIconSize+1), 0600); err != nil {
		t.Fatal(err)
	}
	if scan(t, store).Apps[0].IconURL != "" {
		t.Fatal("oversized icon must not be advertised")
	}
}
