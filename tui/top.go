package tui

import (
	"dexianta/nci/core"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type topModel struct {
	tabs      []string
	activeTab int
	width     int
	height    int
	now       time.Time

	repoModel     repoModel
	branchModel   branchModel
	settingsModel settingsModel
}

type tickMsg time.Time

func newModel(svc core.SvcImpl, dbRepo core.DbRepo) topModel {
	return topModel{
		tabs: []string{
			"Repos",
			"Branches",
			"Logs",
			"Settings",
		},
		now:           time.Now(),
		repoModel:     newRepoModel(svc, dbRepo),
		branchModel:   newBranchModel(dbRepo, svc),
		settingsModel: newSettingModel(),
	}
}

func Run(svc core.SvcImpl, dbRepo core.DbRepo) error {
	p := tea.NewProgram(newModel(svc, dbRepo), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m topModel) Init() tea.Cmd {
	return tea.Sequence(tickCmd(), m.repoModel.Init())
}

func (m topModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle async repo messages even if user switched tabs while IO was in flight.
	switch msg.(type) {
	case addRepoMsg, deleteRepoMsg, loadRepoMsg:
		var cmd1, cmd2 tea.Cmd
		m.repoModel, cmd1 = m.repoModel.Update(msg)
		m.branchModel, cmd2 = m.branchModel.Update(msg)
		return m, tea.Batch(cmd1, cmd2)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "left":
			if m.activeTab > 0 {
				m.activeTab--
			}
			return m, nil
		case "right":
			if m.activeTab < len(m.tabs)-1 {
				m.activeTab++
			}
			return m, nil
		}

		switch m.activeTab {
		case 0:
			var cmd tea.Cmd
			m.repoModel, cmd = m.repoModel.Update(msg)
			return m, cmd
		case 1:
			var cmd tea.Cmd
			m.branchModel, cmd = m.branchModel.Update(msg)
			return m, cmd

		case 3:
			m.settingsModel = m.settingsModel.Update(msg)
			return m, nil
		default:
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()

	default:
		return m, nil
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

var globalFooter = footerBarStyle.Render(
	renderHint("left/right", "switch tab "),
	renderHint("ctrl+c", "quit"),
)

func (m topModel) topHelp() string {
	return footerBarStyle.Render(
		renderHint("tab", "switch section"),
	)
}

func (m topModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	subHeader := mutedStyle.Render(fmt.Sprintf("%s", m.now.Format("2006-01-02 15:04:05 Z07:00")))
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerStyle.Render("nci  -  zero-overhead CI"), " ", subHeader)
	tabs := m.renderTabs()
	body := m.renderBody()

	var footer string
	switch m.activeTab {
	case 0:
		footer = m.topHelp()
	case 3:
		footer = m.settingsModel.help()
	}

	footer = lipgloss.JoinVertical(lipgloss.Top, footer, "", globalFooter)

	return appStyle.Render(strings.Join([]string{
		header,
		"",
		tabs,
		"",
		body,
		"\n\n\n\n\n",
		footer,
	}, "\n"))
}

func (m topModel) renderTabs() string {
	parts := make([]string, 0, len(m.tabs))
	for i, tab := range m.tabs {
		if i == m.activeTab {
			parts = append(parts, tabActiveStyle.Render(" "+tab+" "))
			continue
		}
		parts = append(parts, tabStyle.Render(" "+tab+" "))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m topModel) renderBody() string {
	switch m.tabs[m.activeTab] {
	case "Repos":
		return m.repoModel.View()
	case "Branches":
		return m.branchModel.View()
	//case "Logs":
	//	return "Logs\n\nThis section will stream live job output."
	case "Settings":
		return m.settingsModel.View()
	default:
		return ""
	}
}
