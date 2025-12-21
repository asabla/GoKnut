package integration

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func templateFromRepoFiles(t *testing.T, relPaths ...string) *template.Template {
	t.Helper()

	funcs := template.FuncMap{
		"formatNumber": func(n int64) string {
			if n < 1000 {
				return fmt.Sprintf("%d", n)
			}
			if n < 1000000 {
				return fmt.Sprintf("%.1fK", float64(n)/1000)
			}
			return fmt.Sprintf("%.1fM", float64(n)/1000000)
		},
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("Jan 2, 2006")
		},
		"formatTime": func(t *time.Time) string {
			if t == nil || t.IsZero() {
				return "Never"
			}
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				m[key] = values[i+1]
			}
			return m
		},
	}

	// Test binary CWD is the package directory, so derive repo root
	// from this file location.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	tmpl := template.New("").Funcs(funcs)
	for _, relPath := range relPaths {
		absPath := filepath.Join(repoRoot, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			t.Fatalf("failed to read template %s: %v", absPath, err)
		}
		if _, err := tmpl.Parse(string(content)); err != nil {
			t.Fatalf("failed to parse template %s: %v", absPath, err)
		}
	}
	return tmpl
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return string(b)
}
