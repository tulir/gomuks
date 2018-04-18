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
	"encoding/gob"
	"regexp"
	"time"

	"maunium.net/go/gomuks/ui/messages/tstring"
)

func init() {
	gob.Register(BaseTextMessage{})
}

type BaseTextMessage struct {
	BaseMessage
}

func newBaseTextMessage(id, sender, displayname, msgtype string, timestamp time.Time) BaseTextMessage {
	return BaseTextMessage{newBaseMessage(id, sender, displayname, msgtype, timestamp)}
}

// Regular expressions used to split lines when calculating the buffer.
//
// From tview/textview.go
var (
	boundaryPattern = regexp.MustCompile("([[:punct:]]\\s*|\\s+)")
	spacePattern    = regexp.MustCompile(`\s+`)
)

// CalculateBuffer generates the internal buffer for this message that consists
// of the text of this message split into lines at most as wide as the width
// parameter.
func (msg *BaseTextMessage) calculateBufferWithText(text tstring.TString, width int) {
	if width < 2 {
		return
	}

	msg.buffer = []tstring.TString{}

	forcedLinebreaks := text.Split('\n')
	newlines := 0
	for _, str := range forcedLinebreaks {
		if len(str) == 0 && newlines < 1 {
			msg.buffer = append(msg.buffer, tstring.TString{})
			newlines++
		} else {
			newlines = 0
		}
		// Mostly from tview/textview.go#reindexBuffer()
		for len(str) > 0 {
			extract := str.Truncate(width)
			if len(extract) < len(str) {
				if spaces := spacePattern.FindStringIndex(str[len(extract):].String()); spaces != nil && spaces[0] == 0 {
					extract = str[:len(extract)+spaces[1]]
				}

				matches := boundaryPattern.FindAllStringIndex(extract.String(), -1)
				if len(matches) > 0 {
					extract = extract[:matches[len(matches)-1][1]]
				}
			}
			msg.buffer = append(msg.buffer, extract)
			str = str[len(extract):]
		}
	}
	msg.prevBufferWidth = width
}
