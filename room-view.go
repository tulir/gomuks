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

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell"
	"maunium.net/go/tview"
)

type RoomView struct {
	*tview.Box

	topic    *tview.TextView
	content  *tview.TextView
	status   *tview.TextView
	userlist *tview.TextView
	users    sort.StringSlice
}

func NewRoomView(topic string) *RoomView {
	view := &RoomView{
		Box:      tview.NewBox(),
		topic:    tview.NewTextView(),
		content:  tview.NewTextView(),
		status:   tview.NewTextView(),
		userlist: tview.NewTextView(),
	}
	view.topic.
		SetText(strings.Replace(topic, "\n", " ", -1)).
		SetBackgroundColor(tcell.ColorDarkGreen)
	view.status.SetBackgroundColor(tcell.ColorDimGray)
	return view
}

func (view *RoomView) Draw(screen tcell.Screen) {
	x, y, width, height := view.GetRect()
	view.topic.SetRect(x, y, width, 1)
	view.content.SetRect(x, y+1, width-30, height-2)
	view.status.SetRect(x, y+height-1, width, 1)
	view.userlist.SetRect(x+width-29, y+1, 29, height-2)

	view.topic.Draw(screen)
	view.content.Draw(screen)
	view.status.Draw(screen)

	borderX := x + width - 30
	background := tcell.StyleDefault.Background(view.GetBackgroundColor()).Foreground(view.GetBorderColor())
	for borderY := y + 1; borderY < y+height-1; borderY++ {
		screen.SetContent(borderX, borderY, tview.GraphicsVertBar, nil, background)
	}
	view.userlist.Draw(screen)
}

func (view *RoomView) SetTyping(users []string) {
	if len(users) == 0 {
		view.status.SetText("")
	} else if len(users) < 2 {
		view.status.SetText("Typing: " + strings.Join(users, " and "))
	} else {
		view.status.SetText(fmt.Sprintf(
			"Typing: %s and %s",
			strings.Join(users[:len(users)-1], ", "), users[len(users)-1]))
	}
}

func (view *RoomView) AddMessage(sender, message string) {
	fmt.Fprintf(view.content, "<%s> %s\n", sender, message)
}

func (view *RoomView) SetUsers(users []string) {
	view.users = sort.StringSlice(users)
	view.users.Sort()
	view.userlist.SetText(strings.Join(view.users, "\n"))
}

func (view *RoomView) RemoveUser(user string) {
	i := view.users.Search(user)
	if i >= 0 {
		view.users = append(view.users[:i], view.users[i+1:]...)
		view.userlist.SetText(strings.Join(view.users, "\n"))
	}
}

func (view *RoomView) AddUser(user string) {
	view.SetUsers(append(view.users, user))
}
