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
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomatrix"
	"maunium.net/go/tview"
)

type RoomView struct {
	*tview.Box

	topic    *tview.TextView
	content  *MessageView
	status   *tview.TextView
	userList *tview.TextView
	room     *gomatrix.Room

	parent *MainView
}

var colorNames []string

func init() {
	colorNames = make([]string, len(tcell.ColorNames))
	i := 0
	for name, _ := range tcell.ColorNames {
		colorNames[i] = name
		i++
	}
	sort.Sort(sort.StringSlice(colorNames))
}

func NewRoomView(parent *MainView, room *gomatrix.Room) *RoomView {
	view := &RoomView{
		Box:      tview.NewBox(),
		topic:    tview.NewTextView(),
		content:  NewMessageView(),
		status:   tview.NewTextView(),
		userList: tview.NewTextView(),
		room:     room,
		parent:   parent,
	}
	view.topic.
		SetText(strings.Replace(room.GetTopic(), "\n", " ", -1)).
		SetBackgroundColor(tcell.ColorDarkGreen)
	view.status.SetBackgroundColor(tcell.ColorDimGray)
	view.userList.SetDynamicColors(true)
	return view
}

func (view *RoomView) Draw(screen tcell.Screen) {
	view.Box.Draw(screen)

	x, y, width, height := view.GetRect()
	view.topic.SetRect(x, y, width, 1)
	view.content.SetRect(x, y+1, width-30, height-2)
	view.status.SetRect(x, y+height-1, width, 1)
	view.userList.SetRect(x+width-29, y+1, 29, height-2)

	view.topic.Draw(screen)
	view.content.Draw(screen)
	view.status.Draw(screen)

	borderX := x + width - 30
	background := tcell.StyleDefault.Background(view.GetBackgroundColor()).Foreground(view.GetBorderColor())
	for borderY := y + 1; borderY < y+height-1; borderY++ {
		screen.SetContent(borderX, borderY, tview.GraphicsVertBar, nil, background)
	}
	view.userList.Draw(screen)
}

func (view *RoomView) SetTyping(users []string) {
	for index, user := range users {
		member := view.room.GetMember(user)
		if member != nil {
			users[index] = member.DisplayName
		}
	}
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

func (view *RoomView) AutocompleteUser(existingText string) (completions []string) {
	for _, user := range view.room.GetMembers() {
		if strings.HasPrefix(user.DisplayName, existingText) {
			completions = append(completions, user.DisplayName)
		} else if strings.HasPrefix(user.UserID, existingText) {
			completions = append(completions, user.UserID)
		}
	}
	return
}

func (view *RoomView) MessageView() *MessageView {
	return view.content
}

func getColorName(s string) string {
	switch s {
	case "-->":
		return "green"
	case "<--":
		return "red"
	case "---":
		return "yellow"
	default:
		h := fnv.New32a()
		h.Write([]byte(s))
		return colorNames[int(h.Sum32())%len(colorNames)]
	}
}

func getColor(s string) tcell.Color {
	return tcell.ColorNames[getColorName(s)]
}

func color(s string) string {
	return fmt.Sprintf("[%s]%s[white]", getColorName(s), s)
}

func (view *RoomView) UpdateUserList() {
	var joined strings.Builder
	var invited strings.Builder
	for _, user := range view.room.GetMembers() {
		if user.Membership == "join" {
			joined.WriteString(color(user.DisplayName))
			joined.WriteRune('\n')
		} else if user.Membership == "invite" {
			invited.WriteString(color(user.DisplayName))
			invited.WriteRune('\n')
		}
	}
	view.userList.Clear()
	fmt.Fprintf(view.userList, "%s\n", joined.String())
	if invited.Len() > 0 {
		fmt.Fprintf(view.userList, "\nInvited:\n%s", invited.String())
	}
}

func (view *RoomView) AddMessage(id, sender, message string, timestamp time.Time) {
	member := view.room.GetMember(sender)
	if member != nil {
		sender = member.DisplayName
	}
	view.content.AddMessage(id, sender, message, timestamp)
	view.parent.Render()
}
