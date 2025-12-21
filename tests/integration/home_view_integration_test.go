package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
)

func TestHomeView_DoesNotIncludeLiveFeedOrHomeSSE(t *testing.T) {
	mux := http.NewServeMux()
	logger := observability.NewLogger("test")
	templates := templateFromRepoFiles(t,
		"internal/http/templates/partials/base.html",
		"internal/http/templates/home.html",
	)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.ExecuteTemplate(w, "home", struct{}{}); err != nil {
			t.Fatalf("failed to execute template: %v", err)
		}
	})

	// Register dashboard routes; not required for assertions but matches real behavior.
	dashboard := handlers.NewHomeDashboardHandler(templates, logger, nil, nil, nil, "", 0)
	dashboard.RegisterRoutes(mux)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)
	if strings.Contains(body, "Latest Messages") {
		t.Fatalf("expected home page to not include Latest Messages")
	}
	if strings.Contains(body, "/live?view=home") {
		t.Fatalf("expected home page to not reference /live?view=home")
	}
}
