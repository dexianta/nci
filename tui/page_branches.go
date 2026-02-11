package tui

import (
	"dexianta/nci/core"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	branchesRegionFocusedStyle = branchesRegionStyle.
					BorderForeground(lipgloss.Color("45"))
)

type branchModel struct {
	dbRepo             core.DbRepo
	svc                core.SvcImpl
	repos              []core.CodeRepo
	branchConf         []core.BranchConf
	jobs               []core.Job
	statusMsg          string
	statusInErr        bool
	activeTab          int
	selectedRepo       int
	selectedBranchConf int
	selectedJobs       int
}

func newBranchModel(repo core.DbRepo, svc core.SvcImpl) branchModel {
	return branchModel{
		dbRepo: repo,
		svc:    svc,
	}
}

func (b branchModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, b.renderRepo(), "   ", b.renderBranchConf(), "   ", b.renderJobs())
}

func (b branchModel) Update(msg tea.Msg) (branchModel, tea.Cmd) {
	switch m := msg.(type) {
	case loadRepoMsg:
		b.repos = m.repos
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}
	case addRepoMsg:
		b.repos = append(b.repos, m.repo)
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}
	case deleteRepoMsg:
		b.repos = m.repos
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}
		if b.selectedRepo >= len(b.repos) {
			b.selectedRepo = len(b.repos) - 1
		}

	case branchesLoadBranchConfMsg:
		b.branchConf = m.branchConf
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}

	case branchesLoadJobMsg:
		b.jobs = m.jobs
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}

	case tea.KeyMsg:
		switch m.String() {
		case "tab": // just move tab
			b.activeTab = modIdx(b.activeTab, 3, 1)
			return b, nil
		}

		// reload branch & jobs
		branchCmd := func(repoIdx int) func() tea.Msg {
			return func() tea.Msg {
				branches, err := b.dbRepo.ListBranchConf(core.SafeIdx(repoIdx, b.repos).Repo)
				return branchesLoadBranchConfMsg{
					branchConf: branches,
					err:        err,
				}
			}
		}
		jobCmd := func(repoIdx, branchIdx int) func() tea.Msg {
			return func() tea.Msg {
				jobs, err := b.dbRepo.ListJob(core.JobFilter{
					Repo: core.SafeIdx(repoIdx, b.repos).Repo,
					Ref:  core.SafeIdx(branchIdx, b.branchConf).RefPattern,
				})
				return branchesLoadJobMsg{
					jobs: jobs,
					err:  err,
				}
			}
		}
		switch b.activeTab {
		case 0:
			switch m.String() {
			case "up":
				b.selectedRepo = modIdx(b.selectedRepo, len(b.repos), -1)
				b.selectedBranchConf = 0
				b.selectedJobs = 0
				return b, tea.Sequence(branchCmd(b.selectedRepo), jobCmd(b.selectedRepo, 0))
			case "down":
				b.selectedRepo = modIdx(b.selectedRepo, len(b.repos), 1)
				b.selectedBranchConf = 0
				b.selectedJobs = 0
				return b, tea.Sequence(branchCmd(b.selectedRepo), jobCmd(b.selectedRepo, 0))
			}

		case 1:
			// each switch need to fetch the job
			switch m.String() {
			case "up":
				b.selectedBranchConf = modIdx(b.selectedBranchConf, len(b.branchConf), -1)
				b.selectedJobs = 0
				return b, jobCmd(b.selectedRepo, b.selectedBranchConf)
			case "down":
				b.selectedBranchConf = modIdx(b.selectedBranchConf, len(b.branchConf), 1)
				b.selectedJobs = 0
				return b, jobCmd(b.selectedRepo, b.selectedBranchConf)
			}
		case 2:
			switch m.String() {
			case "up":
				b.selectedJobs = modIdx(b.selectedJobs, len(b.jobs), 1)
			case "down":
				b.selectedJobs = modIdx(b.selectedJobs, len(b.jobs), 1)
			}
		}
	}
	return b, nil
}

func renderRegion(title string, lines []string, focused bool) string {
	style := branchesRegionStyle
	if focused {
		style = branchesRegionFocusedStyle
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionTitleStyle.Render(title),
		"",
		strings.Join(lines, "\n\n"),
	)

	return style.Render(content)
}

func (b branchModel) renderRepo() string {
	text := []string{}
	for i, r := range b.repos {
		if b.selectedRepo == i {
			text = append(text, "> "+r.Repo)
		} else {
			text = append(text, "  "+r.Repo)
		}
	}
	return renderRegion("Repos", text, b.activeTab == 0)
}

func (b branchModel) renderBranchConf() string {
	text := []string{}
	for i, c := range b.branchConf {
		line := c.RefPattern + " -> " + c.ScriptPath
		if b.selectedBranchConf == i {
			text = append(text, "> "+line)
		} else {
			text = append(text, "  "+line)
		}
	}
	return renderRegion("Branch Conf", text, b.activeTab == 1)
}

func (b branchModel) renderJobs() string {
	text := []string{}
	for i, j := range b.jobs {
		line := jobLine(j, time.Now())
		if b.selectedJobs == i {
			text = append(text, "> "+line)
		} else {
			text = append(text, "  "+line)
		}
	}
	return renderRegion("Last Jobs", text, b.activeTab == 2)
}

func jobLine(j core.Job, now time.Time) string {
	status := strings.ToUpper(strings.TrimSpace(j.Status))
	switch status {
	case "FINISHED":
		status = "PASS"
	case "FAILED":
		status = "FAIL"
	case "RUNNING":
		status = "RUN"
	case "PENDING":
		status = "PEND"
	case "CANCELED":
		status = "CANC"
	}
	if status == "" {
		status = "UNKN"
	}
	if len(status) > 4 {
		status = status[:4]
	}

	shortSHA := strings.TrimSpace(j.SHA)
	if len(shortSHA) > 8 {
		shortSHA = shortSHA[:8]
	}
	if shortSHA == "" {
		shortSHA = "-"
	}

	duration := "--"
	if !j.Start.IsZero() && !j.End.IsZero() && j.End.After(j.Start) {
		duration = j.End.Sub(j.Start).Round(time.Millisecond).String()
	}

	when := "--"
	if !j.End.IsZero() {
		when = timeAgo(now, j.End)
	} else if !j.Start.IsZero() {
		when = timeAgo(now, j.Start)
	}

	return fmt.Sprintf("%-4s  %-8s  %-7s  %s", status, shortSHA, duration, when)
}

func timeAgo(now, t time.Time) string {
	if t.IsZero() {
		return "--"
	}
	d := now.Sub(t)
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// renderBranches is a static 4-pane demo so you can iterate on layout/styles quickly.
func (m topModel) renderBranches() string {
	return renderBranches(m.width, m.height, 1)
}

func renderBranches(totalW, totalH, focusPane int) string {
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
		sectionTitleStyle.Render(title),
		"",
		strings.Join(lines, "\n"),
	)

	//return style.Width(width).Height(height).Render(content)
	return style.Render(content)
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
