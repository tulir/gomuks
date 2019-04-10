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
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"
)

// UIMessage is a wrapper for the content and metadata of a Matrix message intended to be displayed.
type UIMessage interface {
	ifc.Message

	Type() mautrix.MessageType
	Sender() string
	SenderColor() tcell.Color
	TextColor() tcell.Color
	TimestampColor() tcell.Color
	FormatTime() string
	FormatDate() string
	SameDate(message UIMessage) bool

	SetReplyTo(message UIMessage)
	CalculateBuffer(preferences config.UserPreferences, width int)
	Draw(screen mauview.Screen)
	Height() int
	PlainText() string

	Clone() UIMessage

	RealSender() string
	RegisterMatrix(matrix ifc.MatrixContainer)
}

const DateFormat = "January _2, 2006"
const TimeFormat = "15:04:05"
