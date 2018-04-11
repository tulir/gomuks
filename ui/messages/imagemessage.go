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
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"image/color"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/ansimage"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
)

func init() {
	gob.Register(&UIImageMessage{})
}

type UIImageMessage struct {
	UITextMessage
	Homeserver string
	FileID     string
	data       []byte

	gmx ifc.Gomuks
}

// NewImageMessage creates a new UIImageMessage object with the provided values and the default state.
func NewImageMessage(gmx ifc.Gomuks, id, sender, msgtype, homeserver, fileID string, data []byte, timestamp time.Time) UIMessage {
	return &UIImageMessage{
		UITextMessage{
			MsgSender:       sender,
			MsgTimestamp:    timestamp,
			MsgSenderColor:  widget.GetHashColor(sender),
			MsgType:         msgtype,
			MsgID:           id,
			prevBufferWidth: 0,
			MsgState:        ifc.MessageStateDefault,
			MsgIsHighlight:  false,
			MsgIsService:    false,
		},
		homeserver,
		fileID,
		data,
		gmx,
	}
}

func (msg *UIImageMessage) RegisterGomuks(gmx ifc.Gomuks) {
	msg.gmx = gmx

	debug.Print(len(msg.data), msg.data)
	if len(msg.data) == 0 {
		go func() {
			defer gmx.Recover()
			msg.updateData()
		}()
	}
}

func (msg *UIImageMessage) updateData() {
	debug.Print("Loading image:", msg.Homeserver, msg.FileID)
	data, _, _, err := msg.gmx.Matrix().Download(fmt.Sprintf("mxc://%s/%s", msg.Homeserver, msg.FileID))
	if err != nil {
		debug.Print("Failed to download image %s/%s: %v", msg.Homeserver, msg.FileID, err)
		return
	}
	msg.data = data
}

func (msg *UIImageMessage) Path() string {
	return msg.gmx.Matrix().GetCachePath(msg.Homeserver, msg.FileID)
}

// CopyFrom replaces the content of this message object with the content of the given object.
func (msg *UIImageMessage) CopyFrom(from ifc.MessageMeta) {
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

		fromImgMsg, ok := from.(*UIImageMessage)
		if ok {
			msg.data = fromImgMsg.data
		}

		msg.RecalculateBuffer()
	}
}

// CalculateBuffer generates the internal buffer for this message that consists
// of the text of this message split into lines at most as wide as the width
// parameter.
func (msg *UIImageMessage) CalculateBuffer(width int) {
	if width < 2 {
		return
	}

	image, err := ansimage.NewScaledFromReader(bytes.NewReader(msg.data), 0, width, color.Black)
	if err != nil {
		msg.buffer = []tstring.TString{tstring.NewColorTString("Failed to display image", tcell.ColorRed)}
		debug.Print("Failed to display image:", err)
		return
	}

	msg.buffer = image.Render()
	msg.prevBufferWidth = width
}
