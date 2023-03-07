// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

package ui

import (
	"time"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix/event"
)

type RosterView struct {
	mauview.Component

	selected *rooms.Room
	rooms    []*rooms.Room

	height, width int
	focused       bool

	parent *MainView
}

func NewRosterView(mainView *MainView) *RosterView {
	rstr := &RosterView{
		parent: mainView,
		rooms:  make([]*rooms.Room, 0),
	}

	return rstr
}

func (rstr *RosterView) Add(room *rooms.Room) {
	if room.IsReplaced() {
		return
	}
	insertAt := len(rstr.rooms)
	for i := 0; i < len(rstr.rooms); i++ {
		if rstr.rooms[i] == room {
			return
		} else if room.LastReceivedMessage.After(rstr.rooms[i].LastReceivedMessage) {
			insertAt = i
			break
		}
	}
	rstr.rooms = append(rstr.rooms, nil)
	copy(rstr.rooms[insertAt+1:], rstr.rooms[insertAt:len(rstr.rooms)-1])
	rstr.rooms[insertAt] = room
}

func (rstr *RosterView) Remove(room *rooms.Room) {
	index := rstr.index(room)
	if index < 0 || index > len(rstr.rooms) {
		return
	}
	last := len(rstr.rooms) - 1
	if index < last {
		copy(rstr.rooms[index:], rstr.rooms[index+1:])
	}
	rstr.rooms[last] = nil
	rstr.rooms = rstr.rooms[:last]
}

func (rstr *RosterView) index(room *rooms.Room) int {
	for index, entry := range rstr.rooms {
		if entry == room {
			return index
		}
	}
	return -1
}

func (rstr *RosterView) getMostRecentMessage(room *rooms.Room) (string, bool) {
	roomView, _ := rstr.parent.getRoomView(room.ID, true)

	if msgView := roomView.MessageView(); len(msgView.messages) < 20 && !msgView.initialHistoryLoaded {
		msgView.initialHistoryLoaded = true
		go rstr.parent.LoadHistory(room.ID)
	}

	if len(roomView.content.messages) > 0 {
		for index := len(roomView.content.messages) - 1; index >= 0; index-- {
			if roomView.content.messages[index].Type == event.MsgText {
				return roomView.content.messages[index].PlainText(), true
			}
		}
	}

	return "It's quite empty in here.", false
}

func (rstr *RosterView) Draw(screen mauview.Screen) {
	if rstr.focused {
		if roomView, ok := rstr.parent.getRoomView(rstr.selected.ID, true); ok {
			roomView.Update()
			roomView.Draw(screen)
			return
		}
	}

	rstr.width, rstr.height = screen.Size()

	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorDefault).Bold(true)
	mainStyle := titleStyle.Bold(false)

	now := time.Now()
	tm := now.Format("15:04")
	tmX := rstr.width - 3 - len(tm)

	// first line
	widget.WriteLine(screen, mauview.AlignLeft, "GOMUKS", 2, 1, tmX, titleStyle)
	widget.WriteLine(screen, mauview.AlignLeft, tm, tmX, 1, 2+len(tm), titleStyle)
	// second line
	widget.WriteLine(screen, mauview.AlignRight, now.Format("Mon, Jan 02"), 0, 2, rstr.width-3, mainStyle)
	// third line
	widget.NewBorder().Draw(mauview.NewProxyScreen(screen, 2, 3, rstr.width-5, 1))

	y := 4
	for _, room := range rstr.rooms {
		if room.IsReplaced() {
			continue
		}

		renderHeight := 2
		if y+renderHeight >= rstr.height {
			renderHeight = rstr.height - y
		}

		isSelected := room == rstr.selected

		style := tcell.StyleDefault.
			Foreground(tcell.ColorDefault).
			Bold(room.HasNewMessages())
		if isSelected {
			style = style.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorWhite)
		}

		timestamp := room.LastReceivedMessage
		tm := timestamp.Format("15:04")
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if timestamp.Before(today) {
			if timestamp.Before(today.AddDate(0, 0, -6)) {
				tm = timestamp.Format("2006-01-02")
			} else {
				tm = timestamp.Format("Monday")
			}
		}

		lastMessage, received := rstr.getMostRecentMessage(room)
		msgStyle := style.Foreground(tcell.ColorGray).Italic(!received)
		if isSelected {
			msgStyle = msgStyle.Background(tcell.ColorWhite)
		}

		tmX := rstr.width - 3 - len(tm)
		widget.WriteLinePadded(screen, mauview.AlignLeft, room.GetTitle(), 2, y, tmX, style)
		widget.WriteLine(screen, mauview.AlignLeft, tm, tmX, y, 2+len(tm), style)
		widget.WriteLinePadded(screen, mauview.AlignLeft, lastMessage, 2, y+1, rstr.width-5, msgStyle)

		y += renderHeight
		if y >= rstr.height {
			break
		}
	}
}

func (rstr *RosterView) OnKeyEvent(event mauview.KeyEvent) bool {
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}
	switch rstr.parent.config.Keybindings.Roster[kb] {
	case "next_room":
		if index := rstr.index(rstr.selected); index == -1 || index == len(rstr.rooms)-1 {
			rstr.selected = rstr.First()
		} else {
			rstr.selected = rstr.rooms[index+1]
		}
	case "prev_room":
		if index := rstr.index(rstr.selected); index < 1 {
			rstr.selected = rstr.Last()
		} else {
			rstr.selected = rstr.rooms[index-1]
		}
	case "clear":
		rstr.focused = false
		rstr.selected = nil
	case "quit":
		rstr.parent.gmx.Stop(true)
	case "enter":
		rstr.focused = rstr.selected != nil
	default:
		return false
	}
	return true
}

func (rstr *RosterView) First() *rooms.Room {
	return rstr.rooms[0]
}

func (rstr *RosterView) Last() *rooms.Room {
	return rstr.rooms[len(rstr.rooms)-1]
}
