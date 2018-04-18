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
	"time"

	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

func init() {
	gob.Register(&TextMessage{})
}

type TextMessage struct {
	BaseTextMessage
	cache   tstring.TString
	MsgText string
}

// NewTextMessage creates a new UITextMessage object with the provided values and the default state.
func NewTextMessage(id, sender, displayname, msgtype, text string, timestamp time.Time) UIMessage {
	return &TextMessage{
		BaseTextMessage: newBaseTextMessage(id, sender, displayname, msgtype, timestamp),
		MsgText:         text,
	}
}

func (msg *TextMessage) getCache() tstring.TString {
	if msg.cache == nil {
		switch msg.MsgType {
		case "m.emote":
			msg.cache = tstring.NewColorTString(fmt.Sprintf("* %s %s", msg.MsgSender, msg.MsgText), msg.TextColor())
			msg.cache.Colorize(0, len(msg.MsgSender)+2, msg.SenderColor())
		default:
			msg.cache = tstring.NewColorTString(msg.MsgText, msg.TextColor())
		}
	}
	return msg.cache
}

// CopyFrom replaces the content of this message object with the content of the given object.
func (msg *TextMessage) CopyFrom(from ifc.MessageMeta) {
	msg.BaseTextMessage.CopyFrom(from)

	fromTextMsg, ok := from.(*TextMessage)
	if ok {
		msg.MsgText = fromTextMsg.MsgText
	}

	msg.cache = nil
	msg.RecalculateBuffer()
}
func (msg *TextMessage) SetType(msgtype string) {
	msg.BaseTextMessage.SetType(msgtype)
	msg.cache = nil
}

func (msg *TextMessage) SetState(state ifc.MessageState) {
	msg.BaseTextMessage.SetState(state)
	msg.cache = nil
}

func (msg *TextMessage) SetIsHighlight(isHighlight bool) {
	msg.BaseTextMessage.SetIsHighlight(isHighlight)
	msg.cache = nil
}

func (msg *TextMessage) SetIsService(isService bool) {
	msg.BaseTextMessage.SetIsService(isService)
	msg.cache = nil
}

func (msg *TextMessage) NotificationContent() string {
	return msg.MsgText
}

func (msg *TextMessage) CalculateBuffer(width int) {
	msg.BaseTextMessage.calculateBufferWithText(msg.getCache(), width)
}

// RecalculateBuffer calculates the buffer again with the previously provided width.
func (msg *TextMessage) RecalculateBuffer() {
	msg.CalculateBuffer(msg.prevBufferWidth)
}
