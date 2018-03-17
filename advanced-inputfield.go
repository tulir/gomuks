// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"github.com/zyedidia/clipboard"
	"maunium.net/go/tview"
)

// AdvancedInputField is a multi-line user-editable text area.
//
// Use SetMaskCharacter() to hide input from onlookers (e.g. for password
// input).
type AdvancedInputField struct {
	*tview.Box

	// Cursor position
	cursorOffset int
	viewOffset   int

	// The text that was entered.
	text string

	// The text to be displayed before the input area.
	label string

	// The text to be displayed in the input area when "text" is empty.
	placeholder string

	// The label color.
	labelColor tcell.Color

	// The background color of the input area.
	fieldBackgroundColor tcell.Color

	// The text color of the input area.
	fieldTextColor tcell.Color

	// The text color of the placeholder.
	placeholderTextColor tcell.Color

	// The screen width of the input area. A value of 0 means extend as much as
	// possible.
	fieldWidth int

	// A character to mask entered text (useful for password fields). A value of 0
	// disables masking.
	maskCharacter rune

	// An optional function which may reject the last character that was entered.
	accept func(text string, ch rune) bool

	// An optional function which is called when the input has changed.
	changed func(text string)

	// An optional function which is called when the user indicated that they
	// are done entering text. The key which was pressed is provided (enter or escape).
	done func(tcell.Key)

	// An optional function which is called when the user presses tab.
	tabComplete func(text string, cursorOffset int) string
}

// NewAdvancedInputField returns a new input field.
func NewAdvancedInputField() *AdvancedInputField {
	return &AdvancedInputField{
		Box:                  tview.NewBox(),
		labelColor:           tview.Styles.SecondaryTextColor,
		fieldBackgroundColor: tview.Styles.ContrastBackgroundColor,
		fieldTextColor:       tview.Styles.PrimaryTextColor,
		placeholderTextColor: tview.Styles.ContrastSecondaryTextColor,
	}
}

// SetText sets the current text of the input field.
func (field *AdvancedInputField) SetText(text string) *AdvancedInputField {
	field.text = text
	if field.changed != nil {
		field.changed(text)
	}
	return field
}

// GetText returns the current text of the input field.
func (field *AdvancedInputField) GetText() string {
	return field.text
}

// SetLabel sets the text to be displayed before the input area.
func (field *AdvancedInputField) SetLabel(label string) *AdvancedInputField {
	field.label = label
	return field
}

// GetLabel returns the text to be displayed before the input area.
func (field *AdvancedInputField) GetLabel() string {
	return field.label
}

// SetPlaceholder sets the text to be displayed when the input text is empty.
func (field *AdvancedInputField) SetPlaceholder(text string) *AdvancedInputField {
	field.placeholder = text
	return field
}

// SetLabelColor sets the color of the label.
func (field *AdvancedInputField) SetLabelColor(color tcell.Color) *AdvancedInputField {
	field.labelColor = color
	return field
}

// SetFieldBackgroundColor sets the background color of the input area.
func (field *AdvancedInputField) SetFieldBackgroundColor(color tcell.Color) *AdvancedInputField {
	field.fieldBackgroundColor = color
	return field
}

// SetFieldTextColor sets the text color of the input area.
func (field *AdvancedInputField) SetFieldTextColor(color tcell.Color) *AdvancedInputField {
	field.fieldTextColor = color
	return field
}

// SetPlaceholderExtColor sets the text color of placeholder text.
func (field *AdvancedInputField) SetPlaceholderExtColor(color tcell.Color) *AdvancedInputField {
	field.placeholderTextColor = color
	return field
}

// SetFormAttributes sets attributes shared by all form items.
func (field *AdvancedInputField) SetFormAttributes(label string, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	field.label = label
	field.labelColor = labelColor
	field.SetBackgroundColor(bgColor)
	field.fieldTextColor = fieldTextColor
	field.fieldBackgroundColor = fieldBgColor
	return field
}

// SetFieldWidth sets the screen width of the input area. A value of 0 means
// extend as much as possible.
func (field *AdvancedInputField) SetFieldWidth(width int) *AdvancedInputField {
	field.fieldWidth = width
	return field
}

// GetFieldWidth returns this primitive's field width.
func (field *AdvancedInputField) GetFieldWidth() int {
	return field.fieldWidth
}

// SetMaskCharacter sets a character that masks user input on a screen. A value
// of 0 disables masking.
func (field *AdvancedInputField) SetMaskCharacter(mask rune) *AdvancedInputField {
	field.maskCharacter = mask
	return field
}

// SetAcceptanceFunc sets a handler which may reject the last character that was
// entered (by returning false).
//
// This package defines a number of variables Prefixed with AdvancedInputField which may
// be used for common input (e.g. numbers, maximum text length).
func (field *AdvancedInputField) SetAcceptanceFunc(handler func(textToCheck string, lastChar rune) bool) *AdvancedInputField {
	field.accept = handler
	return field
}

// SetChangedFunc sets a handler which is called whenever the text of the input
// field has changed. It receives the current text (after the change).
func (field *AdvancedInputField) SetChangedFunc(handler func(text string)) *AdvancedInputField {
	field.changed = handler
	return field
}

// SetDoneFunc sets a handler which is called when the user is done entering
// text. The callback function is provided with the key that was pressed, which
// is one of the following:
//
//   - KeyEnter: Done entering text.
//   - KeyEscape: Abort text input.
func (field *AdvancedInputField) SetDoneFunc(handler func(key tcell.Key)) *AdvancedInputField {
	field.done = handler
	return field
}

func (field *AdvancedInputField) SetTabCompleteFunc(handler func(text string, cursorOffset int) string) *AdvancedInputField {
	field.tabComplete = handler
	return field
}

// SetFinishedFunc calls SetDoneFunc().
func (field *AdvancedInputField) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	return field.SetDoneFunc(handler)
}

// Draw draws this primitive onto the screen.
func (field *AdvancedInputField) Draw(screen tcell.Screen) {
	field.Box.Draw(screen)

	// Prepare
	x, y, width, height := field.GetInnerRect()
	rightLimit := x + width
	if height < 1 || rightLimit <= x {
		return
	}

	// Draw label.
	_, drawnWidth := tview.Print(screen, field.label, x, y, rightLimit-x, tview.AlignLeft, field.labelColor)
	x += drawnWidth

	// Draw input area.
	fieldWidth := field.fieldWidth
	if fieldWidth == 0 {
		fieldWidth = math.MaxInt32
	}
	if rightLimit-x < fieldWidth {
		fieldWidth = rightLimit - x
	}
	fieldStyle := tcell.StyleDefault.Background(field.fieldBackgroundColor)
	for index := 0; index < fieldWidth; index++ {
		screen.SetContent(x+index, y, ' ', nil, fieldStyle)
	}

	text := field.text
	if text == "" && field.placeholder != "" {
		tview.Print(screen, field.placeholder, x, y, fieldWidth, tview.AlignLeft, field.placeholderTextColor)
	}

	// Draw entered text.
	if field.maskCharacter > 0 {
		text = strings.Repeat(string(field.maskCharacter), utf8.RuneCountInString(field.text))
	}
	textWidth := runewidth.StringWidth(text)
	if field.cursorOffset >= textWidth {
		fieldWidth--
	}

	// Recalculate view offset
	if field.cursorOffset < field.viewOffset {
		field.viewOffset = field.cursorOffset
	} else if field.cursorOffset > field.viewOffset+fieldWidth {
		field.viewOffset = field.cursorOffset - fieldWidth
	} else if textWidth-field.viewOffset < fieldWidth {
		field.viewOffset = textWidth - fieldWidth
	}
	// Make sure view offset didn't become negative
	if field.viewOffset < 0 {
		field.viewOffset = 0
	}

	// Draw entered text.
	runes := []rune(text)
	relPos := 0
	for pos := field.viewOffset; pos <= fieldWidth+field.viewOffset && pos < len(runes); pos++ {
		ch := runes[pos]
		w := runewidth.RuneWidth(ch)
		_, _, style, _ := screen.GetContent(x+relPos, y)
		style = style.Foreground(field.fieldTextColor)
		for w > 0 {
			screen.SetContent(x+relPos, y, ch, nil, style)
			relPos++
			w--
		}
	}

	// Set cursor.
	if field.GetFocusable().HasFocus() {
		field.setCursor(screen)
	}
}

func (field *AdvancedInputField) GetCursorOffset() int {
	return field.cursorOffset
}

func (field *AdvancedInputField) SetCursorOffset(offset int) *AdvancedInputField {
	if offset < 0 {
		offset = 0
	} else {
		width := runewidth.StringWidth(field.text)
		if offset >= width {
			offset = width
		}
	}
	field.cursorOffset = offset
	return field
}

// setCursor sets the cursor position.
func (field *AdvancedInputField) setCursor(screen tcell.Screen) {
	x, y, width, _ := field.GetRect()
	origX, origY := x, y
	rightLimit := x + width
	if field.HasBorder() {
		x++
		y++
		rightLimit -= 2
	}
	fieldWidth := runewidth.StringWidth(field.text)
	if field.fieldWidth > 0 && fieldWidth > field.fieldWidth-1 {
		fieldWidth = field.fieldWidth - 1
	}
	x = x + tview.StringWidth(field.label) + field.cursorOffset - field.viewOffset
	if x >= rightLimit {
		x = rightLimit - 1
	} else if x < origX {
		x = origY
	}
	screen.ShowCursor(x, y)
}

var (
	lastWord  = regexp.MustCompile(`\S+\s*$`)
	firstWord = regexp.MustCompile(`^\s*\S+`)
)

func SubstringBefore(s string, w int) string {
	return runewidth.Truncate(s, w, "")
}

// InputHandler returns the handler for this primitive.
func (field *AdvancedInputField) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return field.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		currentText := field.text
		defer func() {
			// Trigger changed events.
			if field.text != currentText && field.changed != nil {
				field.changed(field.text)
			}
			// Make sure cursor offset is valid
			if field.cursorOffset < 0 {
				field.cursorOffset = 0
			}
			width := runewidth.StringWidth(field.text)
			if field.cursorOffset > width {
				field.cursorOffset = width
			}
		}()

		// Process key event.
		switch key := event.Key(); key {
		case tcell.KeyRune: // Regular character.
			leftPart := SubstringBefore(field.text, field.cursorOffset)
			newText := leftPart + string(event.Rune()) + field.text[len(leftPart):]
			if field.accept != nil {
				if !field.accept(newText, event.Rune()) {
					break
				}
			}
			field.text = newText
			field.cursorOffset += runewidth.RuneWidth(event.Rune())
		case tcell.KeyCtrlV:
			clip, _ := clipboard.ReadAll("clipboard")
			leftPart := SubstringBefore(field.text, field.cursorOffset)
			field.text = leftPart + clip + field.text[len(leftPart):]
			field.cursorOffset += runewidth.StringWidth(clip)
		case tcell.KeyLeft: // Move cursor left.
			before := SubstringBefore(field.text, field.cursorOffset)
			if event.Modifiers() == tcell.ModCtrl {
				found := lastWord.FindString(before)
				field.cursorOffset -= runewidth.StringWidth(found)
			} else if len(before) > 0 {
				beforeRunes := []rune(before)
				char := beforeRunes[len(beforeRunes)-1]
				field.cursorOffset -= runewidth.RuneWidth(char)
			}
		case tcell.KeyRight: // Move cursor right.
			before := SubstringBefore(field.text, field.cursorOffset)
			after := field.text[len(before):]
			if event.Modifiers() == tcell.ModCtrl {
				found := firstWord.FindString(after)
				field.cursorOffset += runewidth.StringWidth(found)
			} else if len(after) > 0 {
				char := []rune(after)[0]
				field.cursorOffset += runewidth.RuneWidth(char)
			}
		case tcell.KeyDelete: // Delete next character.
			if field.cursorOffset >= runewidth.StringWidth(field.text) {
				break
			}
			leftPart := SubstringBefore(field.text, field.cursorOffset)
			rightPart := field.text[len(leftPart):]
			rightPartRunes := []rune(rightPart)
			rightPartRunes = rightPartRunes[1:]
			rightPart = string(rightPartRunes)
			field.text = leftPart + rightPart
		case tcell.KeyBackspace, tcell.KeyBackspace2: // Delete last character.
			if field.cursorOffset == 0 {
				break
			}
			if key == tcell.KeyBackspace { // Ctrl+backspace
				leftPart := SubstringBefore(field.text, field.cursorOffset)
				rightPart := field.text[len(leftPart):]
				replacement := lastWord.ReplaceAllString(leftPart, "")
				field.text = replacement + rightPart

				field.cursorOffset -= runewidth.StringWidth(leftPart) - runewidth.StringWidth(replacement)
			} else { // Just backspace
				leftPart := SubstringBefore(field.text, field.cursorOffset)
				rightPart := field.text[len(leftPart):]
				leftPartRunes := []rune(leftPart)
				leftPartRunes = leftPartRunes[0 : len(leftPartRunes)-1]
				leftPart = string(leftPartRunes)
				removedChar := field.text[len(leftPart) : len(field.text)-len(rightPart)]
				field.text = leftPart + rightPart
				field.cursorOffset -= runewidth.StringWidth(removedChar)
			}
		case tcell.KeyTab: // Tab-completion
			if field.tabComplete != nil {
				oldWidth := runewidth.StringWidth(field.text)
				field.text = field.tabComplete(field.text, field.cursorOffset)
				newWidth := runewidth.StringWidth(field.text)
				if oldWidth != newWidth {
					field.cursorOffset += newWidth - oldWidth
				}
			}
		case tcell.KeyEnter, tcell.KeyEscape: // We're done.
			if field.done != nil {
				field.done(key)
			}
		}
	})
}
