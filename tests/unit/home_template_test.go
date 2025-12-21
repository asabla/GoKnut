package unit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHomeTemplateIncludesDashboardPollingContainers(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	b, err := os.ReadFile(filepath.Join(repoRoot, "internal/http/templates/home.html"))
	if err != nil {
		t.Fatalf("failed to read home template: %v", err)
	}

	html := string(b)

	// US1: home.html includes HTMX containers that poll these fragments at a 60s interval.
	if !strings.Contains(html, "hx-get=\"/dashboard/home/summary\"") {
		t.Fatalf("expected home template to poll summary fragment")
	}
	if !strings.Contains(html, "hx-get=\"/dashboard/home/diagrams\"") {
		t.Fatalf("expected home template to poll diagrams fragment")
	}
	if !strings.Contains(html, "every 60s") {
		t.Fatalf("expected home template to poll every 60s")
	}
}

func TestHomeTemplateIncludesShortcutLinks(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	b, err := os.ReadFile(filepath.Join(repoRoot, "internal/http/templates/home.html"))
	if err != nil {
		t.Fatalf("failed to read home template: %v", err)
	}

	html := string(b)

	// US2: shortcuts should remain visible regardless of dashboard loading.
	if !strings.Contains(html, "href=\"/channels\"") {
		t.Fatalf("expected /channels shortcut link")
	}
	if !strings.Contains(html, "href=\"/users\"") {
		t.Fatalf("expected /users shortcut link")
	}
	if !strings.Contains(html, "href=\"/messages\"") {
		t.Fatalf("expected /messages shortcut link")
	}
}

func TestHomeTemplateDoesNotIncludeLiveFeedOrHomeSSE(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	b, err := os.ReadFile(filepath.Join(repoRoot, "internal/http/templates/home.html"))
	if err != nil {
		t.Fatalf("failed to read home template: %v", err)
	}

	html := string(b)

	// US3: start page must not show "Latest Messages" or rely on home SSE.
	if strings.Contains(html, "Latest Messages") {
		t.Fatalf("expected home template to not include Latest Messages")
	}
	if strings.Contains(html, "/live?view=home") {
		t.Fatalf("expected home template to not reference /live?view=home")
	}
}
