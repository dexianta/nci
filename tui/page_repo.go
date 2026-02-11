package tui

import (
	"dexianta/nci/core"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type repoModel struct {
	svc            core.SvcImpl
	dbRepo         core.DbRepo
	repoInput      string
	repos          []core.CodeRepo
	selectedRepo   int
	projectFocus   projectFocus
	statusMessage  string
	statusIsError  bool
	isCloning      bool
	isLoadingRepo  bool
	isDeletingRepo bool
}

type projectFocus int

const (
	focusInput projectFocus = iota
	focusRepoList
)

func newRepoModel(svc core.SvcImpl, dbRepo core.DbRepo) repoModel {
	return repoModel{
		svc:          svc,
		dbRepo:       dbRepo,
		repos:        []core.CodeRepo{},
		projectFocus: focusInput,
	}
}

func (m repoModel) Init() tea.Cmd {
	return func() tea.Msg {
		repos, err := m.dbRepo.ListCodeRepo()
		return loadRepoMsg{
			repos: repos,
			err:   err,
		}
	}
}

func (m repoModel) Update(msg tea.Msg) (repoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deleteRepoMsg:
		m.isDeletingRepo = false
		if msg.err != nil {
			m.statusMessage = "Failed to delete repo: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.repos = msg.repos
		if m.selectedRepo >= len(m.repos) && m.selectedRepo > 0 {
			m.selectedRepo--
		}
		m.statusMessage = "Deleted: " + msg.deleted
		m.statusIsError = false
		return m, nil

	case addRepoMsg:
		m.isCloning = false
		if msg.err != nil {
			m.statusMessage = "Failed to clone repo: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.repos = append(m.repos, msg.repo)
		m.selectedRepo = len(m.repos) - 1
		m.repoInput = ""
		m.statusMessage = "Repository added."
		m.statusIsError = false
		return m, nil

	case loadRepoMsg:
		m.isLoadingRepo = false
		if msg.err != nil {
			m.statusMessage = "Failed to list repos: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}

		m.repos = msg.repos
		m.selectedRepo = 0
		m.statusIsError = false
		m.statusMessage = "Repos loaded"
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(msg)

	default:
		return m, nil
	}
}

func (m repoModel) updateKey(msg tea.KeyMsg) (repoModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "tab", "shift+tab":
		if m.projectFocus == focusInput {
			m.projectFocus = focusRepoList
		} else {
			m.projectFocus = focusInput
		}
		return m, nil
	case "enter":
		if m.projectFocus == focusInput {
			m, cmd = m.addRepo()
		}
		return m, cmd
	case "backspace":
		if m.projectFocus == focusInput && len(m.repoInput) > 0 {
			runes := []rune(m.repoInput)
			m.repoInput = string(runes[:len(runes)-1])
		}
		return m, nil
	case "up", "k":
		if m.projectFocus == focusRepoList && m.selectedRepo > 0 {
			m.selectedRepo--
		}
		return m, nil
	case "down", "j":
		if m.projectFocus == focusRepoList && m.selectedRepo < len(m.repos)-1 {
			m.selectedRepo++
		}
		return m, nil
	case "d":
		if m.projectFocus == focusRepoList {
			m, cmd = m.deleteRepo()
		}
		return m, cmd
	case "esc":
		if m.projectFocus == focusInput {
			m.repoInput = ""
		}
		m.statusMessage = ""
		m.statusIsError = false
		return m, nil
	}

	if m.projectFocus == focusInput && len(msg.Runes) > 0 {
		m.repoInput += string(msg.Runes)
	}
	return m, nil
}

func (m repoModel) addRepo() (repoModel, tea.Cmd) {
	if m.isCloning {
		m.statusMessage = "Repo cloning in progress"
		m.statusIsError = false
		return m, nil
	}
	raw := strings.TrimSpace(m.repoInput)
	if raw == "" {
		m.statusMessage = "Please enter a repository URL."
		m.statusIsError = true
		return m, nil
	}
	if !isRepoURL(raw) {
		m.statusMessage = "Use a full repo URL (https://... or git@...)."
		m.statusIsError = true
		return m, nil
	}

	repoName := core.ParseGithubUrl(raw)
	for _, existing := range m.repos {
		if strings.EqualFold(existing.Repo, repoName) {
			m.statusMessage = "Repo already exists in the list."
			m.statusIsError = true
			return m, nil
		}
	}
	if repoName == "" {
		m.statusMessage = "Only GitHub repo URLs are supported for now."
		m.statusIsError = true
		return m, nil
	}

	m.isCloning = true
	m.statusMessage = "Cloning repo .."
	m.statusIsError = false

	cmd := func() tea.Msg {
		err := m.svc.CloneRepo(repoName, raw)
		return addRepoMsg{
			repo: core.CodeRepo{
				Repo: repoName,
				URL:  raw,
			},
			err: err,
		}
	}

	return m, cmd
}

func (m repoModel) deleteRepo() (repoModel, tea.Cmd) {
	if len(m.repos) == 0 {
		m.statusMessage = "No repository selected."
		m.statusIsError = true
		return m, nil
	}

	m.isDeletingRepo = true
	deleteCmd := func() tea.Msg {
		err := m.svc.DeleteRepo(m.repos[m.selectedRepo].Repo)
		if err != nil {
			return deleteRepoMsg{err: err, repos: m.repos, deleted: m.repos[m.selectedRepo].Repo}
		}
		repos, err := m.dbRepo.ListCodeRepo()
		return deleteRepoMsg{err: err, repos: repos}
	}
	m.statusMessage = "Delete repo.."
	m.statusIsError = false
	return m, deleteCmd
}

func (m repoModel) View() string {
	logo := logoStyle.Render(strings.Join([]string{
		" _   _  ____ ___ ",
		"| \\ | |/ ___|_ _|",
		"|  \\| | |    | | ",
		"| |\\  | |___ | | ",
		"|_| \\_|\\____|___|",
	}, "\n"))

	inputCard := m.renderRepoInput()
	repoListCard := m.renderRepoList()
	sections := lipgloss.JoinVertical(lipgloss.Left, inputCard, repoListCard)

	return strings.Join([]string{
		logo,
		"",
		sections,
	}, "\n")
}

func (m repoModel) renderRepoInput() string {
	var b strings.Builder
	b.WriteString(sectionTitleStyle.Render("1) Add Repo"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Paste GitHub repo URL, then press enter."))
	b.WriteString("\n\n")

	inputText := m.repoInput
	if inputText == "" {
		inputText = placeholderStyle.Render("https://github.com/owner/repo.git")
	}
	cursor := ""
	if m.projectFocus == focusInput {
		cursor = "█"
	}
	b.WriteString(inputStyle.Render("> " + inputText + cursor))

	if m.statusMessage != "" {
		b.WriteString("\n\n")
		if m.statusIsError {
			b.WriteString(errorStyle.Render(m.statusMessage))
		} else {
			b.WriteString(successStyle.Render(m.statusMessage))
		}
	}

	style := cardStyle.Border(lipgloss.NormalBorder(), false, false, false, true)
	if m.projectFocus == focusInput {
		style = style.BorderForeground(lipgloss.Color("45"))
	}

	return style.Render(b.String())
}

func (m repoModel) renderRepoList() string {
	var b strings.Builder
	b.WriteString(sectionTitleStyle.Render("2) Existing Repos"))
	b.WriteString("\n")

	if len(m.repos) == 0 {
		b.WriteString(mutedStyle.Render("No repos yet."))
	} else {
		for i, repo := range m.repos {
			line := fmt.Sprintf("  %s\n  %s", repo.Repo, repo.URL)
			if i == m.selectedRepo {
				b.WriteString(selectedItemStyle.Render("> " + strings.TrimSpace(line)))
			} else {
				b.WriteString(line)
			}
			if i < len(m.repos)-1 {
				b.WriteString("\n\n")
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("tab focus this section, j/k move, d delete"))

	style := cardStyle.Border(lipgloss.NormalBorder(), false, false, false, true)
	if m.projectFocus == focusRepoList {
		style = style.BorderForeground(lipgloss.Color("45"))
	}
	return style.Render(b.String())
}

func isRepoURL(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "git@")
}
