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

type MessageMeta interface {
	GetSender() string
	GetSenderColor() tcell.Color
	GetTextColor() tcell.Color
	GetTimestampColor() tcell.Color
	GetTimestamp() string
	GetDate() string
}

type BasicMeta struct {
	Sender, Timestamp, Date                string
	SenderColor, TextColor, TimestampColor tcell.Color
}

func (meta *BasicMeta) GetSender() string {
	return meta.Sender
}

func (meta *BasicMeta) GetSenderColor() tcell.Color {
	return meta.SenderColor
}

func (meta *BasicMeta) GetTimestamp() string {
	return meta.Timestamp
}

func (meta *BasicMeta) GetDate() string {
	return meta.Date
}

func (meta *BasicMeta) GetTextColor() tcell.Color {
	return meta.TextColor
}

func (meta *BasicMeta) GetTimestampColor() tcell.Color {
	return meta.TimestampColor
}
