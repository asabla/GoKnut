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

	// Failing-first for US1: once implemented, home.html includes HTMX containers
	// that poll these fragments.
	if !strings.Contains(html, "hx-get=\"/dashboard/home/summary\"") {
		t.Fatalf("expected home template to poll summary fragment")
	}
	if !strings.Contains(html, "hx-get=\"/dashboard/home/diagrams\"") {
		t.Fatalf("expected home template to poll diagrams fragment")
	}
}
