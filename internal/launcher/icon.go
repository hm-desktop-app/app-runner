package launcher

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const iconPrefix = "/app-icons/"
const maxIconSize = 4 << 20

func iconFile(directory string) (string, os.FileInfo, error) {
	path := filepath.Join(directory, "app.ico")
	info, err := os.Lstat(path)
	if err != nil {
		return path, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 22 || info.Size() > maxIconSize {
		return path, nil, fmt.Errorf("invalid icon file")
	}
	return path, info, nil
}

func refreshIcon(a *App) {
	a.IconURL = ""
	if _, info, err := iconFile(a.Directory); err == nil {
		a.IconURL = fmt.Sprintf("%s%s?v=%d-%d", iconPrefix, a.UUID, info.ModTime().UnixNano(), info.Size())
	}
}

// Only registered application directories can supply icons; other assets go to Wails.
func (s *Store) IconHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, iconPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, iconPrefix)
		a, err := s.Get(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path, info, err := iconFile(a.Directory)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		actual, err := file.Stat()
		if err != nil || !os.SameFile(info, actual) {
			http.NotFound(w, r)
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxIconSize+1))
		if err != nil || len(data) < 22 || len(data) > maxIconSize || !bytes.Equal(data[:4], []byte{0, 0, 1, 0}) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-cache")
		w.Header().Set("ETag", fmt.Sprintf("\"%x\"", sha256.Sum256(data)))
		http.ServeContent(w, r, "app.ico", actual.ModTime(), bytes.NewReader(data))
	})
}
