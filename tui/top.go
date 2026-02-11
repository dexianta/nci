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
	tabs         []string
	activeTab    int
	width        int
	height       int
	now          time.Time
	selectedRepo string

	repoModel     repoModel
	branchModel   branchModel
	settingsModel settingsModel
}

type tickMsg time.Time

func newModel(svc core.SvcImpl, dbRepo core.DbRepo) topModel {
	return topModel{
		tabs: []string{
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
	var handled bool
	var cmd tea.Cmd

	// special state msgs
	switch mg := msg.(type) {
	case addRepoMsg, deleteRepoMsg, loadRepoMsg, loadBranchConfMsg:
		var cmd1, cmd2 tea.Cmd
		m.repoModel, cmd1 = m.repoModel.Update(msg)
		m.branchModel, cmd2, handled = m.branchModel.Update(msg)
		return m, tea.Batch(cmd1, cmd2)
	case selectRepoMsg:
		var cmd1 tea.Cmd
		m.selectedRepo = mg.repo
		m.branchModel, cmd1, handled = m.branchModel.Update(msg)
		return m, cmd1
	}

	// the result of the key stuff
	switch m.selectedRepo {
	case "":
		// top
		m.repoModel, cmd = m.repoModel.Update(msg)

	default:
		// entered a project
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch m.activeTab {
			case 0:
				m.branchModel, cmd, handled = m.branchModel.Update(msg)

			case 2:
				m.settingsModel, handled = m.settingsModel.Update(msg)
			}

			switch msg.String() {
			case "left":
				if !handled {
					if m.selectedRepo != "" {
						if m.activeTab > 0 {
							m.activeTab--
						}
					}
					return m, nil
				}
			case "right":
				if !handled {
					if m.selectedRepo != "" {
						if m.activeTab < len(m.tabs)-1 {
							m.activeTab++
						}
					}
					return m, nil
				}
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if !handled {
				return m, tea.Quit
			}

		case "esc":
			if !handled {
				m.selectedRepo = ""
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()
	}

	return m, cmd
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

var globalFooter = footerBarStyle.Render(
	renderHint("esc", "top"),
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
	proj := m.selectedRepo
	tabs := ""
	if m.selectedRepo != "" {
		tabs = m.renderTabs()
	}
	body := m.renderBody()

	var footer string
	if m.selectedRepo == "" {
		footer = m.topHelp()
	} else {
		switch m.activeTab {
		case 2:
			footer = m.settingsModel.help()
		}
	}

	footer = lipgloss.JoinVertical(lipgloss.Top, footer, "", globalFooter)
	if proj != "" {
		proj = sectionTitleStyle.Render(fmt.Sprint("\n", ">> "+proj, "\n"))
	}
	return appStyle.Render(strings.Join([]string{
		header,
		proj,
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
	if m.selectedRepo == "" {
		return m.repoModel.View()
	}

	switch m.tabs[m.activeTab] {
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
