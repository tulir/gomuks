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

package types

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

type Message struct {
	BasicMeta
	Type            string
	ID              string
	Text            string
	sending         bool
	buffer          []string
	prevBufferWidth int
}

func NewMessage(id, sender, msgtype, text, timestamp, date string, senderColor tcell.Color) *Message {
	return &Message{
		BasicMeta: BasicMeta{
			Sender:         sender,
			Timestamp:      timestamp,
			Date:           date,
			SenderColor:    senderColor,
			TextColor:      tcell.ColorDefault,
			TimestampColor: tcell.ColorDefault,
		},
		Type:            msgtype,
		Text:            text,
		ID:              id,
		prevBufferWidth: 0,
		sending:         false,
	}
}

var (
	boundaryPattern = regexp.MustCompile("([[:punct:]]\\s*|\\s+)")
	spacePattern    = regexp.MustCompile(`\s+`)
)

func (message *Message) CopyTo(to *Message) {
	to.BasicMeta = message.BasicMeta
	to.ID = message.ID
	to.Text = message.Text
	to.RecalculateBuffer()
}

func (message *Message) CalculateBuffer(width int) {
	if width < 2 {
		return
	}
	message.buffer = []string{}
	text := message.Text
	if message.Type == "m.emote" {
		text = fmt.Sprintf("* %s %s", message.Sender, message.Text)
	}
	forcedLinebreaks := strings.Split(text, "\n")
	newlines := 0
	for _, str := range forcedLinebreaks {
		if len(str) == 0 && newlines < 1 {
			message.buffer = append(message.buffer, "")
			newlines++
		} else {
			newlines = 0
		}
		// From tview/textview.go#reindexBuffer()
		for len(str) > 0 {
			extract := runewidth.Truncate(str, width, "")
			if len(extract) < len(str) {
				if spaces := spacePattern.FindStringIndex(str[len(extract):]); spaces != nil && spaces[0] == 0 {
					extract = str[:len(extract)+spaces[1]]
				}

				matches := boundaryPattern.FindAllStringIndex(extract, -1)
				if len(matches) > 0 {
					extract = extract[:matches[len(matches)-1][1]]
				}
			}
			message.buffer = append(message.buffer, extract)
			str = str[len(extract):]
		}
	}
	message.prevBufferWidth = width
}

func (message *Message) GetDisplaySender() string {
	if message.sending {
		return "Sending..."
	}
	switch message.Type {
	case "m.emote":
		return ""
	default:
		return message.Sender
	}
}

func (message *Message) SetIsSending(sending bool) {
	message.sending = sending
}

func (message *Message) RecalculateBuffer() {
	message.CalculateBuffer(message.prevBufferWidth)
}

func (message *Message) Buffer() []string {
	return message.buffer
}

func (message *Message) Height() int {
	return len(message.buffer)
}
