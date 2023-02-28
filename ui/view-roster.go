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
	"strings"
	"time"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
)

type RosterView struct {
	mauview.Component

	selected *rooms.Room
	rooms    []*rooms.Room

	height, width int

	// The item main text color.
	mainTextColor tcell.Color
	// The text color for selected items.
	selectedTextColor tcell.Color
	// The background color for selected items.
	selectedBackgroundColor tcell.Color

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

func (rstr *RosterView) Draw(screen mauview.Screen) {
	rstr.width, rstr.height = screen.Size()

	titleStyle := tcell.StyleDefault.Foreground(rstr.mainTextColor).Bold(true)
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
	widget.WriteLine(
		screen, mauview.AlignCenter,
		strings.Repeat(string(mauview.BoxDrawingsLightHorizontal), rstr.width-5),
		2, 3, rstr.width-2,
		mainStyle,
	)

	y := 4
	for _, room := range rstr.rooms {
		if room.IsReplaced() {
			continue
		}

		renderHeight := 1
		if y+renderHeight >= rstr.height {
			renderHeight = rstr.height - y
		}

		isSelected := room == rstr.selected

		style := tcell.StyleDefault.
			Foreground(rstr.mainTextColor).
			Bold(room.HasNewMessages())
		if isSelected {
			style = style.
				Foreground(rstr.selectedTextColor).
				Background(rstr.selectedBackgroundColor)
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

		tmX := rstr.width - 3 - len(tm)
		widget.WriteLine(screen, mauview.AlignLeft, room.GetTitle(), 2, y, tmX, style)
		widget.WriteLine(screen, mauview.AlignLeft, tm, tmX, y, 2+len(tm), style)

		y += renderHeight
		if y >= rstr.height {
			break
		}
	}
}
