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
	"time"

	"image/color"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/pixterm/ansimage"
)

func init() {
	gob.Register(&UIImageMessage{})
}

type UIImageMessage struct {
	UITextMessage
	data []byte
}

// NewImageMessage creates a new UIImageMessage object with the provided values and the default state.
func NewImageMessage(id, sender, msgtype string, data []byte, timestamp time.Time) UIMessage {
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
		data,
	}
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

	image, err := ansimage.NewScaledFromReader(bytes.NewReader(msg.data), -1, width, color.Black, ansimage.ScaleModeResize, ansimage.NoDithering)
	if err != nil {
		msg.buffer = []UIString{NewColorUIString("Failed to display image", tcell.ColorRed)}
		debug.Print("Failed to display image:", err)
		return
	}

	msg.buffer = make([]UIString, image.Height())
	pixels := image.Pixmap()
	for row, pixelRow := range pixels {
		msg.buffer[row] = make(UIString, len(pixelRow))
		for column, pixel := range pixelRow {
			pixelColor := tcell.NewRGBColor(int32(pixel.R), int32(pixel.G), int32(pixel.B))
			msg.buffer[row][column] = Cell{
				Char: ' ',
				Style: tcell.StyleDefault.Background(pixelColor),
			}
		}
	}
	msg.prevBufferWidth = width
}
