package handlers

import (
	"fmt"
	"html/template"
	"strings"
)

const (
	dashboardSparklineWidth  = 240
	dashboardSparklineHeight = 48
	dashboardSparklinePad    = 2
)

func RenderSparklineSVG(points []PromPoint) template.HTML {
	if len(points) == 0 {
		return RenderEmptySparklineSVG()
	}

	minV := points[0].Value
	maxV := points[0].Value
	for _, p := range points[1:] {
		if p.Value < minV {
			minV = p.Value
		}
		if p.Value > maxV {
			maxV = p.Value
		}
	}
	if maxV == minV {
		maxV = minV + 1
	}

	scaleX := float64(dashboardSparklineWidth-2*dashboardSparklinePad) / float64(max(1, len(points)-1))
	scaleY := float64(dashboardSparklineHeight-2*dashboardSparklinePad) / (maxV - minV)

	path := strings.Builder{}
	for i, p := range points {
		x := float64(dashboardSparklinePad) + float64(i)*scaleX
		y := float64(dashboardSparklineHeight-dashboardSparklinePad) - (p.Value-minV)*scaleY
		if i == 0 {
			path.WriteString(fmt.Sprintf("M%.2f %.2f", x, y))
			continue
		}
		path.WriteString(fmt.Sprintf(" L%.2f %.2f", x, y))
	}

	svg := fmt.Sprintf(
		"<svg width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" fill=\"none\" xmlns=\"http://www.w3.org/2000/svg\"><path d=\"%s\" stroke=\"#9146FF\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"/></svg>",
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		path.String(),
	)
	return template.HTML(svg)
}

func RenderEmptySparklineSVG() template.HTML {
	return template.HTML(fmt.Sprintf(
		"<svg width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" fill=\"none\" xmlns=\"http://www.w3.org/2000/svg\"><rect x=\"1\" y=\"1\" width=\"%d\" height=\"%d\" rx=\"6\" stroke=\"#2f2f35\" stroke-width=\"1\"/><path d=\"M%d %d H%d\" stroke=\"#2f2f35\" stroke-width=\"1\"/></svg>",
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		dashboardSparklineWidth-2,
		dashboardSparklineHeight-2,
		dashboardSparklinePad,
		dashboardSparklineHeight/2,
		dashboardSparklineWidth-dashboardSparklinePad,
	))
}

func RenderDegradedSparklineSVG() template.HTML {
	return template.HTML(fmt.Sprintf(
		"<svg width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" fill=\"none\" xmlns=\"http://www.w3.org/2000/svg\"><rect x=\"1\" y=\"1\" width=\"%d\" height=\"%d\" rx=\"6\" stroke=\"#2f2f35\" stroke-width=\"1\"/><path d=\"M%d %d H%d\" stroke=\"#f59e0b\" stroke-width=\"2\" stroke-dasharray=\"4 4\"/></svg>",
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		dashboardSparklineWidth,
		dashboardSparklineHeight,
		dashboardSparklineWidth-2,
		dashboardSparklineHeight-2,
		dashboardSparklinePad,
		dashboardSparklineHeight/2,
		dashboardSparklineWidth-dashboardSparklinePad,
	))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
