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
	"fmt"

	"maunium.net/go/gomuks/matrix/event"
	"maunium.net/go/mauview"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type VideoMessage struct {
	Body       string
	Homeserver string
	FileID     string
	buffer     []tstring.TString

	matrix ifc.MatrixContainer
}

// NewVideoMessage creates a new VideoMessage object with the provided values and the default state.
func NewVideoMessage(matrix ifc.MatrixContainer, evt *event.Event, displayname string, body, homeserver, fileID string) *UIMessage {
	return newUIMessage(evt, displayname, &VideoMessage{
		Body:       body,
		Homeserver: homeserver,
		FileID:     fileID,
		matrix:     matrix,
	})
}

func (msg *VideoMessage) Clone() MessageRenderer {
	return &VideoMessage{
		Body:       msg.Body,
		Homeserver: msg.Homeserver,
		FileID:     msg.FileID,
		matrix:     msg.matrix,
	}
}

func (msg *VideoMessage) NotificationContent() string {
	return "Sent a video"
}

func (msg *VideoMessage) PlainText() string {
	return fmt.Sprintf("%s: %s", msg.Body, msg.matrix.GetDownloadURL(msg.Homeserver, msg.FileID))
}

func (msg *VideoMessage) String() string {
	return fmt.Sprintf(`&messages.VideoMessage{Body="%s", Homeserver="%s", FileID="%s"}`, msg.Body, msg.Homeserver, msg.FileID)
}

func (msg *VideoMessage) Path() string {
	return msg.matrix.GetCachePath(msg.Homeserver, msg.FileID)
}

func (msg *VideoMessage) RegisterMatrix(matrix ifc.MatrixContainer) {
	msg.matrix = matrix
}

// Print only Plain Text
func (msg *VideoMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	msg.buffer = calculateBufferWithText(prefs, tstring.NewTString(msg.PlainText()), width, uiMsg)
	return
}

func (msg *VideoMessage) Height() int {
	return len(msg.buffer)
}

func (msg *VideoMessage) Draw(screen mauview.Screen) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
