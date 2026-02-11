package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type textInput struct {
	value       string
	cursor      int
	width       int
	placeholder string
}

func newTextInput(width int, placeholder string) textInput {
	if width < 1 {
		width = 1
	}
	return textInput{
		width:       width,
		placeholder: placeholder,
	}
}

func (in textInput) Value() string {
	return in.value
}

func (in textInput) SetValue(v string) textInput {
	in.value = v
	in.cursor = len([]rune(v))
	return in
}

func (in textInput) Clear() textInput {
	in.value = ""
	in.cursor = 0
	return in
}

func (in textInput) SetWidth(width int) textInput {
	if width < 1 {
		width = 1
	}
	in.width = width
	in = in.clampCursor()
	return in
}

func (in textInput) Update(msg tea.KeyMsg) (textInput, bool) {
	in = in.clampCursor()
	runes := []rune(in.value)

	switch msg.String() {
	case "left", "ctrl+b":
		if in.cursor > 0 {
			in.cursor--
		}
		return in, true
	case "right", "ctrl+f":
		if in.cursor < len(runes) {
			in.cursor++
		}
		return in, true
	case "home", "ctrl+a":
		in.cursor = 0
		return in, true
	case "end", "ctrl+e":
		in.cursor = len(runes)
		return in, true
	case "backspace":
		if in.cursor > 0 && len(runes) > 0 {
			in.value = string(append(runes[:in.cursor-1], runes[in.cursor:]...))
			in.cursor--
		}
		return in, true
	case "delete":
		if in.cursor < len(runes) {
			in.value = string(append(runes[:in.cursor], runes[in.cursor+1:]...))
		}
		return in, true
	}

	if len(msg.Runes) > 0 {
		insert := msg.Runes
		runes = append(runes[:in.cursor], append(insert, runes[in.cursor:]...)...)
		in.value = string(runes)
		in.cursor += len(insert)
		return in, true
	}

	return in, false
}

func (in textInput) View(focused bool) string {
	if focused {
		return inputViewport(in.value, in.cursor, in.width)
	}
	if strings.TrimSpace(in.value) == "" {
		return placeholderStyle.Render(truncateRunes(in.placeholder, in.width))
	}
	return truncateRunes(in.value, in.width)
}

func (in textInput) clampCursor() textInput {
	if in.cursor < 0 {
		in.cursor = 0
	}
	max := len([]rune(in.value))
	if in.cursor > max {
		in.cursor = max
	}
	return in
}

func inputViewport(value string, cursor, width int) string {
	if width < 1 {
		width = 1
	}

	runes := []rune(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}

	visibleInput := max(
		// keep room for cursor
		width-1, 1)

	start := 0
	if cursor > visibleInput {
		start = cursor - visibleInput
	}
	end := min(start+visibleInput, len(runes))

	segment := runes[start:end]
	pos := min(max(cursor-start, 0), len(segment))

	withCursor := string(segment[:pos]) + "|" + string(segment[pos:])
	return truncateRunes(withCursor, width)
}
