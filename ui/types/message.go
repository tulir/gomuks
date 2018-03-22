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

// MessageState is an enum to specify if a Message is being sent, failed to send or was successfully sent.
type MessageState int

// Allowed MessageStates.
const (
	MessageStateSending MessageState = iota
	MessageStateDefault
	MessageStateFailed
)

// Message is a wrapper for the content and metadata of a Matrix message intended to be displayed.
type Message struct {
	ID              string
	Type            string
	Sender          string
	SenderColor     tcell.Color
	TextColor       tcell.Color
	Timestamp       string
	Date            string
	Text            string
	State           MessageState
	buffer          []string
	prevBufferWidth int
}

// NewMessage creates a new Message object with the provided values and the default state.
func NewMessage(id, sender, msgtype, text, timestamp, date string, senderColor tcell.Color) *Message {
	return &Message{
		Sender:          sender,
		Timestamp:       timestamp,
		Date:            date,
		SenderColor:     senderColor,
		TextColor:       tcell.ColorDefault,
		Type:            msgtype,
		Text:            text,
		ID:              id,
		prevBufferWidth: 0,
		State:           MessageStateDefault,
	}
}

// CopyTo copies the content of this message to the given message.
func (message *Message) CopyTo(to *Message) {
	to.ID = message.ID
	to.Type = message.Type
	to.Sender = message.Sender
	to.SenderColor = message.SenderColor
	to.TextColor = message.TextColor
	to.Timestamp = message.Timestamp
	to.Date = message.Date
	to.Text = message.Text
	to.State = message.State
	to.RecalculateBuffer()
}

// GetSender gets the string that should be displayed as the sender of this message.
//
// If the message is being sent, the sender is "Sending...".
// If sending has failed, the sender is "Error".
// If the message is an emote, the sender is blank.
// In any other case, the sender is the display name of the user who sent the message.
func (message *Message) GetSender() string {
	switch message.State {
	case MessageStateSending:
		return "Sending..."
	case MessageStateFailed:
		return "Error"
	}
	switch message.Type {
	case "m.emote":
		// Emotes don't show a separate sender, it's included in the buffer.
		return ""
	default:
		return message.Sender
	}
}

func (message *Message) getStateSpecificColor() tcell.Color {
	switch message.State {
	case MessageStateSending:
		return tcell.ColorGray
	case MessageStateFailed:
		return tcell.ColorRed
	case MessageStateDefault:
		fallthrough
	default:
		return tcell.ColorDefault
	}
}

// GetSenderColor returns the color the name of the sender should be shown in.
//
// If the message is being sent, the color is gray.
// If sending has failed, the color is red.
//
// In any other case, the color is whatever is specified in the Message struct.
// Usually that means it is the hash-based color of the sender (see ui/widget/color.go)
func (message *Message) GetSenderColor() (color tcell.Color) {
	color = message.getStateSpecificColor()
	if color == tcell.ColorDefault {
		color = message.SenderColor
	}
	return
}

// GetTextColor returns the color the actual content of the message should be shown in.
//
// This returns the same colors as GetSenderColor(), but takes the default color from a different variable.
func (message *Message) GetTextColor() (color tcell.Color) {
	color = message.getStateSpecificColor()
	if color == tcell.ColorDefault {
		color = message.TextColor
	}
	return
}

// GetTimestampColor returns the color the timestamp should be shown in.
//
// As with GetSenderColor(), messages being sent and messages that failed to be sent are
// gray and red respectively.
//
// However, other messages are the default color instead of a color stored in the struct.
func (message *Message) GetTimestampColor() tcell.Color {
	return message.getStateSpecificColor()
}

// RecalculateBuffer calculates the buffer again with the previously provided width.
func (message *Message) RecalculateBuffer() {
	message.CalculateBuffer(message.prevBufferWidth)
}

// Buffer returns the computed text buffer.
//
// The buffer contains the text of the message split into lines with a maximum
// width of whatever was provided to CalculateBuffer().
//
// N.B. This will NOT automatically calculate the buffer if it hasn't been
//      calculated already, as that requires the target width.
func (message *Message) Buffer() []string {
	return message.buffer
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (message *Message) Height() int {
	return len(message.buffer)
}

// GetTimestamp returns the formatted time when the message was sent.
func (message *Message) GetTimestamp() string {
	return message.Timestamp
}

// GetDate returns the formatted date when the message was sent.
func (message *Message) GetDate() string {
	return message.Date
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
