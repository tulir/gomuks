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

package ui

import (
	"fmt"
	"strconv"

	"maunium.net/go/tcell"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

type RoomList struct {
	*tview.Box

	indices  map[*rooms.Room]int
	items    []*rooms.Room
	selected *rooms.Room

	// The item main text color.
	mainTextColor tcell.Color
	// The text color for selected items.
	selectedTextColor tcell.Color
	// The background color for selected items.
	selectedBackgroundColor tcell.Color
}

func NewRoomList() *RoomList {
	return &RoomList{
		Box:     tview.NewBox(),
		indices: make(map[*rooms.Room]int),
		items:   []*rooms.Room{},

		mainTextColor:           tcell.ColorWhite,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
}

func (list *RoomList) Add(room *rooms.Room) {
	list.indices[room] = len(list.items)
	list.items = append(list.items, room)
	if list.selected == nil {
		list.selected = room
	}
}

func (list *RoomList) Remove(room *rooms.Room) {
	index, ok := list.indices[room]
	if !ok {
		return
	}
	delete(list.indices, room)
	list.items = append(list.items[0:index], list.items[index+1:]...)
	if len(list.items) == 0 {
		list.selected = nil
	}
}

func (list *RoomList) Clear() {
	list.indices = make(map[*rooms.Room]int)
	list.items = []*rooms.Room{}
	list.selected = nil
}

func (list *RoomList) SetSelected(room *rooms.Room) {
	list.selected = room
}

// Draw draws this primitive onto the screen.
func (list *RoomList) Draw(screen tcell.Screen) {
	list.Box.Draw(screen)

	x, y, width, height := list.GetInnerRect()
	bottomLimit := y + height

	var offset int
	currentItemIndex, hasSelected := list.indices[list.selected]
	if hasSelected && currentItemIndex >= height {
		offset = currentItemIndex + 1 - height
	}

	// Draw the list items.
	for index, item := range list.items {
		if index < offset {
			continue
		}

		if y >= bottomLimit {
			break
		}

		text := item.GetTitle()

		lineWidth := width

		style := tcell.StyleDefault.Foreground(list.mainTextColor)
		if item == list.selected {
			style = style.Foreground(list.selectedTextColor).Background(list.selectedBackgroundColor)
		}
		if item.HasNewMessages {
			style = style.Bold(true)
		}

		if item.UnreadMessages > 0 {
			unreadMessageCount := "99+"
			if item.UnreadMessages < 100 {
				unreadMessageCount = strconv.Itoa(item.UnreadMessages)
			}
			if item.Highlighted {
				unreadMessageCount += "!"
			}
			unreadMessageCount = fmt.Sprintf("(%s)", unreadMessageCount)
			widget.WriteLine(screen, tview.AlignRight, unreadMessageCount, x+lineWidth-6, y, 6, style)
			lineWidth -= len(unreadMessageCount) + 1
		}

		widget.WriteLine(screen, tview.AlignLeft, text, x, y, lineWidth, style)

		y++
		if y >= bottomLimit {
			break
		}
	}
}
