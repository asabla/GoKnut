package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/asabla/goknut/internal/http/handlers"
)

func TestRenderEmptySparklineSVG(t *testing.T) {
	svg := string(handlers.RenderEmptySparklineSVG())
	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected svg tag")
	}
	if !strings.Contains(svg, "<rect") {
		t.Fatalf("expected placeholder rect")
	}
}

func TestRenderDegradedSparklineSVG(t *testing.T) {
	svg := string(handlers.RenderDegradedSparklineSVG())
	if !strings.Contains(svg, "stroke-dasharray") {
		t.Fatalf("expected dashed line for degraded state")
	}
	if !strings.Contains(svg, "#f59e0b") {
		t.Fatalf("expected amber stroke color")
	}
}

func TestRenderSparklineSVG(t *testing.T) {
	points := []handlers.PromPoint{
		{Timestamp: time.Unix(0, 0), Value: 1},
		{Timestamp: time.Unix(30, 0), Value: 2},
		{Timestamp: time.Unix(60, 0), Value: 1.5},
	}

	svg := string(handlers.RenderSparklineSVG(points))
	if !strings.Contains(svg, "<path") {
		t.Fatalf("expected path element")
	}
	if !strings.Contains(svg, "stroke=\"#9146FF\"") {
		t.Fatalf("expected twitch purple stroke")
	}
	if !strings.Contains(svg, "M") || !strings.Contains(svg, "L") {
		t.Fatalf("expected path data with line commands")
	}
}
