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
	"maunium.net/go/gomuks/ui/widget"
)

func init() {
	gob.Register(&UITextMessage{})
	gob.Register(&UIExpandedTextMessage{})
}

type UIExpandedTextMessage struct {
	UITextMessage
	MsgTStringText tstring.TString
}

// NewExpandedTextMessage creates a new UIExpandedTextMessage object with the provided values and the default state.
func NewExpandedTextMessage(id, sender, msgtype string, text tstring.TString, timestamp time.Time) UIMessage {
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

func (msg *UIExpandedTextMessage) GetTStringText() tstring.TString {
	return msg.MsgTStringText
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
			msg.MsgTStringText = fromExpandedMsg.MsgTStringText
		} else {
			msg.MsgTStringText = tstring.NewColorTString(fromMsg.Text(), from.TextColor())
		}

		msg.RecalculateBuffer()
	}
}
