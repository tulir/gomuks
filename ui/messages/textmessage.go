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
	"fmt"
	"regexp"
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/widget"
)

func init() {
	gob.Register(&UITextMessage{})
	gob.Register(&UIExpandedTextMessage{})
}

type UIExpandedTextMessage struct {
	UITextMessage
	MsgUIStringText UIString
}

// NewExpandedTextMessage creates a new UIExpandedTextMessage object with the provided values and the default state.
func NewExpandedTextMessage(id, sender, msgtype string, text UIString, timestamp time.Time) UIMessage {
	return &UIExpandedTextMessage{
		UITextMessage{
			MsgSender:       sender,
			MsgTimestamp:    timestamp,
			MsgSenderColor:  widget.GetHashColor(sender),
			MsgType:         msgtype,
			MsgText:         text.String(),
			MsgID:           id,
			prevBufferWidth: 0,
			MsgState:        ifc.MessageStateDefault,
			MsgIsHighlight:  false,
			MsgIsService:    false,
		},
		text,
	}
}

func (msg *UIExpandedTextMessage) GetUIStringText() UIString {
	return msg.MsgUIStringText
}

// CopyFrom replaces the content of this message object with the content of the given object.
func (msg *UIExpandedTextMessage) CopyFrom(from ifc.MessageMeta) {
	msg.MsgSender = from.Sender()
	msg.MsgSenderColor = from.SenderColor()

	fromMsg, ok := from.(UIMessage)
	if ok {
		msg.MsgSender = fromMsg.RealSender()
		msg.MsgID = fromMsg.ID()
		msg.MsgType = fromMsg.Type()
		msg.MsgTimestamp = fromMsg.Timestamp()
		msg.MsgState = fromMsg.State()
		msg.MsgIsService = fromMsg.IsService()
		msg.MsgIsHighlight = fromMsg.IsHighlight()
		msg.buffer = nil

		fromExpandedMsg, ok := from.(*UIExpandedTextMessage)
		if ok {
			msg.MsgUIStringText = fromExpandedMsg.MsgUIStringText
		} else {
			msg.MsgUIStringText = NewColorUIString(fromMsg.Text(), from.TextColor())
		}

		msg.RecalculateBuffer()
	}
}

type UITextMessage struct {
	MsgID           string
	MsgType         string
	MsgSender       string
	MsgSenderColor  tcell.Color
	MsgTimestamp    time.Time
	MsgText         string
	MsgState        ifc.MessageState
	MsgIsHighlight  bool
	MsgIsService    bool
	buffer          []UIString
	prevBufferWidth int
}

// NewTextMessage creates a new UITextMessage object with the provided values and the default state.
func NewTextMessage(id, sender, msgtype, text string, timestamp time.Time) UIMessage {
	return &UITextMessage{
		MsgSender:       sender,
		MsgTimestamp:    timestamp,
		MsgSenderColor:  widget.GetHashColor(sender),
		MsgType:         msgtype,
		MsgText:         text,
		MsgID:           id,
		prevBufferWidth: 0,
		MsgState:        ifc.MessageStateDefault,
		MsgIsHighlight:  false,
		MsgIsService:    false,
	}
}

// CopyFrom replaces the content of this message object with the content of the given object.
func (msg *UITextMessage) CopyFrom(from ifc.MessageMeta) {
	msg.MsgSender = from.Sender()
	msg.MsgSenderColor = from.SenderColor()

	fromMsg, ok := from.(UIMessage)
	if ok {
		msg.MsgSender = fromMsg.RealSender()
		msg.MsgID = fromMsg.ID()
		msg.MsgType = fromMsg.Type()
		msg.MsgTimestamp = fromMsg.Timestamp()
		msg.MsgText = fromMsg.Text()
		msg.MsgState = fromMsg.State()
		msg.MsgIsService = fromMsg.IsService()
		msg.MsgIsHighlight = fromMsg.IsHighlight()
		msg.buffer = nil

		msg.RecalculateBuffer()
	}
}

// Sender gets the string that should be displayed as the sender of this message.
//
// If the message is being sent, the sender is "Sending...".
// If sending has failed, the sender is "Error".
// If the message is an emote, the sender is blank.
// In any other case, the sender is the display name of the user who sent the message.
func (msg *UITextMessage) Sender() string {
	switch msg.MsgState {
	case ifc.MessageStateSending:
		return "Sending..."
	case ifc.MessageStateFailed:
		return "Error"
	}
	switch msg.MsgType {
	case "m.emote":
		// Emotes don't show a separate sender, it's included in the buffer.
		return ""
	default:
		return msg.MsgSender
	}
}

func (msg *UITextMessage) RealSender() string {
	return msg.MsgSender
}

func (msg *UITextMessage) getStateSpecificColor() tcell.Color {
	switch msg.MsgState {
	case ifc.MessageStateSending:
		return tcell.ColorGray
	case ifc.MessageStateFailed:
		return tcell.ColorRed
	case ifc.MessageStateDefault:
		fallthrough
	default:
		return tcell.ColorDefault
	}
}

// SenderColor returns the color the name of the sender should be shown in.
//
// If the message is being sent, the color is gray.
// If sending has failed, the color is red.
//
// In any other case, the color is whatever is specified in the Message struct.
// Usually that means it is the hash-based color of the sender (see ui/widget/color.go)
func (msg *UITextMessage) SenderColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.MsgIsService:
		return tcell.ColorGray
	default:
		return msg.MsgSenderColor
	}
}

// TextColor returns the color the actual content of the message should be shown in.
func (msg *UITextMessage) TextColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.MsgIsService:
		return tcell.ColorGray
	case msg.MsgIsHighlight:
		return tcell.ColorYellow
	case msg.MsgType == "m.room.member":
		return tcell.ColorGreen
	default:
		return tcell.ColorDefault
	}
}

// TimestampColor returns the color the timestamp should be shown in.
//
// As with SenderColor(), messages being sent and messages that failed to be sent are
// gray and red respectively.
//
// However, other messages are the default color instead of a color stored in the struct.
func (msg *UITextMessage) TimestampColor() tcell.Color {
	return msg.getStateSpecificColor()
}

// RecalculateBuffer calculates the buffer again with the previously provided width.
func (msg *UITextMessage) RecalculateBuffer() {
	msg.CalculateBuffer(msg.prevBufferWidth)
}

// Buffer returns the computed text buffer.
//
// The buffer contains the text of the message split into lines with a maximum
// width of whatever was provided to CalculateBuffer().
//
// N.B. This will NOT automatically calculate the buffer if it hasn't been
//      calculated already, as that requires the target width.
func (msg *UITextMessage) Buffer() []UIString {
	return msg.buffer
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (msg *UITextMessage) Height() int {
	return len(msg.buffer)
}

// Timestamp returns the full timestamp when the message was sent.
func (msg *UITextMessage) Timestamp() time.Time {
	return msg.MsgTimestamp
}

// FormatTime returns the formatted time when the message was sent.
func (msg *UITextMessage) FormatTime() string {
	return msg.MsgTimestamp.Format(TimeFormat)
}

// FormatDate returns the formatted date when the message was sent.
func (msg *UITextMessage) FormatDate() string {
	return msg.MsgTimestamp.Format(DateFormat)
}

func (msg *UITextMessage) ID() string {
	return msg.MsgID
}

func (msg *UITextMessage) SetID(id string) {
	msg.MsgID = id
}

func (msg *UITextMessage) Type() string {
	return msg.MsgType
}

func (msg *UITextMessage) SetType(msgtype string) {
	msg.MsgType = msgtype
}

func (msg *UITextMessage) Text() string {
	return msg.MsgText
}

func (msg *UITextMessage) SetText(text string) {
	msg.MsgText = text
}

func (msg *UITextMessage) State() ifc.MessageState {
	return msg.MsgState
}

func (msg *UITextMessage) SetState(state ifc.MessageState) {
	msg.MsgState = state
}

func (msg *UITextMessage) IsHighlight() bool {
	return msg.MsgIsHighlight
}

func (msg *UITextMessage) SetIsHighlight(isHighlight bool) {
	msg.MsgIsHighlight = isHighlight
}

func (msg *UITextMessage) IsService() bool {
	return msg.MsgIsService
}

func (msg *UITextMessage) SetIsService(isService bool) {
	msg.MsgIsService = isService
}

func (msg *UITextMessage) GetUIStringText() UIString {
	return NewColorUIString(msg.Text(), msg.TextColor())
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
func (msg *UITextMessage) CalculateBuffer(width int) {
	if width < 2 {
		return
	}

	msg.buffer = []UIString{}
	text := msg.GetUIStringText()
	if msg.MsgType == "m.emote" {
		text = NewColorUIString(fmt.Sprintf("* %s %s", msg.MsgSender, text.String()), msg.TextColor())
		text.Colorize(2, len(msg.MsgSender), msg.SenderColor())
	}

	forcedLinebreaks := text.Split('\n')
	newlines := 0
	for _, str := range forcedLinebreaks {
		if len(str) == 0 && newlines < 1 {
			msg.buffer = append(msg.buffer, UIString{})
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
