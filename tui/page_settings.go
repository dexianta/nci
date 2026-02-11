package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsModel struct {
	sectionIdx   int
	sectionFocus bool
	sectionItems []string

	// sections
	globalConf globalConf
	EnvVars    []KV
	sshConfig  []SSHConfig

	globalConfForm form
	varsForm       form
	sshViewer      sshViewer
}

type SSHConfig struct {
}

type KV struct {
	key   string
	val   string
	dtype string // string, number, bool, date
}

type globalConf struct {
	pollInterval      int
	logRetentionDays  int
	maxConcurrentJobs int
	gitTimeoutSec     int
}

func newSettingModel() settingsModel {
	return settingsModel{
		sectionFocus: true,
		sectionItems: []string{"Global", "Env Vars", "SSH"},
		globalConfForm: newForm([]KV{
			{key: "Poll Interval (s)", val: "3", dtype: "int"},
			{key: "Git Timeout (s)", val: "3", dtype: "int"},
			{key: "Log Retention Days", val: "3", dtype: "int"},
		}, 14, false),
		varsForm:  newForm([]KV{}, 20, true),
		sshViewer: newSSHViewer(),
	}
}

func (s settingsModel) Update(msg tea.KeyMsg) (settingsModel, bool) {
	var handled bool
	switch s.sectionFocus {
	case true:
		// focus on section
		switch msg.String() {
		case "up":
			s.sectionIdx = modIdx(s.sectionIdx, len(s.sectionItems), -1)
			handled = true
		case "down":
			s.sectionIdx = modIdx(s.sectionIdx, len(s.sectionItems), 1)
			handled = true
		case "tab":
			s.sectionFocus = false
			handled = true
		}
	case false:
		// focus on editor
		switch msg.String() {
		case "tab":
			if !s.activeForm().IsEditing() {
				s.sectionFocus = true
				handled = true
			}
		}
		switch s.sectionIdx {
		case 0:
			s.globalConfForm, handled = s.globalConfForm.Update(msg)
		case 1:
			s.varsForm, handled = s.varsForm.Update(msg)
		case 2:
			s.sshViewer = s.sshViewer.Update(msg)
		}
	}
	return s, handled
}

func (s settingsModel) View() string {
	editor := ""
	switch s.sectionIdx {
	case 0:
		editor = s.globalConfForm.View()
	case 1:
		editor = s.varsForm.View()
	case 2:
		editor = s.sshViewer.View()
	}

	section := s.renderSection()
	if !s.sectionFocus {
		editor = regionFocusedStyle.Render(editor)
		section = regionStyle.Render(section)
	} else {
		editor = regionStyle.Render(editor)
		section = regionFocusedStyle.Render(section)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, section, "   ", editor)
}

func (s settingsModel) renderSection() string {
	var text = []string{sectionTitleStyle.Render("Sections"), ""}
	for i, t := range s.sectionItems {
		if i == s.sectionIdx {
			text = append(text, itemSelectedStyle.Render(t))
		} else {
			text = append(text, itemStyle.Background(lipgloss.Color("236")).Render(t))
		}
	}
	list := lipgloss.NewStyle().Width(16).Render(lipgloss.JoinVertical(lipgloss.Top, text...))
	return list
}

func (s settingsModel) activeForm() form {
	switch s.sectionIdx {
	case 0:
		return s.globalConfForm
	case 1:
		return s.varsForm
	default:
		return s.globalConfForm
	}
}

func (s settingsModel) help() string {
	return footerBarStyle.Render(
		renderHint("up/down", "move "),
		renderHint("tab", "to focus"),
	)
}
