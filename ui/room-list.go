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
	"math"
	"regexp"
	"strings"
	"sync"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
)

type RoomList struct {
	sync.RWMutex

	parent *MainView

	// The list of tags in display order.
	tags []string
	// The list of rooms, in reverse order.
	items map[string]*TagRoomList
	// The selected room.
	selected    *rooms.Room
	selectedTag string

	scrollOffset int
	height       int
	width        int

	// The item main text color.
	mainTextColor tcell.Color
	// The text color for selected items.
	selectedTextColor tcell.Color
	// The background color for selected items.
	selectedBackgroundColor tcell.Color
}

func NewRoomList(parent *MainView) *RoomList {
	list := &RoomList{
		parent: parent,

		items: make(map[string]*TagRoomList),
		tags:  []string{"m.favourite", "net.maunium.gomuks.fake.direct", "", "m.lowpriority"},

		scrollOffset: 0,

		mainTextColor:           tcell.ColorWhite,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
	for _, tag := range list.tags {
		list.items[tag] = NewTagRoomList(list, tag)
	}
	return list
}

func (list *RoomList) Contains(roomID string) bool {
	list.RLock()
	defer list.RUnlock()
	for _, trl := range list.items {
		for _, room := range trl.All() {
			if room.ID == roomID {
				return true
			}
		}
	}
	return false
}

func (list *RoomList) Add(room *rooms.Room) {
	debug.Print("Adding room to list", room.ID, room.GetTitle(), room.IsDirect, room.Tags())
	for _, tag := range room.Tags() {
		list.AddToTag(tag, room)
	}
}

func (list *RoomList) checkTag(tag string) {
	index := list.indexTag(tag)

	trl, ok := list.items[tag]

	if ok && trl.IsEmpty() {
		//delete(list.items, tag)
		ok = false
	}

	if ok && index == -1 {
		list.tags = append(list.tags, tag)
	} /* TODO this doesn't work properly
	else if index != -1 {
		list.tags = append(list.tags[0:index], list.tags[index+1:]...)
	}*/
}

func (list *RoomList) AddToTag(tag rooms.RoomTag, room *rooms.Room) {
	list.Lock()
	defer list.Unlock()
	trl, ok := list.items[tag.Tag]
	if !ok {
		list.items[tag.Tag] = NewTagRoomList(list, tag.Tag, NewDefaultOrderedRoom(room))
		return
	}

	trl.Insert(tag.Order, room)
	list.checkTag(tag.Tag)
}

func (list *RoomList) Remove(room *rooms.Room) {
	for _, tag := range list.tags {
		list.RemoveFromTag(tag, room)
	}
}

func (list *RoomList) RemoveFromTag(tag string, room *rooms.Room) {
	list.Lock()
	defer list.Unlock()
	trl, ok := list.items[tag]
	if !ok {
		return
	}

	index := trl.Index(room)
	if index == -1 {
		return
	}

	trl.RemoveIndex(index)

	if trl.IsEmpty() {
		// delete(list.items, tag)
	}

	if room == list.selected {
		if index > 0 {
			list.selected = trl.All()[index-1].Room
		} else if trl.Length() > 0 {
			list.selected = trl.Visible()[0].Room
		} else if len(list.items) > 0 {
			for _, tag := range list.tags {
				moreItems := list.items[tag]
				if moreItems.Length() > 0 {
					list.selected = moreItems.Visible()[0].Room
					list.selectedTag = tag
				}
			}
		} else {
			list.selected = nil
			list.selectedTag = ""
		}
	}
	list.checkTag(tag)
}

func (list *RoomList) Bump(room *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	for _, tag := range room.Tags() {
		trl, ok := list.items[tag.Tag]
		if !ok {
			return
		}
		trl.Bump(room)
	}
}

func (list *RoomList) Clear() {
	list.Lock()
	defer list.Unlock()
	list.items = make(map[string]*TagRoomList)
	list.tags = []string{"m.favourite", "net.maunium.gomuks.fake.direct", "", "m.lowpriority"}
	for _, tag := range list.tags {
		list.items[tag] = NewTagRoomList(list, tag)
	}
	list.selected = nil
	list.selectedTag = ""
}

func (list *RoomList) SetSelected(tag string, room *rooms.Room) {
	list.selected = room
	list.selectedTag = tag
	pos := list.index(tag, room)
	if pos <= list.scrollOffset {
		list.scrollOffset = pos - 1
	} else if pos >= list.scrollOffset+list.height {
		list.scrollOffset = pos - list.height + 1
	}
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
	debug.Print("Selecting", room.GetTitle(), "in", list.GetTagDisplayName(tag))
}

func (list *RoomList) HasSelected() bool {
	return list.selected != nil
}

func (list *RoomList) Selected() (string, *rooms.Room) {
	return list.selectedTag, list.selected
}

func (list *RoomList) SelectedRoom() *rooms.Room {
	return list.selected
}

func (list *RoomList) AddScrollOffset(offset int) {
	list.scrollOffset += offset
	contentHeight := list.ContentHeight()
	if list.scrollOffset > contentHeight-list.height {
		list.scrollOffset = contentHeight - list.height
	}
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
}

func (list *RoomList) First() (string, *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	for _, tag := range list.tags {
		trl := list.items[tag]
		if trl.HasVisibleRooms() {
			return tag, trl.FirstVisible()
		}
	}
	return "", nil
}

func (list *RoomList) Last() (string, *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	for tagIndex := len(list.tags) - 1; tagIndex >= 0; tagIndex-- {
		tag := list.tags[tagIndex]
		trl := list.items[tag]
		if trl.HasVisibleRooms() {
			return tag, trl.LastVisible()
		}
	}
	return "", nil
}

func (list *RoomList) indexTag(tag string) int {
	for index, entry := range list.tags {
		if tag == entry {
			return index
		}
	}
	return -1
}

func (list *RoomList) Previous() (string, *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	if len(list.items) == 0 {
		return "", nil
	} else if list.selected == nil {
		return list.First()
	}

	trl := list.items[list.selectedTag]
	index := trl.IndexVisible(list.selected)
	indexInvisible := trl.Index(list.selected)
	if index == -1 && indexInvisible >= 0 {
		num := trl.TotalLength() - indexInvisible
		trl.maxShown = int(math.Ceil(float64(num)/10.0) * 10.0)
		index = trl.IndexVisible(list.selected)
	}

	if index == trl.Length()-1 {
		tagIndex := list.indexTag(list.selectedTag)
		tagIndex--
		for ; tagIndex >= 0; tagIndex-- {
			prevTag := list.tags[tagIndex]
			prevTRL := list.items[prevTag]
			if prevTRL.HasVisibleRooms() {
				return prevTag, prevTRL.LastVisible()
			}
		}
		return list.Last()
	} else if index >= 0 {
		return list.selectedTag, trl.Visible()[index+1].Room
	}
	return list.First()
}

func (list *RoomList) Next() (string, *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	if len(list.items) == 0 {
		return "", nil
	} else if list.selected == nil {
		return list.First()
	}

	trl := list.items[list.selectedTag]
	index := trl.IndexVisible(list.selected)
	indexInvisible := trl.Index(list.selected)
	if index == -1 && indexInvisible >= 0 {
		num := trl.TotalLength() - indexInvisible + 1
		trl.maxShown = int(math.Ceil(float64(num)/10.0) * 10.0)
		index = trl.IndexVisible(list.selected)
	}

	if index == 0 {
		tagIndex := list.indexTag(list.selectedTag)
		tagIndex++
		for ; tagIndex < len(list.tags); tagIndex++ {
			nextTag := list.tags[tagIndex]
			nextTRL := list.items[nextTag]
			if nextTRL.HasVisibleRooms() {
				return nextTag, nextTRL.FirstVisible()
			}
		}
		return list.First()
	} else if index > 0 {
		return list.selectedTag, trl.Visible()[index-1].Room
	}
	return list.Last()
}

// NextWithActivity Returns next room with activity.
//
// Sorted by (in priority):
//
// - Highlights
// - Messages
// - Other traffic (joins, parts, etc)
//
// TODO: Sorting. Now just finds first room with new messages.
func (list *RoomList) NextWithActivity() (string, *rooms.Room) {
	list.RLock()
	defer list.RUnlock()
	for tag, trl := range list.items {
		for _, room := range trl.All() {
			if room.HasNewMessages() {
				return tag, room.Room
			}
		}
	}
	// No room with activity found
	return "", nil
}

func (list *RoomList) index(tag string, room *rooms.Room) int {
	tagIndex := list.indexTag(tag)
	if tagIndex == -1 {
		return -1
	}

	trl, ok := list.items[tag]
	localIndex := -1
	if ok {
		localIndex = trl.IndexVisible(room)
	}
	if localIndex == -1 {
		return -1
	}
	localIndex = trl.Length() - 1 - localIndex

	// Tag header
	localIndex++

	if tagIndex > 0 {
		for i := 0; i < tagIndex; i++ {
			prevTag := list.tags[i]

			prevTRL := list.items[prevTag]
			localIndex += prevTRL.RenderHeight()
		}
	}

	return localIndex
}

func (list *RoomList) ContentHeight() (height int) {
	list.RLock()
	for _, tag := range list.tags {
		height += list.items[tag].RenderHeight()
	}
	list.RUnlock()
	return
}

func (list *RoomList) OnKeyEvent(event mauview.KeyEvent) bool {
	return false
}

func (list *RoomList) OnPasteEvent(event mauview.PasteEvent) bool {
	return false
}

func (list *RoomList) OnMouseEvent(event mauview.MouseEvent) bool {
	if event.HasMotion() {
		return false
	}
	switch event.Buttons() {
	case tcell.WheelUp:
		list.AddScrollOffset(-WheelScrollOffsetDiff)
	case tcell.WheelDown:
		list.AddScrollOffset(WheelScrollOffsetDiff)
	case tcell.Button1:
		x, y := event.Position()
		return list.clickRoom(y, x, event.Modifiers() == tcell.ModCtrl)
	}
	return false
}

func (list *RoomList) Focus() {

}

func (list *RoomList) Blur() {

}

func (list *RoomList) clickRoom(line, column int, mod bool) bool {
	line += list.scrollOffset
	if line < 0 {
		return false
	}
	list.RLock()
	defer list.RUnlock()
	for _, tag := range list.tags {
		trl := list.items[tag]
		if line--; line == -1 {
			trl.ToggleCollapse()
			return true
		}

		if trl.IsCollapsed() {
			continue
		}

		if line < 0 {
			break
		} else if line < trl.Length() {
			list.parent.SwitchRoom(tag, trl.Visible()[trl.Length()-1-line].Room)
			return true
		}

		// Tag items
		line -= trl.Length()

		hasMore := trl.HasInvisibleRooms()
		hasLess := trl.maxShown > 10
		if hasMore || hasLess {
			if line--; line == -1 {
				diff := 10
				if mod {
					diff = 100
				}
				if column <= 6 && hasLess {
					trl.maxShown -= diff
				} else if column >= list.width-6 && hasMore {
					trl.maxShown += diff
				}
				if trl.maxShown < 10 {
					trl.maxShown = 10
				}
				break
			}
		}
		// Tag footer
		line--
	}
	return false
}
var nsRegex = regexp.MustCompile("^[a-z]\\.[a-z](?:\\.[a-z])*$")

func (list *RoomList) GetTagDisplayName(tag string) string {
	switch {
	case len(tag) == 0:
		return "Rooms"
	case tag == "m.favourite":
		return "Favorites"
	case tag == "m.lowpriority":
		return "Low Priority"
	case tag == "net.maunium.gomuks.fake.direct":
		return "People"
	case strings.HasPrefix(tag, "u."):
		return tag[len("u."):]
	case !nsRegex.MatchString(tag):
		return tag
	default:
		return ""
	}
}

// Draw draws this primitive onto the screen.
func (list *RoomList) Draw(screen mauview.Screen) {
	list.width, list.height = screen.Size()
	y := 0
	yLimit := y + list.height
	y -= list.scrollOffset

	// Draw the list items.
	list.RLock()
	for _, tag := range list.tags {
		trl := list.items[tag]
		tagDisplayName := list.GetTagDisplayName(tag)
		if trl == nil || len(tagDisplayName) == 0 {
			continue
		}

		renderHeight := trl.RenderHeight()
		if y+renderHeight >= yLimit {
			renderHeight = yLimit - y
		}
		trl.Draw(mauview.NewProxyScreen(screen, 0, y, list.width, renderHeight))
		y += renderHeight
		if y >= yLimit {
			break
		}
	}
	list.RUnlock()
}
