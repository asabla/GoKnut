package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
)

// HomeDashboardHandler serves the dashboard fragments embedded on the home page.
//
// In the foundational phase this handler only returns placeholder HTML.
// Later phases add data aggregation and Prometheus queries.
type HomeDashboardHandler struct {
	templates *template.Template
	logger    *observability.Logger

	messageRepo *repository.MessageRepository
	channelRepo *repository.ChannelRepository
	userRepo    *repository.UserRepository

	httpClient *http.Client

	prometheusBaseURL string
	prometheusTimeout time.Duration

	rateLimitMu       sync.Mutex
	lastDegradedLogAt map[string]time.Time
}

func NewHomeDashboardHandler(
	templates *template.Template,
	logger *observability.Logger,
	messageRepo *repository.MessageRepository,
	channelRepo *repository.ChannelRepository,
	userRepo *repository.UserRepository,
	prometheusBaseURL string,
	prometheusTimeout time.Duration,
) *HomeDashboardHandler {
	return &HomeDashboardHandler{
		templates:         templates,
		logger:            logger,
		messageRepo:       messageRepo,
		channelRepo:       channelRepo,
		userRepo:          userRepo,
		httpClient:        &http.Client{},
		prometheusBaseURL: prometheusBaseURL,
		prometheusTimeout: prometheusTimeout,
		lastDegradedLogAt: map[string]time.Time{},
	}
}

func (h *HomeDashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /dashboard/home/summary", h.handleSummary)
	mux.HandleFunc("GET /dashboard/home/diagrams", h.handleDiagrams)
}

func (h *HomeDashboardHandler) logDashboardDegraded(fragment string, errors []string) {
	const minLogInterval = 60 * time.Second

	if h.logger == nil {
		return
	}

	now := time.Now()

	h.rateLimitMu.Lock()
	last := h.lastDegradedLogAt[fragment]
	if !last.IsZero() && now.Sub(last) < minLogInterval {
		h.rateLimitMu.Unlock()
		return
	}
	h.lastDegradedLogAt[fragment] = now
	h.rateLimitMu.Unlock()

	h.logger.Warn("dashboard fragment degraded", "fragment", fragment, "errors", errors)
}

type homeSummaryData struct {
	Snapshot homeKPISnapshot
}

func (h *HomeDashboardHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	snapshot := h.buildKPISnapshot(r.Context())
	data := homeSummaryData{Snapshot: snapshot}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "dashboard/home_summary", data); err != nil {
		h.logger.Error("failed to execute dashboard summary template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	if snapshot.Degraded() {
		h.logDashboardDegraded("summary", snapshot.Errors)
	}
}

func (h *HomeDashboardHandler) handleDiagrams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	window := 15 * time.Minute
	step := 30 * time.Second
	end := time.Now()
	start := end.Add(-window)

	// PromQL per spec.
	messagesQuery := "goknut_db_total_messages"
	usersQuery := "goknut_db_total_users"

	messages, errMessages := h.queryPrometheusRange(ctx, messagesQuery, start, end, step)
	users, errUsers := h.queryPrometheusRange(ctx, usersQuery, start, end, step)

	data := homeDiagramsData{
		WindowLabel: "Last 15m",
		MessagesSVG: RenderSparklineSVG(messages),
		UsersSVG:    RenderSparklineSVG(users),
		Degraded:    errMessages != nil || errUsers != nil,
	}
	if errMessages != nil {
		data.Errors = append(data.Errors, "messages")
		data.MessagesSVG = RenderDegradedSparklineSVG()
	}
	if errUsers != nil {
		data.Errors = append(data.Errors, "users")
		data.UsersSVG = RenderDegradedSparklineSVG()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "dashboard/home_diagrams", data); err != nil {
		h.logger.Error("failed to execute dashboard diagrams template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	if data.Degraded {
		h.logDashboardDegraded("diagrams", data.Errors)
	}
}

type homeDiagramsData struct {
	WindowLabel string
	Degraded    bool
	Errors      []string

	MessagesSVG template.HTML
	UsersSVG    template.HTML
}

type homeKPISnapshot struct {
	TotalMessages   int64
	TotalChannels   int64
	EnabledChannels int64
	TotalUsers      int64

	UpdatedAt time.Time
	Errors    []string
}

func (s homeKPISnapshot) Degraded() bool {
	return len(s.Errors) > 0
}

func (h *HomeDashboardHandler) buildKPISnapshot(ctx context.Context) homeKPISnapshot {
	snapshot := homeKPISnapshot{UpdatedAt: time.Now()}

	if h.messageRepo == nil {
		snapshot.Errors = append(snapshot.Errors, "messages")
	} else {
		count, err := h.messageRepo.GetTotalCount(ctx)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, "messages")
		} else {
			snapshot.TotalMessages = count
		}
	}

	if h.channelRepo == nil {
		snapshot.Errors = append(snapshot.Errors, "channels")
	} else {
		count, err := h.channelRepo.GetCount(ctx)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, "channels")
		} else {
			snapshot.TotalChannels = count
		}

		enabled, err := h.channelRepo.GetEnabledCount(ctx)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, "enabled_channels")
		} else {
			snapshot.EnabledChannels = enabled
		}
	}

	if h.userRepo == nil {
		snapshot.Errors = append(snapshot.Errors, "users")
	} else {
		count, err := h.userRepo.GetCount(ctx)
		if err != nil {
			snapshot.Errors = append(snapshot.Errors, "users")
		} else {
			snapshot.TotalUsers = count
		}
	}

	return snapshot
}

type PromPoint struct {
	Timestamp time.Time
	Value     float64
}

type promQueryRangeResponse struct {
	Status    string             `json:"status"`
	Data      promQueryRangeData `json:"data"`
	ErrorType string             `json:"errorType,omitempty"`
	Error     string             `json:"error,omitempty"`
}

type promQueryRangeData struct {
	ResultType string                 `json:"resultType"`
	Result     []promQueryRangeResult `json:"result"`
}

type promQueryRangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

func (h *HomeDashboardHandler) queryPrometheusRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]PromPoint, error) {
	if strings.TrimSpace(h.prometheusBaseURL) == "" {
		return nil, errors.New("prometheus base url not configured")
	}
	if step <= 0 {
		return nil, errors.New("invalid step")
	}

	baseURL, err := url.Parse(h.prometheusBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus base url: %w", err)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/api/v1/query_range"

	q := baseURL.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", step.String())
	baseURL.RawQuery = q.Encode()

	requestCtx := ctx
	cancel := func() {}
	if h.prometheusTimeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, h.prometheusTimeout)
	}
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read prometheus response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus returned %d", resp.StatusCode)
	}

	var decoded promQueryRangeResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if decoded.Status != "success" {
		if decoded.Error != "" {
			return nil, fmt.Errorf("prometheus error: %s", decoded.Error)
		}
		return nil, fmt.Errorf("prometheus error (%s)", decoded.ErrorType)
	}
	if decoded.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("unexpected prometheus result type: %s", decoded.Data.ResultType)
	}
	if len(decoded.Data.Result) == 0 {
		return []PromPoint{}, nil
	}

	series := decoded.Data.Result[0]
	if len(decoded.Data.Result) > 1 {
		for _, candidate := range decoded.Data.Result {
			if candidate.Metric != nil {
				if name, ok := candidate.Metric["__name__"]; ok && name == query {
					series = candidate
					break
				}
			}
		}
	}

	points := make([]PromPoint, 0, len(series.Values))
	for _, v := range series.Values {
		if len(v) != 2 {
			continue
		}

		ts, ok := v[0].(float64)
		if !ok {
			continue
		}

		strVal, ok := v[1].(string)
		if !ok {
			continue
		}

		f, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			continue
		}

		points = append(points, PromPoint{Timestamp: time.Unix(int64(ts), 0), Value: f})
	}

	return points, nil
}
