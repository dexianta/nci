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

	// reload branch & jobs
	loadBranchCmd = func(dbRepo core.DbRepo, repoName string) func() tea.Msg {
		return func() tea.Msg {
			branches, err := dbRepo.ListBranchConf(repoName)
			return loadBranchConfMsg{
				branchConf: branches,
				err:        err,
			}
		}
	}

	addBranchConfCmd = func(dbRepo core.DbRepo, repo, ref, script string) func() tea.Msg {
		return func() tea.Msg {
			added := core.BranchConf{
				Repo:       repo,
				RefPattern: ref,
				ScriptPath: script,
			}
			err := dbRepo.SaveBranchConf(added)
			return addBranchMsg{
				added: added,
				err:   err,
			}
		}
	}

	delBranchConfCmd = func(dbRepo core.DbRepo, repo, ref string) func() tea.Msg {
		return func() tea.Msg {
			del := core.BranchConf{
				Repo:       repo,
				RefPattern: ref,
			}
			err := dbRepo.DeleteBranchConf(repo, ref)
			return delBranchMsg{
				del: del,
				err: err,
			}
		}
	}
)

type branchModel struct {
	dbRepo             core.DbRepo
	svc                core.SvcImpl
	repo               string
	branchConf         []core.BranchConf
	branchConfForm     form
	jobs               []core.Job
	statusMsg          string
	statusInErr        bool
	activeTab          int
	selectedBranchConf int
	selectedJobs       int
}

func newBranchModel(repo core.DbRepo, svc core.SvcImpl) branchModel {
	return branchModel{
		dbRepo:         repo,
		svc:            svc,
		branchConfForm: newForm([]KV{}, 20, true),
	}
}

func initForm(form form, branchConfs []core.BranchConf) form {
	var kvs []KV
	for _, bc := range branchConfs {
		kvs = append(kvs, KV{
			key:   bc.RefPattern,
			val:   bc.ScriptPath,
			dtype: "string",
		})
	}
	form.kvs = kvs
	return form
}

func (b branchModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, b.renderBranchConf(), "   ", b.renderJobs())
}

func (b branchModel) Update(msg tea.Msg) (branchModel, tea.Cmd, bool) {
	switch m := msg.(type) {
	case addBranchMsg:
		// render new kvs for the form
		b.branchConf = append(b.branchConf, m.added)
		return b, nil, true
	case delBranchMsg:
		// render new kvs for the form
		b.branchConf = core.RemoveBranchConf(b.branchConf, m.del.Repo, m.del.RefPattern)
		return b, nil, true

	case selectRepoMsg: // can be reused for reload
		b.repo = m.repo
		return b, loadBranchCmd(b.dbRepo, b.repo), true

	case loadBranchConfMsg:
		b.branchConf = m.branchConf
		if m.err != nil {
			b.statusMsg = m.err.Error()
			b.statusInErr = true
		}
		b.branchConfForm = initForm(b.branchConfForm, m.branchConf)
		return b, nil, true

	case tea.KeyMsg:
		switch m.String() {
		case "tab": // just move tab
			b.activeTab = modIdx(b.activeTab, 2, 1)
			return b, nil, true
		}

		// section specific
		switch b.activeTab {
		case 0:
			var handled bool
			// each switch need to fetch the job
			b.branchConfForm, handled = b.branchConfForm.Update(m)
			b.selectedBranchConf = b.branchConfForm.idx
			b.selectedJobs = 0
			addKV, delKV := b.branchConfForm.addKV, b.branchConfForm.delKV
			var addCmd, delCmd tea.Cmd
			if addKV != nil {
				addCmd = addBranchConfCmd(b.dbRepo, b.repo, addKV.key, addKV.val)
				b.branchConfForm.addKV = nil
			}
			if delKV != nil {
				delCmd = delBranchConfCmd(b.dbRepo, b.repo, delKV.key)
				b.branchConfForm.delKV = nil
			}
			return b, tea.Batch(addCmd, delCmd), handled

		case 1:
			switch m.String() {
			case "up":
				b.selectedJobs = modIdx(b.selectedJobs, len(b.jobs), 1)
				return b, nil, true
			case "down":
				b.selectedJobs = modIdx(b.selectedJobs, len(b.jobs), 1)
				return b, nil, true
			}
		}
	}
	return b, nil, false
}

func renderRegion(title string, lines []string, helpText string, focused bool) string {
	style := regionStyle
	if focused {
		style = regionFocusedStyle
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionTitleStyle.Render(title),
		"",
		strings.Join(lines, "\n\n"),
		"",
		"",
		mutedStyle.Render(helpText),
	)

	return style.Render(content)
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

	return renderRegion("Branch Conf", []string{b.branchConfForm.View()}, "", b.activeTab == 0)
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
	return renderRegion("Last Jobs", text, "", b.activeTab == 1)
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
