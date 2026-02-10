package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type form struct {
	idx        int
	valueWidth int
	kvs        []KV
}

func newForm(kvs []KV, valueWidth int) form {
	return form{
		kvs:        kvs,
		valueWidth: valueWidth,
	}
}

// Update implements tea.Model.
func (f form) Update(msg tea.KeyMsg) form {
	switch msg.String() {
	case "up":
		f.idx = modIdx(f.idx, len(f.kvs), -1)
	case "down":
		f.idx = modIdx(f.idx, len(f.kvs), 1)
	}
	return f
}

// View implements tea.Model.
func (f form) View() string {
	var text = []string{}
	for idx, pair := range f.kvs {
		text = append(text, renderLabelInputRow(pair.key, pair.val, idx == f.idx, f.valueWidth))
	}
	return strings.Join(text, "\n\n")
}
