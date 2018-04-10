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
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/interface"
)

// BasicMeta is a simple variable store implementation of MessageMeta.
type BasicMeta struct {
	BSender string
	BTimestamp time.Time
	BSenderColor, BTextColor, BTimestampColor tcell.Color
}

// Sender gets the string that should be displayed as the sender of this message.
func (meta *BasicMeta) Sender() string {
	return meta.BSender
}

// SenderColor returns the color the name of the sender should be shown in.
func (meta *BasicMeta) SenderColor() tcell.Color {
	return meta.BSenderColor
}

// Timestamp returns the full time when the message was sent.
func (meta *BasicMeta) Timestamp() time.Time {
	return meta.BTimestamp
}

// FormatTime returns the formatted time when the message was sent.
func (meta *BasicMeta) FormatTime() string {
	return meta.BTimestamp.Format(TimeFormat)
}

// FormatDate returns the formatted date when the message was sent.
func (meta *BasicMeta) FormatDate() string {
	return meta.BTimestamp.Format(DateFormat)
}

// TextColor returns the color the actual content of the message should be shown in.
func (meta *BasicMeta) TextColor() tcell.Color {
	return meta.BTextColor
}

// TimestampColor returns the color the timestamp should be shown in.
//
// This usually does not apply to the date, as it is rendered separately from the message.
func (meta *BasicMeta) TimestampColor() tcell.Color {
	return meta.BTimestampColor
}

// CopyFrom replaces the content of this meta object with the content of the given object.
func (meta *BasicMeta) CopyFrom(from ifc.MessageMeta) {
	meta.BSender = from.Sender()
	meta.BTimestamp = from.Timestamp()
	meta.BSenderColor = from.SenderColor()
	meta.BTextColor = from.TextColor()
	meta.BTimestampColor = from.TimestampColor()
}
