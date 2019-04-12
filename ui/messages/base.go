// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package messages

import (
	"encoding/json"
	"fmt"
	"time"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
)

type BaseMessage struct {
	MsgID          string
	MsgTxnID       string
	MsgType        mautrix.MessageType
	MsgSenderID    string
	MsgSender      string
	MsgSenderColor tcell.Color
	MsgTimestamp   time.Time
	MsgState       mautrix.OutgoingEventState
	MsgIsHighlight bool
	MsgIsService   bool
	MsgSource      json.RawMessage
	ReplyTo        UIMessage
	buffer         []tstring.TString
}

func newBaseMessage(event *mautrix.Event, displayname string) BaseMessage {
	msgtype := event.Content.MsgType
	if len(msgtype) == 0 {
		msgtype = mautrix.MessageType(event.Type.String())
	}

	return BaseMessage{
		MsgSenderID:    event.Sender,
		MsgSender:      displayname,
		MsgTimestamp:   unixToTime(event.Timestamp),
		MsgSenderColor: widget.GetHashColor(event.Sender),
		MsgType:        msgtype,
		MsgID:          event.ID,
		MsgTxnID:       event.Unsigned.TransactionID,
		MsgState:       event.Unsigned.OutgoingState,
		MsgIsHighlight: false,
		MsgIsService:   false,
		MsgSource:      event.Content.VeryRaw,
	}
}

func unixToTime(unix int64) time.Time {
	timestamp := time.Now()
	if unix != 0 {
		timestamp = time.Unix(unix/1000, unix%1000*1000)
	}
	return timestamp
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
	case mautrix.EventStateLocalEcho:
		return "Sending..."
	case mautrix.EventStateSendFail:
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

func (msg *BaseMessage) NotificationSenderName() string {
	return msg.MsgSender
}

func (msg *BaseMessage) getStateSpecificColor() tcell.Color {
	switch msg.MsgState {
	case mautrix.EventStateLocalEcho:
		return tcell.ColorGray
	case mautrix.EventStateSendFail:
		return tcell.ColorRed
	case mautrix.EventStateDefault:
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
	if msg.MsgIsService {
		return tcell.ColorGray
	}
	return msg.getStateSpecificColor()
}

func (msg *BaseMessage) ReplyHeight() int {
	if msg.ReplyTo != nil {
		return 2 + msg.ReplyTo.Height()
	}
	return 0
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (msg *BaseMessage) Height() int {
	return msg.ReplyHeight() + len(msg.buffer)
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

func (msg *BaseMessage) SameDate(message UIMessage) bool {
	year1, month1, day1 := msg.Timestamp().Date()
	year2, month2, day2 := message.Timestamp().Date()
	return day1 == day2 && month1 == month2 && year1 == year2
}

func (msg *BaseMessage) ID() string {
	if len(msg.MsgID) == 0 {
		return msg.MsgTxnID
	}
	return msg.MsgID
}

func (msg *BaseMessage) SetID(id string) {
	msg.MsgID = id
}

func (msg *BaseMessage) TxnID() string {
	return msg.MsgTxnID
}

func (msg *BaseMessage) Type() mautrix.MessageType {
	return msg.MsgType
}

func (msg *BaseMessage) State() mautrix.OutgoingEventState {
	return msg.MsgState
}

func (msg *BaseMessage) SetState(state mautrix.OutgoingEventState) {
	msg.MsgState = state
}

func (msg *BaseMessage) IsHighlight() bool {
	return msg.MsgIsHighlight
}

func (msg *BaseMessage) SetIsHighlight(isHighlight bool) {
	msg.MsgIsHighlight = isHighlight
}

func (msg *BaseMessage) Source() json.RawMessage {
	return msg.MsgSource
}

func (msg *BaseMessage) SetReplyTo(event UIMessage) {
	msg.ReplyTo = event
}

func (msg *BaseMessage) Draw(screen mauview.Screen) {
	screen = msg.DrawReply(screen)
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}

func (msg *BaseMessage) clone() BaseMessage {
	clone := *msg
	clone.buffer = nil
	return clone
}

func (msg *BaseMessage) CalculateReplyBuffer(preferences config.UserPreferences, width int) {
	if msg.ReplyTo == nil {
		return
	}
	msg.ReplyTo.CalculateBuffer(preferences, width-1)
}

func (msg *BaseMessage) DrawReply(screen mauview.Screen) mauview.Screen {
	if msg.ReplyTo == nil {
		return screen
	}
	width, height := screen.Size()
	replyHeight := msg.ReplyTo.Height()
	widget.WriteLineSimpleColor(screen, "In reply to", 0, 0, tcell.ColorGreen)
	widget.WriteLineSimpleColor(screen, msg.ReplyTo.RealSender(), len("In reply to "), 0, msg.ReplyTo.SenderColor())
	for y := 1; y < 1+replyHeight; y++ {
		screen.SetCell(0, y, tcell.StyleDefault, '▋')
	}
	replyScreen := mauview.NewProxyScreen(screen, 1, 1, width-1, replyHeight)
	msg.ReplyTo.Draw(replyScreen)
	return mauview.NewProxyScreen(screen, 0, replyHeight+2, width, height-replyHeight-2)
}

func (msg *BaseMessage) String() string {
	return fmt.Sprintf(`&messages.BaseMessage{
    ID="%s", TxnID="%s",
    Type="%s", Timestamp=%s,
    Sender={ID="%s", Name="%s", Color=#%X},
    IsService=%t, IsHighlight=%t,
}`,
		msg.MsgID, msg.MsgTxnID,
		msg.MsgType, msg.MsgTimestamp.String(),
		msg.MsgSenderID, msg.MsgSender, msg.MsgSenderColor.Hex(),
		msg.MsgIsService, msg.MsgIsHighlight,
	)
}
