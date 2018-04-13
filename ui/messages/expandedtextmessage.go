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
	"time"

	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

func init() {
	gob.Register(&ExpandedTextMessage{})
}

type ExpandedTextMessage struct {
	BaseTextMessage
	MsgText tstring.TString
}

// NewExpandedTextMessage creates a new ExpandedTextMessage object with the provided values and the default state.
func NewExpandedTextMessage(id, sender, msgtype string, text tstring.TString, timestamp time.Time) UIMessage {
	return &ExpandedTextMessage{
		BaseTextMessage: newBaseTextMessage(id, sender, msgtype, timestamp),
		MsgText: text,
	}
}

func (msg *ExpandedTextMessage) GenerateText() tstring.TString {
	return msg.MsgText
}

// CopyFrom replaces the content of this message object with the content of the given object.
func (msg *ExpandedTextMessage) CopyFrom(from ifc.MessageMeta) {
	msg.BaseTextMessage.CopyFrom(from)

	fromExpandedMsg, ok := from.(*ExpandedTextMessage)
	if ok {
		msg.MsgText = fromExpandedMsg.MsgText
	}

	msg.RecalculateBuffer()
}

func (msg *ExpandedTextMessage) NotificationContent() string {
	return msg.MsgText.String()
}

func (msg *ExpandedTextMessage) CalculateBuffer(width int) {
	msg.BaseTextMessage.calculateBufferWithText(msg.MsgText, width)
}

// RecalculateBuffer calculates the buffer again with the previously provided width.
func (msg *ExpandedTextMessage) RecalculateBuffer() {
	msg.CalculateBuffer(msg.prevBufferWidth)
}
