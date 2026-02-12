package tui

import (
	"dexianta/nci/core"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	globalSettingsRepo = "global_sys"
	globalLogKey       = "log_retention_days"
)

type settingsModel struct {
	dbRepo core.DbRepo
	repo   string

	sectionIdx   int
	sectionFocus bool
	sectionItems []string

	// sections
	globalSetting core.GlobalSetting
	EnvVars       []KV
	sshConfig     []SSHConfig

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

func newSettingModel(dbRepo core.DbRepo) settingsModel {
	return settingsModel{
		dbRepo:        dbRepo,
		sectionFocus:  true,
		sectionItems:  []string{"Global", "Env Vars", "SSH"},
		globalSetting: core.GlobalSetting{LogRetentionDays: 3},
		globalConfForm: newForm([]KV{
			{key: "Log Retention Days", val: "3", dtype: "int"},
		}, 14, false),
		varsForm:  newForm([]KV{}, 20, true),
		sshViewer: newSSHViewer(),
	}
}

func (s settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd, bool) {
	switch m := msg.(type) {
	case selectRepoMsg:
		s.repo = m.repo
		if s.dbRepo == nil || s.repo == "" {
			s.varsForm.kvs = nil
			return s, nil, true
		}
		return s, tea.Batch(
			loadRepoSettingCmd(s.dbRepo, s.repo),
			loadRepoSettingCmd(s.dbRepo, globalSettingsRepo),
		), true

	case loadRepoSettingMsg:
		if m.repo == globalSettingsRepo {
			if m.err != nil {
				s.globalConfForm.statusMessage = "Failed to load global settings: " + m.err.Error()
				s.globalConfForm.statusIsError = true
				return s, nil, true
			}

			days := s.globalSetting.LogRetentionDays
			for _, setting := range m.settings {
				if setting.Key != globalLogKey {
					continue
				}
				parsed, err := strconv.Atoi(setting.Value)
				if err != nil {
					s.globalConfForm.statusMessage = "Invalid global setting value for " + globalLogKey
					s.globalConfForm.statusIsError = true
					return s, nil, true
				}
				days = parsed
				break
			}
			s.globalSetting.LogRetentionDays = days
			if len(s.globalConfForm.kvs) == 0 {
				s.globalConfForm.kvs = []KV{
					{key: "Log Retention Days", val: strconv.Itoa(days), dtype: "int"},
				}
			} else {
				s.globalConfForm.kvs[0].val = strconv.Itoa(days)
			}
			return s, nil, true
		}

		if m.repo != s.repo {
			return s, nil, true
		}
		if m.err != nil {
			s.varsForm.statusMessage = "Failed to load repo env vars: " + m.err.Error()
			s.varsForm.statusIsError = true
			return s, nil, true
		}
		s.varsForm.kvs = repoSettingToKVs(m.settings)
		if len(s.varsForm.kvs) == 0 {
			s.varsForm.idx = 0
		} else if s.varsForm.idx >= len(s.varsForm.kvs) {
			s.varsForm.idx = len(s.varsForm.kvs) - 1
		}
		return s, nil, true

	case addSettingMsg:
		if m.add.Repo == globalSettingsRepo {
			if m.err != nil {
				s.globalConfForm.statusMessage = "Failed to save global setting " + m.add.Key + ": " + m.err.Error()
				s.globalConfForm.statusIsError = true
				return s, nil, true
			}
			if m.add.Key == globalLogKey {
				if parsed, err := strconv.Atoi(m.add.Value); err == nil {
					s.globalSetting.LogRetentionDays = parsed
				}
			}
			return s, nil, true
		}

		if m.add.Repo != s.repo {
			return s, nil, true
		}
		if m.err != nil {
			s.varsForm.statusMessage = "Failed to save env var " + m.add.Key + ": " + m.err.Error()
			s.varsForm.statusIsError = true
		}
		return s, nil, true
	case delSettingMsg:
		if m.del.Repo != s.repo {
			return s, nil, true
		}
		if m.err != nil {
			s.varsForm.statusMessage = "Failed to delete env var " + m.del.Key + ": " + m.err.Error()
			s.varsForm.statusIsError = true
		}
		return s, nil, true

	case tea.KeyMsg:
		return s.updateKey(m)
	}

	return s, nil, false
}

func (s settingsModel) updateKey(msg tea.KeyMsg) (settingsModel, tea.Cmd, bool) {
	var handled bool
	switch s.sectionFocus {
	case true:
		// focus on section
		switch msg.String() {
		case "up":
			s.sectionIdx = modIdx(s.sectionIdx, len(s.sectionItems), -1)
			return s, nil, true
		case "down":
			s.sectionIdx = modIdx(s.sectionIdx, len(s.sectionItems), 1)
			return s, nil, true
		case "tab":
			s.sectionFocus = false
			return s, nil, true
		}
	case false:
		// focus on editor
		switch msg.String() {
		case "tab":
			if !s.activeForm().IsEditing() {
				s.sectionFocus = true
				return s, nil, true
			}
		}
		switch s.sectionIdx {
		case 0:
			s.globalConfForm.addKVCmd = func(kv KV) tea.Cmd {
				db := s.dbRepo
				return func() tea.Msg {
					if db == nil {
						return nil
					}
					setting := core.RepoSetting{
						Repo:  globalSettingsRepo,
						Key:   globalLogKey,
						Value: kv.val,
					}
					return addSettingMsg{
						add: setting,
						err: db.SaveRepoSetting(setting),
					}
				}
			}
			var cmd tea.Cmd
			s.globalConfForm, cmd, handled = s.globalConfForm.Update(msg)
			return s, cmd, handled
		case 1:
			s.varsForm.addKVCmd = func(kv KV) tea.Cmd {
				repo := s.repo
				db := s.dbRepo
				return func() tea.Msg {
					if db == nil || repo == "" {
						return nil
					}

					setting := core.RepoSetting{
						Repo:  repo,
						Key:   kv.key,
						Value: kv.val,
					}
					return addSettingMsg{
						add: setting,
						err: db.SaveRepoSetting(setting),
					}
				}
			}

			s.varsForm.delKVCmd = func(kv KV) tea.Cmd {
				repo := s.repo
				db := s.dbRepo
				return func() tea.Msg {
					if db == nil || repo == "" {
						return nil
					}
					setting := core.RepoSetting{
						Repo:  repo,
						Key:   kv.key,
						Value: kv.val,
					}
					return delSettingMsg{
						del: setting,
						err: db.DeleteRepoSetting(repo, kv.key),
					}
				}
			}

			var cmd tea.Cmd
			s.varsForm, cmd, handled = s.varsForm.Update(msg)
			return s, cmd, handled
		case 2:
			s.sshViewer = s.sshViewer.Update(msg)
		}
	}
	return s, nil, false
}

func loadRepoSettingCmd(dbRepo core.DbRepo, repo string) tea.Cmd {
	return func() tea.Msg {
		settings, err := dbRepo.ListRepoSetting(repo)
		return loadRepoSettingMsg{
			repo:     repo,
			settings: settings,
			err:      err,
		}
	}
}

func repoSettingToKVs(settings []core.RepoSetting) []KV {
	out := make([]KV, 0, len(settings))
	for _, s := range settings {
		out = append(out, KV{
			key:   s.Key,
			val:   s.Value,
			dtype: "string",
		})
	}
	return out
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
		renderHint("UP/DOWN", "move "),
		renderHint("TAB", "switch focus"),
	)
}
