package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type formMode int

const (
	formModeBrowse formMode = iota
	formModeEditValue
	formModeAddKey
	formModeAddValue
)

type form struct {
	idx        int
	valueWidth int
	kvs        []KV
	editable   bool

	mode     formMode
	input    textInput
	draftKey string

	addKVCmd func(KV) tea.Cmd
	delKVCmd func(KV) tea.Cmd

	statusMessage string
	statusIsError bool
}

func newForm(kvs []KV, valueWidth int, editable bool) form {
	return form{
		kvs:        kvs,
		valueWidth: valueWidth,
		editable:   editable,
		input:      newTextInput(valueWidth, ""),
	}
}

func (f form) Update(msg tea.KeyMsg) (form, tea.Cmd, bool) {
	var handled bool
	if f.mode != formModeBrowse {
		return f.updateInputMode(msg)
	}

	switch msg.String() {
	case "up":
		if len(f.kvs) > 0 {
			f.idx = modIdx(f.idx, len(f.kvs), -1)
			return f, nil, true
		}
	case "down":
		if len(f.kvs) > 0 {
			f.idx = modIdx(f.idx, len(f.kvs), 1)
			return f, nil, true
		}
	case "e":
		if len(f.kvs) > 0 {
			f.mode = formModeEditValue
			f.input = f.input.SetWidth(f.valueContentWidth()).SetValue(f.kvs[f.idx].val)
			return f, nil, true
		}
	case "a":
		if f.editable {
			f.mode = formModeAddKey
			f.input = f.input.SetWidth(f.keyContentWidth()).Clear()
			f.draftKey = ""
			return f, nil, true
		}
	case "d":
		if f.editable {
			var cmd tea.Cmd
			f, cmd = f.deleteAtSelection()
			return f, cmd, true
		}
	}
	return f, nil, handled
}

func (f form) View() string {
	var text = []string{}
	for i, pair := range f.kvs {
		rowFocused := i == f.idx
		label := pair.key
		value := pair.val

		if i == f.idx {
			switch f.mode {
			case formModeEditValue:
				value = f.input.SetWidth(f.valueContentWidth()).View(true)
			}
		}

		text = append(text, renderLabelInputRow(label, value, rowFocused, f.valueWidth))
	}

	if len(f.kvs) == 0 {
		text = append(text, mutedStyle.Render("No entries yet."))
	}

	if f.mode == formModeAddKey || f.mode == formModeAddValue {
		addLabel := "new key"
		addValue := ""
		if f.mode == formModeAddKey {
			addLabel = f.input.SetWidth(f.keyContentWidth()).View(true)
		}
		if f.mode == formModeAddValue {
			addLabel = f.draftKey
			addValue = f.input.SetWidth(f.valueContentWidth()).View(true)
		}
		text = append(text, renderLabelInputRow(addLabel, addValue, true, f.valueWidth))
	}

	text = append(text, mutedStyle.Render(f.helpText()))

	if f.statusMessage != "" {
		if f.statusIsError {
			text = append(text, errorStyle.Render(f.statusMessage))
		} else {
			text = append(text, successStyle.Render(f.statusMessage))
		}
	}

	return strings.Join(text, "\n\n")
}

func (f form) IsEditing() bool {
	return f.mode != formModeBrowse
}

func (f form) helpText() string {
	if f.mode == formModeBrowse {
		if f.editable {
			return "[up/down] [e]dit-value [a]dd [d]elete"
		}
		return "[up/down] [e]dit"
	}

	switch f.mode {
	case formModeEditValue:
		return "Editing value: [left/right]move [enter]save [esc]cancel"
	case formModeAddKey:
		return "Adding entry key: [left/right]move [enter]to val [esc]cancel"
	case formModeAddValue:
		return "Adding entry value: [left/right]move [enter]save [esc]cancel"
	default:
		return ""
	}
}

func (f form) updateInputMode(msg tea.KeyMsg) (form, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		f.mode = formModeBrowse
		f.input = f.input.Clear()
		f.draftKey = ""
		return f, nil, true
	case "enter":
		f, cmd := f.commitInput()
		return f, cmd, true
	}

	nextInput, handled := f.input.Update(msg)
	f.input = nextInput
	return f, nil, handled
}

func (f form) commitInput() (form, tea.Cmd) {
	raw := f.input.Value()

	switch f.mode {
	case formModeEditValue:
		if len(f.kvs) == 0 {
			return f, nil
		}
		sanitized, err := sanitizeValueForType(f.kvs[f.idx].dtype, raw)
		if err != nil {
			f.statusMessage = fmt.Sprintf("Invalid value for %s: %v", f.kvs[f.idx].key, err)
			f.statusIsError = true
			return f, nil
		}
		f.kvs[f.idx].val = sanitized
		f.statusMessage = fmt.Sprintf("Updated value: %s", f.kvs[f.idx].key)
		f.statusIsError = false
		f.mode = formModeBrowse
		f.input = f.input.Clear()
		if f.addKVCmd == nil {
			return f, nil
		}
		return f, f.addKVCmd(f.kvs[f.idx])

	case formModeAddKey:
		key := strings.TrimSpace(raw)
		if key == "" {
			f.statusMessage = "Key cannot be empty."
			f.statusIsError = true
			return f, nil
		}
		if f.hasDuplicateKey(key, -1) {
			f.statusMessage = "Key already exists."
			f.statusIsError = true
			return f, nil
		}
		f.draftKey = key
		f.input = f.input.SetWidth(f.valueContentWidth()).Clear()
		f.mode = formModeAddValue
		f.statusMessage = ""
		f.statusIsError = false

	case formModeAddValue:
		dtype := "string"
		sanitized, err := sanitizeValueForType(dtype, raw)
		if err != nil {
			f.statusMessage = fmt.Sprintf("Invalid value: %v", err)
			f.statusIsError = true
			return f, nil
		}

		kv := KV{
			key:   f.draftKey,
			val:   sanitized,
			dtype: dtype,
		}
		f.kvs = append(f.kvs, kv)
		f.idx = len(f.kvs) - 1
		f.mode = formModeBrowse
		f.statusMessage = "Added new entry."
		f.statusIsError = false
		f.input = f.input.Clear()
		f.draftKey = ""
		if f.addKVCmd != nil {
			return f, f.addKVCmd(kv)
		}
		return f, nil
	}
	return f, nil
}

func (f form) hasDuplicateKey(key string, exceptIdx int) bool {
	for i, kv := range f.kvs {
		if i == exceptIdx {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv.key), strings.TrimSpace(key)) {
			return true
		}
	}
	return false
}

func (f form) deleteAtSelection() (form, tea.Cmd) {
	if len(f.kvs) == 0 {
		f.statusMessage = "No entry to delete."
		f.statusIsError = true
		return f, nil
	}

	deleted := f.kvs[f.idx]
	f.kvs = append(f.kvs[:f.idx], f.kvs[f.idx+1:]...)
	if f.idx >= len(f.kvs) && f.idx > 0 {
		f.idx--
	}
	f.statusMessage = fmt.Sprintf("Deleted: %s", deleted.key)
	f.statusIsError = false
	if f.delKVCmd == nil {
		return f, nil
	}
	return f, f.delKVCmd(deleted)
}

func sanitizeValueForType(dtype, raw string) (string, error) {
	kind := strings.ToLower(strings.TrimSpace(dtype))
	switch kind {
	case "", "string":
		return raw, nil
	case "int", "integer", "number":
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return "", fmt.Errorf("must be an integer")
		}
		return strconv.Itoa(n), nil
	case "bool", "boolean":
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "1", "yes", "y", "on":
			return "true", nil
		case "false", "0", "no", "n", "off":
			return "false", nil
		default:
			return "", fmt.Errorf("must be true/false")
		}
	case "date":
		s := strings.TrimSpace(raw)
		if s == "" {
			return "", fmt.Errorf("date cannot be empty")
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t.Format("2006-01-02"), nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.Format(time.RFC3339), nil
		}
		return "", fmt.Errorf("must be YYYY-MM-DD or RFC3339")
	default:
		return raw, nil
	}
}

func (f form) valueContentWidth() int {
	w := f.valueWidth - valueStyle.GetHorizontalFrameSize()
	if w < 1 {
		return 1
	}
	return w
}

func (f form) keyContentWidth() int {
	w := labelStyle.GetWidth()
	if w < 1 {
		return 22
	}
	return w
}
