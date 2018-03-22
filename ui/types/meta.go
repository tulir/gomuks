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
	"github.com/gdamore/tcell"
)

// MessageMeta is an interface to get the metadata of a message.
//
// See BasicMeta for a simple implementation and documentation of methods.
type MessageMeta interface {
	GetSender() string
	GetSenderColor() tcell.Color
	GetTextColor() tcell.Color
	GetTimestampColor() tcell.Color
	GetTimestamp() string
	GetDate() string
}

// BasicMeta is a simple variable store implementation of MessageMeta.
type BasicMeta struct {
	Sender, Timestamp, Date                string
	SenderColor, TextColor, TimestampColor tcell.Color
}

// GetSender gets the string that should be displayed as the sender of this message.
func (meta *BasicMeta) GetSender() string {
	return meta.Sender
}

// GetSenderColor returns the color the name of the sender should be shown in.
func (meta *BasicMeta) GetSenderColor() tcell.Color {
	return meta.SenderColor
}

// GetTimestamp returns the formatted time when the message was sent.
func (meta *BasicMeta) GetTimestamp() string {
	return meta.Timestamp
}

// GetDate returns the formatted date when the message was sent.
func (meta *BasicMeta) GetDate() string {
	return meta.Date
}

// GetTextColor returns the color the actual content of the message should be shown in.
func (meta *BasicMeta) GetTextColor() tcell.Color {
	return meta.TextColor
}

// GetTimestampColor returns the color the timestamp should be shown in.
//
// This usually does not apply to the date, as it is rendered separately from the message.
func (meta *BasicMeta) GetTimestampColor() tcell.Color {
	return meta.TimestampColor
}
