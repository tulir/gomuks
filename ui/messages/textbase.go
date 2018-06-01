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

package messages

import (
	"regexp"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"fmt"
	"maunium.net/go/gomuks/config"
)

// Regular expressions used to split lines when calculating the buffer.
//
// From tview/textview.go
var (
	boundaryPattern = regexp.MustCompile(`([[:punct:]]\s*|\s+)`)
	bareBoundaryPattern = regexp.MustCompile(`(\s+)`)
	spacePattern    = regexp.MustCompile(`\s+`)
)

func matchBoundaryPattern(bare bool, extract tstring.TString) tstring.TString {
	regex := boundaryPattern
	if bare {
		regex = bareBoundaryPattern
	}
	matches := regex.FindAllStringIndex(extract.String(), -1)
	if len(matches) > 0 {
		if match := matches[len(matches)-1]; len(match) >= 2 {
			if until := match[1]; until < len(extract) {
				extract = extract[:until]
			}
		}
	}
	return extract
}

// CalculateBuffer generates the internal buffer for this message that consists
// of the text of this message split into lines at most as wide as the width
// parameter.
func (msg *BaseMessage) calculateBufferWithText(prefs config.UserPreferences, text tstring.TString, width int) {
	if width < 2 {
		return
	}

	msg.buffer = []tstring.TString{}

	if prefs.BareMessageView {
		newText := tstring.NewTString(msg.FormatTime())
		if len(msg.Sender()) > 0 {
			newText = newText.AppendTString(tstring.NewColorTString(fmt.Sprintf(" <%s> ", msg.Sender()), msg.SenderColor()))
		} else {
			newText = newText.Append(" ")
		}
		newText = newText.AppendTString(text)
		text = newText
	}

	forcedLinebreaks := text.Split('\n')
	newlines := 0
	for _, str := range forcedLinebreaks {
		if len(str) == 0 && newlines < 1 {
			msg.buffer = append(msg.buffer, tstring.TString{})
			newlines++
		} else {
			newlines = 0
		}
		// Adapted from tview/textview.go#reindexBuffer()
		for len(str) > 0 {
			extract := str.Truncate(width)
			if len(extract) < len(str) {
				if spaces := spacePattern.FindStringIndex(str[len(extract):].String()); spaces != nil && spaces[0] == 0 {
					extract = str[:len(extract)+spaces[1]]
				}
				extract = matchBoundaryPattern(prefs.BareMessageView, extract)
			}
			msg.buffer = append(msg.buffer, extract)
			str = str[len(extract):]
		}
	}
	msg.prevBufferWidth = width
	msg.prevPrefs = prefs
}
