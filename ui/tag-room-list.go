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

package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
)

type OrderedRoom struct {
	*rooms.Room
	order json.Number
}

func NewOrderedRoom(order json.Number, room *rooms.Room) *OrderedRoom {
	return &OrderedRoom{
		Room:  room,
		order: order,
	}
}

func NewDefaultOrderedRoom(room *rooms.Room) *OrderedRoom {
	return NewOrderedRoom("0.5", room)
}

func (or *OrderedRoom) Draw(roomList *RoomList, screen mauview.Screen, x, y, lineWidth int, isSelected bool) {
	style := tcell.StyleDefault.
		Foreground(roomList.mainTextColor).
		Bold(or.HasNewMessages())
	if isSelected {
		style = style.
			Foreground(roomList.selectedTextColor).
			Background(roomList.selectedBackgroundColor)
	}

	unreadCount := or.UnreadCount()

	widget.WriteLinePadded(screen, mauview.AlignLeft, or.GetTitle(), x, y, lineWidth, style)

	if unreadCount > 0 {
		unreadMessageCount := "99+"
		if unreadCount < 100 {
			unreadMessageCount = strconv.Itoa(unreadCount)
		}
		if or.Highlighted() {
			unreadMessageCount += "!"
		}
		unreadMessageCount = fmt.Sprintf("(%s)", unreadMessageCount)
		widget.WriteLine(screen, mauview.AlignRight, unreadMessageCount, x+lineWidth-7, y, 7, style)
		lineWidth -= len(unreadMessageCount)
	}
}

type TagRoomList struct {
	mauview.NoopEventHandler
	rooms       []*OrderedRoom
	maxShown    int
	name        string
	displayname string
	parent      *RoomList
}

func NewTagRoomList(parent *RoomList, name string, rooms ...*OrderedRoom) *TagRoomList {
	return &TagRoomList{
		maxShown:    10,
		rooms:       rooms,
		name:        name,
		displayname: parent.GetTagDisplayName(name),
		parent:      parent,
	}
}

func (trl *TagRoomList) Visible() []*OrderedRoom {
	return trl.rooms[len(trl.rooms)-trl.Length():]
}

func (trl *TagRoomList) FirstVisible() *rooms.Room {
	visible := trl.Visible()
	if len(visible) > 0 {
		return visible[len(visible)-1].Room
	}
	return nil
}

func (trl *TagRoomList) LastVisible() *rooms.Room {
	visible := trl.Visible()
	if len(visible) > 0 {
		return visible[0].Room
	}
	return nil
}

func (trl *TagRoomList) All() []*OrderedRoom {
	return trl.rooms
}

func (trl *TagRoomList) Length() int {
	if len(trl.rooms) < trl.maxShown {
		return len(trl.rooms)
	}
	return trl.maxShown
}

func (trl *TagRoomList) TotalLength() int {
	return len(trl.rooms)
}

func (trl *TagRoomList) IsEmpty() bool {
	return len(trl.rooms) == 0
}

func (trl *TagRoomList) IsCollapsed() bool {
	return trl.maxShown == 0
}

func (trl *TagRoomList) ToggleCollapse() {
	if trl.IsCollapsed() {
		trl.maxShown = 10
	} else {
		trl.maxShown = 0
	}
}

func (trl *TagRoomList) HasInvisibleRooms() bool {
	return trl.maxShown < trl.TotalLength()
}

func (trl *TagRoomList) HasVisibleRooms() bool {
	return !trl.IsEmpty() && trl.maxShown > 0
}

// ShouldBeBefore returns if the first room should be after the second room in the room list.
// The manual order and last received message timestamp are considered.
func (trl *TagRoomList) ShouldBeAfter(room1 *OrderedRoom, room2 *OrderedRoom) bool {
	orderComp := strings.Compare(string(room1.order), string(room2.order))
	return orderComp == 1 || (orderComp == 0 && room2.LastReceivedMessage.After(room1.LastReceivedMessage))
}

func (trl *TagRoomList) Insert(order json.Number, mxRoom *rooms.Room) {
	room := NewOrderedRoom(order, mxRoom)
	// The default insert index is the newly added slot.
	// That index will be used if all other rooms in the list have the same LastReceivedMessage timestamp.
	insertAt := len(trl.rooms)
	// Find the spot where the new room should be put according to the last received message timestamps.
	for i := 0; i < len(trl.rooms)-1; i++ {
		if trl.rooms[i].Room == mxRoom {
			debug.Printf("Warning: tried to re-insert room %s into tag %s", mxRoom.ID, trl.name)
			return
		} else if trl.ShouldBeAfter(room, trl.rooms[i]) {
			insertAt = i
		}
	}
	trl.rooms = append(trl.rooms, nil)
	// Move newer rooms forward in the array.
	for i := len(trl.rooms) - 1; i > insertAt; i-- {
		trl.rooms[i] = trl.rooms[i-1]
	}
	// Insert room.
	trl.rooms[insertAt] = room
}

func (trl *TagRoomList) Bump(mxRoom *rooms.Room) {
	var found *OrderedRoom
	for i := 0; i < len(trl.rooms); i++ {
		currentRoom := trl.rooms[i]
		if found != nil {
			if trl.ShouldBeAfter(found, trl.rooms[i]) {
				// This room should be after the room being bumped, so insert the
				// room being bumped here and return
				trl.rooms[i-1] = found
				return
			}
			// Move older rooms back in the array
			trl.rooms[i-1] = currentRoom
		} else if currentRoom.Room == mxRoom {
			found = currentRoom
		}
	}
	// If the room being bumped should be first in the list, it won't be inserted during the loop.
	trl.rooms[len(trl.rooms)-1] = found
}

func (trl *TagRoomList) Remove(room *rooms.Room) {
	trl.RemoveIndex(trl.Index(room))
}

func (trl *TagRoomList) RemoveIndex(index int) {
	if index < 0 || index > len(trl.rooms) {
		return
	}
	last := len(trl.rooms) - 1
	if index < last {
		copy(trl.rooms[index:], trl.rooms[index+1:])
	}
	trl.rooms[last] = nil
	trl.rooms = trl.rooms[:last]
}

func (trl *TagRoomList) Index(room *rooms.Room) int {
	return trl.indexInList(trl.All(), room)
}

func (trl *TagRoomList) IndexVisible(room *rooms.Room) int {
	return trl.indexInList(trl.Visible(), room)
}

func (trl *TagRoomList) indexInList(list []*OrderedRoom, room *rooms.Room) int {
	for index, entry := range list {
		if entry.Room == room {
			return index
		}
	}
	return -1
}

var TagDisplayNameStyle = tcell.StyleDefault.Underline(true).Bold(true)
var TagRoomCountStyle = tcell.StyleDefault.Italic(true)

func (trl *TagRoomList) RenderHeight() int {
	if len(trl.displayname) == 0 {
		return 0
	}

	if trl.IsCollapsed() {
		return 1
	}
	height := 2 + trl.Length()
	if trl.HasInvisibleRooms() || trl.maxShown > 10 {
		height++
	}
	return height
}

func (trl *TagRoomList) DrawHeader(screen mauview.Screen) {
	width, _ := screen.Size()
	roomCount := strconv.Itoa(trl.TotalLength())

	// Draw tag name
	displayNameWidth := width - 1 - len(roomCount)
	widget.WriteLine(screen, mauview.AlignLeft, trl.displayname, 0, 0, displayNameWidth, TagDisplayNameStyle)

	// Draw tag room count
	roomCountX := len(trl.displayname) + 1
	roomCountWidth := width - 2 - len(trl.displayname)
	widget.WriteLine(screen, mauview.AlignLeft, roomCount, roomCountX, 0, roomCountWidth, TagRoomCountStyle)
}

func (trl *TagRoomList) Draw(screen mauview.Screen) {
	if len(trl.displayname) == 0 {
		return
	}

	trl.DrawHeader(screen)

	width, height := screen.Size()

	items := trl.Visible()

	if trl.IsCollapsed() {
		screen.SetCell(width-1, 0, tcell.StyleDefault, '▶')
		return
	}
	screen.SetCell(width-1, 0, tcell.StyleDefault, '▼')

	y := 1
	for i := len(items) - 1; i >= 0; i-- {
		if y >= height {
			return
		}

		item := items[i]

		lineWidth := width
		isSelected := trl.name == trl.parent.selectedTag && item.Room == trl.parent.selected
		item.Draw(trl.parent, screen, 0, y, lineWidth, isSelected)
		y++
	}
	hasLess := trl.maxShown > 10
	hasMore := trl.HasInvisibleRooms()
	if (hasLess || hasMore) && y < height {
		if hasMore {
			widget.WriteLine(screen, mauview.AlignRight, "More ↓", 0, y, width, tcell.StyleDefault)
		}
		if hasLess {
			widget.WriteLine(screen, mauview.AlignLeft, "↑ Less", 0, y, width, tcell.StyleDefault)
		}
		y++
	}
}
