package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderHint(key, desc string) string {
	return keycapStyle.Render(key) + " " + desc
}

func modIdx(idx, mod, delta int) int {
	if mod == 0 {
		return 0
	}
	if delta > 0 {
		delta = 1
	}
	if delta < 0 {
		delta = -1
	}

	if idx+delta >= mod {
		return (idx + delta) % mod
	}

	if idx+delta < 0 {
		//
		return mod + delta
	}
	return idx + delta
}

func renderLabelInputRow(label, value string, focused bool, width int) string {
	display := value
	isViewport := strings.Contains(display, "|")
	isPlaceholder := strings.TrimSpace(display) == ""
	if strings.TrimSpace(display) == "" {
		display = "not set"
	}

	prefix := "  "
	v := valueStyle.Width(width)

	if focused {
		prefix = "▶ "
		v = valueFocus.Width(width)
	}

	contentWidth := width - v.GetHorizontalFrameSize()
	if contentWidth < 1 {
		contentWidth = 1
	}
	if !isViewport {
		display = truncateRunes(display, contentWidth)
	}
	if isPlaceholder {
		display = placeholderStyle.Render(display)
	}

	left := labelStyle.Render(label + ":")
	right := v.Render(display)

	return prefix + lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}
