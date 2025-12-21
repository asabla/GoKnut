package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/tests/integration/fakes"
)

func TestHomeDashboardDiagramsFragment_Success(t *testing.T) {
	logger := observability.NewLogger("test")

	prom := fakes.NewPrometheusFake()
	prom.SetQueryRangeResponseForQuery("goknut_db_total_messages", fakes.PromQueryRangeResponse{
		Status: "success",
		Data: fakes.PromQueryRangeData{
			ResultType: "matrix",
			Result: []fakes.PromQueryRangeResult{
				{
					Metric: map[string]string{"__name__": "goknut_db_total_messages"},
					Values: [][]any{{float64(1730000000), "1"}, {float64(1730000030), "2"}},
				},
			},
		},
	})
	prom.SetQueryRangeResponseForQuery("goknut_db_total_users", fakes.PromQueryRangeResponse{
		Status: "success",
		Data: fakes.PromQueryRangeData{
			ResultType: "matrix",
			Result: []fakes.PromQueryRangeResult{
				{
					Metric: map[string]string{"__name__": "goknut_db_total_users"},
					Values: [][]any{{float64(1730000000), "1"}, {float64(1730000030), "1"}},
				},
			},
		},
	})
	promSrv := prom.Server()
	defer promSrv.Close()

	mux := http.NewServeMux()
	templates := templateFromRepoFiles(t,
		"internal/http/templates/dashboard/home_diagrams.html",
	)

	h := handlers.NewHomeDashboardHandler(templates, logger, nil, nil, nil, promSrv.URL, 50*time.Millisecond)
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/dashboard/home/diagrams")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)

	if !strings.Contains(body, "data-testid=\"dashboard-diagrams\"") {
		t.Fatalf("expected diagrams marker, got body: %s", body)
	}

	if !strings.Contains(body, "<svg") {
		t.Fatalf("expected SVG diagram output")
	}
	if !strings.Contains(body, "Number of messages") {
		t.Fatalf("expected messages diagram label")
	}
	if !strings.Contains(body, "Number of users") {
		t.Fatalf("expected users diagram label")
	}
}

func TestHomeDashboardDiagramsFragment_PrometheusTimeoutDegraded(t *testing.T) {
	logger := observability.NewLogger("test")

	prom := fakes.NewPrometheusFake()
	prom.BlockQueryRange()
	promSrv := prom.Server()
	defer promSrv.Close()
	defer prom.UnblockQueryRange()

	mux := http.NewServeMux()
	templates := templateFromRepoFiles(t,
		"internal/http/templates/dashboard/home_diagrams.html",
	)

	h := handlers.NewHomeDashboardHandler(templates, logger, nil, nil, nil, promSrv.URL, 5*time.Millisecond)
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/dashboard/home/diagrams")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)

	// Failing-first: implementation should keep layout stable and show degraded indicator.
	if !strings.Contains(body, "data-testid=\"dashboard-diagrams\"") {
		t.Fatalf("expected diagrams marker")
	}
	if !strings.Contains(body, "data-testid=\"dashboard-diagrams-degraded\"") {
		t.Fatalf("expected degraded diagrams indicator")
	}
}
