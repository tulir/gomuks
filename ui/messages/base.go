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
	"maunium.net/go/mautrix"
	"time"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
)

func init() {
	gob.Register(&BaseMessage{})
}

type BaseMessage struct {
	MsgID           string
	MsgType         mautrix.MessageType
	MsgSenderID     string
	MsgSender       string
	MsgSenderColor  tcell.Color
	MsgTimestamp    time.Time
	MsgState        ifc.MessageState
	MsgIsHighlight  bool
	MsgIsService    bool
	buffer          []tstring.TString
	plainBuffer     []tstring.TString
	prevBufferWidth int
	prevPrefs       config.UserPreferences
}

func newBaseMessage(id, sender, displayname string, msgtype mautrix.MessageType, timestamp time.Time) BaseMessage {
	return BaseMessage{
		MsgSenderID:     sender,
		MsgSender:       displayname,
		MsgTimestamp:    timestamp,
		MsgSenderColor:  widget.GetHashColor(sender),
		MsgType:         msgtype,
		MsgID:           id,
		prevBufferWidth: 0,
		MsgState:        ifc.MessageStateDefault,
		MsgIsHighlight:  false,
		MsgIsService:    false,
	}
}

func (msg *BaseMessage) RegisterMatrix(matrix ifc.MatrixContainer) {}

// Sender gets the string that should be displayed as the sender of this message.
//
// If the message is being sent, the sender is "Sending...".
// If sending has failed, the sender is "Error".
// If the message is an emote, the sender is blank.
// In any other case, the sender is the display name of the user who sent the message.
func (msg *BaseMessage) Sender() string {
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

func (msg *BaseMessage) SenderID() string {
	return msg.MsgSenderID
}

func (msg *BaseMessage) RealSender() string {
	return msg.MsgSender
}

func (msg *BaseMessage) getStateSpecificColor() tcell.Color {
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
func (msg *BaseMessage) SenderColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.MsgType == "m.room.member":
		return widget.GetHashColor(msg.MsgSender)
	case msg.MsgIsService:
		return tcell.ColorGray
	default:
		return msg.MsgSenderColor
	}
}

// TextColor returns the color the actual content of the message should be shown in.
func (msg *BaseMessage) TextColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.MsgIsService, msg.MsgType == "m.notice":
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
func (msg *BaseMessage) TimestampColor() tcell.Color {
	return msg.getStateSpecificColor()
}

// Buffer returns the computed text buffer.
//
// The buffer contains the text of the message split into lines with a maximum
// width of whatever was provided to CalculateBuffer().
//
// N.B. This will NOT automatically calculate the buffer if it hasn't been
//      calculated already, as that requires the target width.
func (msg *BaseMessage) Buffer() []tstring.TString {
	return msg.buffer
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (msg *BaseMessage) Height() int {
	return len(msg.buffer)
}

// Timestamp returns the full timestamp when the message was sent.
func (msg *BaseMessage) Timestamp() time.Time {
	return msg.MsgTimestamp
}

// FormatTime returns the formatted time when the message was sent.
func (msg *BaseMessage) FormatTime() string {
	return msg.MsgTimestamp.Format(TimeFormat)
}

// FormatDate returns the formatted date when the message was sent.
func (msg *BaseMessage) FormatDate() string {
	return msg.MsgTimestamp.Format(DateFormat)
}

func (msg *BaseMessage) ID() string {
	return msg.MsgID
}

func (msg *BaseMessage) SetID(id string) {
	msg.MsgID = id
}

func (msg *BaseMessage) Type() mautrix.MessageType {
	return msg.MsgType
}

func (msg *BaseMessage) SetType(msgtype mautrix.MessageType) {
	msg.MsgType = msgtype
}

func (msg *BaseMessage) State() ifc.MessageState {
	return msg.MsgState
}

func (msg *BaseMessage) SetState(state ifc.MessageState) {
	msg.MsgState = state
}

func (msg *BaseMessage) IsHighlight() bool {
	return msg.MsgIsHighlight
}

func (msg *BaseMessage) SetIsHighlight(isHighlight bool) {
	msg.MsgIsHighlight = isHighlight
}

func (msg *BaseMessage) IsService() bool {
	return msg.MsgIsService
}

func (msg *BaseMessage) SetIsService(isService bool) {
	msg.MsgIsService = isService
}
