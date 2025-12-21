package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
)

func TestHomeDashboardSummaryFragment(t *testing.T) {
	logger := observability.NewLogger("test")

	mux := http.NewServeMux()
	templates := templateFromRepoFiles(t,
		"internal/http/templates/dashboard/home_summary.html",
	)
	h := handlers.NewHomeDashboardHandler(templates, logger, "", 0)
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/dashboard/home/summary")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)

	if !strings.Contains(body, "data-testid=\"dashboard-summary\"") {
		t.Fatalf("expected dashboard summary marker, got body: %s", body)
	}

	// Failing-first: once implemented, summary should include KPI labels.
	if !strings.Contains(body, "Messages Archived") {
		t.Fatalf("expected KPI labels in summary fragment")
	}
}
