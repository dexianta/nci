package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	branchesRegionStyle = lipgloss.NewStyle().
				Padding(0, 1).
				BorderStyle(lipgloss.NormalBorder()).
				BorderTop(true).
				BorderLeft(false).
				BorderRight(false).
				BorderBottom(false).
				BorderForeground(lipgloss.Color("240"))

	branchesRegionFocusedStyle = branchesRegionStyle.Copy().
					BorderForeground(lipgloss.Color("45"))

	branchesTitleStyle = sectionTitleStyle.Copy()
)

// renderBranchesPlayground is a static 3-pane demo so you can iterate on layout/styles quickly.
func (m topModel) renderBranchesPlayground() string {
	return renderBranchesThreePaneDemo(m.width, m.height, 1)
}

func renderBranchesThreePaneDemo(totalW, totalH, focusPane int) string {
	if totalW <= 0 || totalH <= 0 {
		return "loading..."
	}

	gap := 1
	bodyH := branchesClamp(totalH-14, 10, 40)
	usableW := branchesClamp(totalW-8, 60, 220)
	contentW := usableW - (gap * 2)

	leftW := contentW * 25 / 100
	midW := contentW * 45 / 100
	rightW := contentW - leftW - midW

	repos := []string{
		"> acme/api",
		"  acme/web",
		"  foo/worker",
		"  dex/nci",
	}
	mappings := []string{
		"> main      -> .nci/main.sh",
		"  staging   -> .nci/staging.sh",
		"  feat-*    -> .nci/feat.sh",
		"  release-* -> .nci/release.sh",
	}
	jobs := []string{
		"PASS  a1b2c3  0.8s  2m ago",
		"FAIL  8f9d10  1.1s  8m ago",
		"PASS  a7c1ef  0.9s  11m ago",
		"PASS  2dd981  0.7s  14m ago",
	}

	left := renderBranchesRegion("Repos", repos, leftW, bodyH, focusPane == 0)
	mid := renderBranchesRegion("Branch Mappings", mappings, midW, bodyH, focusPane == 1)
	right := renderBranchesRegion("Last Jobs", jobs, rightW, bodyH, focusPane == 2)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), mid, strings.Repeat(" ", gap), right)
}

func renderBranchesRegion(title string, lines []string, width, height int, focused bool) string {
	style := branchesRegionStyle
	if focused {
		style = branchesRegionFocusedStyle
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		branchesTitleStyle.Render(title),
		"",
		strings.Join(lines, "\n"),
	)

	return style.Width(width).Height(height).Render(content)
}

func branchesClamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func branchesMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
