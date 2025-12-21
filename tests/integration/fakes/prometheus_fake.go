// Package fakes provides test fakes for integration testing.
package fakes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// PrometheusFake is a minimal fake of the Prometheus HTTP API.
//
// It supports the endpoints required by the home dashboard diagrams.
// The fake keeps behavior deterministic and controllable from tests.
type PrometheusFake struct {
	mu sync.RWMutex

	queryRangeResponse PromQueryRangeResponse
	queryRangeStatus   int
	queryRangeDelayCh  chan struct{}
}

// NewPrometheusFake creates a new fake Prometheus server.
func NewPrometheusFake() *PrometheusFake {
	return &PrometheusFake{
		queryRangeStatus: http.StatusOK,
		queryRangeResponse: PromQueryRangeResponse{
			Status: "success",
			Data: PromQueryRangeData{
				ResultType: "matrix",
				Result:     []PromQueryRangeResult{},
			},
		},
	}
}

// Server returns an httptest server that hosts the fake.
func (p *PrometheusFake) Server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(p.serveHTTP))
}

func (p *PrometheusFake) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/query_range" {
		p.mu.RLock()
		delayCh := p.queryRangeDelayCh
		status := p.queryRangeStatus
		resp := p.queryRangeResponse
		p.mu.RUnlock()

		if delayCh != nil {
			<-delayCh
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	http.NotFound(w, r)
}

// SetQueryRangeResponse sets the query_range response.
func (p *PrometheusFake) SetQueryRangeResponse(resp PromQueryRangeResponse) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queryRangeResponse = resp
}

// SetQueryRangeStatus sets the HTTP status for query_range.
func (p *PrometheusFake) SetQueryRangeStatus(code int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queryRangeStatus = code
}

// BlockQueryRange makes the query_range endpoint block until UnblockQueryRange is called.
func (p *PrometheusFake) BlockQueryRange() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queryRangeDelayCh = make(chan struct{})
}

// UnblockQueryRange unblocks a previously blocked query_range request.
func (p *PrometheusFake) UnblockQueryRange() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.queryRangeDelayCh != nil {
		close(p.queryRangeDelayCh)
		p.queryRangeDelayCh = nil
	}
}

// Minimal Prometheus query_range response structs.

type PromQueryRangeResponse struct {
	Status string             `json:"status"`
	Data   PromQueryRangeData `json:"data"`
}

type PromQueryRangeData struct {
	ResultType string                 `json:"resultType"`
	Result     []PromQueryRangeResult `json:"result"`
}

type PromQueryRangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}
