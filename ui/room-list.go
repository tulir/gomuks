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

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type RoomList struct {
	*tview.Box

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
		Box:   tview.NewBox(),
		items: []*rooms.Room{},

		mainTextColor:           tcell.ColorWhite,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
}

func (list *RoomList) Contains(roomID string) bool {
	for _, room := range list.items {
		if room.ID == roomID {
			return true
		}
	}
	return false
}

func (list *RoomList) Add(room *rooms.Room) {
	list.items = append(list.items, room)
}

func (list *RoomList) Remove(room *rooms.Room) {
	index := list.Index(room)
	if index != -1 {
		list.items = append(list.items[0:index], list.items[index+1:]...)
		if room == list.selected {
			if index > 0 {
				list.selected = list.items[index-1]
			} else if len(list.items) > 0 {
				list.selected = list.items[0]
			} else {
				list.selected = nil
			}
		}
	}
}

func (list *RoomList) Bump(room *rooms.Room) {
	found := false
	for i := 0; i < len(list.items)-1; i++ {
		if list.items[i] == room {
			found = true
		}
		if found {
			list.items[i] = list.items[i+1]
		}
	}
	list.items[len(list.items)-1] = room
}

func (list *RoomList) Clear() {
	list.items = []*rooms.Room{}
	list.selected = nil
}

func (list *RoomList) SetSelected(room *rooms.Room) {
	list.selected = room
}

func (list *RoomList) HasSelected() bool {
	return list.selected != nil
}

func (list *RoomList) Selected() *rooms.Room {
	return list.selected
}

func (list *RoomList) Previous() *rooms.Room {
	if len(list.items) == 0 {
		return nil
	} else if list.selected == nil {
		return list.items[0]
	}

	index := list.Index(list.selected)
	if index == len(list.items)-1 {
		return list.items[0]
	}
	return list.items[index+1]
}

func (list *RoomList) Next() *rooms.Room {
	if len(list.items) == 0 {
		return nil
	} else if list.selected == nil {
		return list.items[0]
	}

	index := list.Index(list.selected)
	if index == 0 {
		return list.items[len(list.items)-1]
	}
	return list.items[index-1]
}

func (list *RoomList) Index(room *rooms.Room) int {
	roomIndex := -1
	for index, entry := range list.items {
		if entry == room {
			roomIndex = index
			break
		}
	}
	return roomIndex
}

func (list *RoomList) Get(n int) *rooms.Room {
	return list.items[len(list.items)-1-(n%len(list.items))]
}

// Draw draws this primitive onto the screen.
func (list *RoomList) Draw(screen tcell.Screen) {
	list.Box.Draw(screen)

	x, y, width, height := list.GetInnerRect()
	bottomLimit := y + height

	var offset int
	currentItemIndex := list.Index(list.selected)
	if currentItemIndex >= height {
		offset = currentItemIndex + 1 - height
	}

	// Draw the list items.
	for i := len(list.items) - 1; i >= 0; i-- {
		item := list.items[i]
		index := len(list.items) - 1 - i

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
