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
	"regexp"
	"strconv"
	"strings"
	"time"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type roomListItem struct {
	room     *rooms.Room
	priority float64
}

type RoomList struct {
	*tview.Box

	// The list of tags in display order.
	tags []string
	// The list of rooms, in reverse order.
	items map[string][]*rooms.Room
	// The selected room.
	selected    *rooms.Room
	selectedTag string

	scrollOffset int

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
		items: make(map[string][]*rooms.Room),
		tags:  []string{"m.favourite", "net.maunium.gomuks.fake.direct", "", "m.lowpriority"},

		scrollOffset: 0,

		mainTextColor:           tcell.ColorWhite,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
}

func (list *RoomList) Contains(roomID string) bool {
	for _, roomList := range list.items {
		for _, room := range roomList {
			if room.ID == roomID {
				return true
			}
		}
	}
	return false
}

func (list *RoomList) Add(room *rooms.Room) {
	for _, tag := range room.Tags() {
		list.AddToTag(tag.Tag, room)
	}
}

func (list *RoomList) CheckTag(tag string) {
	index := list.IndexTag(tag)

	items, ok := list.items[tag]

	if ok && len(items) == 0 {
		delete(list.items, tag)
		ok = false
	}

	if ok && index == -1 {
		list.tags = append(list.tags, tag)
	} /* TODO this doesn't work properly
	else if index != -1 {
		list.tags = append(list.tags[0:index], list.tags[index+1:]...)
	}*/
}

func (list *RoomList) AddToTag(tag string, room *rooms.Room) {
	if tag == "" && len(room.GetMembers()) == 2 {
		tag = "net.maunium.gomuks.fake.direct"
	}
	items, ok := list.items[tag]
	if !ok {
		list.items[tag] = []*rooms.Room{room}
		return
	}

	// Add space for new item.
	items = append(items, nil)
	// The default insert index is the newly added slot.
	// That index will be used if all other rooms in the list have the same LastReceivedMessage timestamp.
	insertAt := len(items) - 1
	// Find the spot where the new room should be put according to the last received message timestamps.
	for i := 0; i < len(items)-1; i++ {
		if items[i].LastReceivedMessage.After(room.LastReceivedMessage) {
			insertAt = i
			break
		}
	}
	// Move newer rooms forward in the array.
	for i := len(items) - 1; i > insertAt; i-- {
		items[i] = items[i-1]
	}
	// Insert room.
	items[insertAt] = room

	list.items[tag] = items
	list.CheckTag(tag)
}

func (list *RoomList) Remove(room *rooms.Room) {
	for _, tag := range room.Tags() {
		list.RemoveFromTag(tag.Tag, room)
	}
}

func (list *RoomList) RemoveFromTag(tag string, room *rooms.Room) {
	items, ok := list.items[tag]
	if !ok {
		return
	}

	index := list.indexInTag(tag, room)
	if index == -1 {
		return
	}

	items = append(items[0:index], items[index+1:]...)

	if len(items) == 0 {
		delete(list.items, tag)
	} else {
		list.items[tag] = items
	}

	if room == list.selected {
		// Room is currently selected, move selection to another room.
		if index > 0 {
			list.selected = items[index-1]
		} else if len(items) > 0 {
			list.selected = items[0]
		} else if len(list.items) > 0 {
			for _, tag := range list.tags {
				moreItems := list.items[tag]
				if len(moreItems) > 0 {
					list.selected = moreItems[0]
					list.selectedTag = tag
				}
			}
		} else {
			list.selected = nil
			list.selectedTag = ""
		}
	}
	list.CheckTag(tag)
}

func (list *RoomList) Bump(room *rooms.Room) {
	for _, tag := range room.Tags() {
		list.bumpInTag(tag.Tag, room)
	}
}

func (list *RoomList) bumpInTag(tag string, room *rooms.Room) {
	items, ok := list.items[tag]
	if !ok {
		return
	}

	found := false
	for i := 0; i < len(items)-1; i++ {
		if items[i] == room {
			found = true
		}
		if found {
			items[i] = items[i+1]
		}
	}
	if found {
		items[len(items)-1] = room
		room.LastReceivedMessage = time.Now()
	}
}

func (list *RoomList) Clear() {
	list.items = make(map[string][]*rooms.Room)
	list.selected = nil
	list.selectedTag = ""
}

func (list *RoomList) SetSelected(tag string, room *rooms.Room) {
	list.selected = room
	list.selectedTag = tag
	pos := list.index(tag, room)
	_, _, _, height := list.GetRect()
	if pos <= list.scrollOffset {
		list.scrollOffset = pos-1
	} else if pos >= list.scrollOffset+height {
		list.scrollOffset = pos-height+1
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
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
	_, _, _, viewHeight := list.GetRect()
	contentHeight := list.ContentHeight()
	if list.scrollOffset > contentHeight-viewHeight {
		list.scrollOffset = contentHeight - viewHeight
	}
}

func (list *RoomList) First() (string, *rooms.Room) {
	for _, tag := range list.tags {
		items := list.items[tag]
		if len(items) > 0 {
			return tag, items[len(items)-1]
		}
	}
	return "", nil
}

func (list *RoomList) Last() (string, *rooms.Room) {
	for tagIndex := len(list.tags) - 1; tagIndex >= 0; tagIndex-- {
		tag := list.tags[tagIndex]
		items := list.items[tag]
		if len(items) > 0 {
			return tag, items[0]
		}
	}
	return "", nil
}

func (list *RoomList) IndexTag(tag string) int {
	for index, entry := range list.tags {
		if tag == entry {
			return index
		}
	}
	return -1
}

func (list *RoomList) Previous() (string, *rooms.Room) {
	if len(list.items) == 0 {
		return "", nil
	} else if list.selected == nil {
		return list.First()
	}

	items := list.items[list.selectedTag]
	index := list.indexInTag(list.selectedTag, list.selected)
	if index == -1 {
		return list.First()
	} else if index == len(items)-1 {
		tagIndex := list.IndexTag(list.selectedTag)
		tagIndex--
		for ; tagIndex >= 0; tagIndex-- {
			prevTag := list.tags[tagIndex]
			prevTagItems := list.items[prevTag]
			if len(prevTagItems) > 0 {
				return prevTag, prevTagItems[0]
			}
		}
		return list.Last()
	}
	return list.selectedTag, items[index+1]
}

func (list *RoomList) Next() (string, *rooms.Room) {
	if len(list.items) == 0 {
		return "", nil
	} else if list.selected == nil {
		return list.First()
	}

	items := list.items[list.selectedTag]
	index := list.indexInTag(list.selectedTag, list.selected)
	if index == -1 {
		return list.Last()
	} else if index == 0 {
		tagIndex := list.IndexTag(list.selectedTag)
		tagIndex++
		for ; tagIndex < len(list.tags); tagIndex++ {
			nextTag := list.tags[tagIndex]
			nextTagItems := list.items[nextTag]
			if len(nextTagItems) > 0 {
				return nextTag, nextTagItems[len(nextTagItems)-1]
			}
		}
		return list.First()
	}
	return list.selectedTag, items[index-1]
}

func (list *RoomList) indexInTag(tag string, room *rooms.Room) int {
	roomIndex := -1
	items := list.items[tag]
	for index, entry := range items {
		if entry == room {
			roomIndex = index
			break
		}
	}
	return roomIndex
}

func (list *RoomList) index(tag string, room *rooms.Room) int {
	tagIndex := list.IndexTag(tag)
	if tagIndex == -1 {
		return -1
	}

	localIndex := list.indexInTag(tag, room)
	if localIndex == -1 {
		return -1
	}
	localIndex = len(list.items[tag]) - 1 - localIndex

	// Tag header
	localIndex += 1

	if tagIndex > 0 {
		for i := 0; i < tagIndex; i++ {
			previousTag := list.tags[i]
			previousItems := list.items[previousTag]

			tagDisplayName := list.GetTagDisplayName(previousTag)
			if len(tagDisplayName) > 0 {
				// Previous tag header + space
				localIndex += 2
				// Previous tag items
				localIndex += len(previousItems)
			}
		}
	}

	return localIndex
}

func (list *RoomList) ContentHeight() (height int) {
	for _, tag := range list.tags {
		items := list.items[tag]
		tagDisplayName := list.GetTagDisplayName(tag)
		if len(tagDisplayName) == 0 {
			continue
		}
		height += 2 + len(items)
	}
	return
}

func (list *RoomList) Get(n int) (string, *rooms.Room) {
	n += list.scrollOffset
	if n < 0 {
		return "", nil
	}
	for _, tag := range list.tags {
		// Tag header
		n--

		items := list.items[tag]
		if n < 0 {
			return "", nil
		} else if n < len(items) {
			return tag, items[len(items)-1-n]
		}

		// Tag items
		n -= len(items)
		// Tag footer
		n--
	}
	return "", nil
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
func (list *RoomList) Draw(screen tcell.Screen) {
	list.Box.Draw(screen)

	x, y, width, height := list.GetInnerRect()
	bottomLimit := y + height

	handledOffset := 0

	// Draw the list items.
	for _, tag := range list.tags {
		items := list.items[tag]
		tagDisplayName := list.GetTagDisplayName(tag)
		if len(tagDisplayName) == 0 {
			continue
		}

		localOffset := 0

		if handledOffset < list.scrollOffset {
			if handledOffset+len(items) < list.scrollOffset {
				handledOffset += len(items) + 2
				continue
			} else {
				localOffset = list.scrollOffset - handledOffset
				handledOffset += localOffset
			}
		}

		widget.WriteLine(screen, tview.AlignLeft, tagDisplayName, x, y, width, tcell.StyleDefault.Underline(true).Bold(true))
		y++
		for i := len(items) - 1; i >= 0; i-- {
			item := items[i]
			index := len(items) - 1 - i

			if y >= bottomLimit {
				break
			}

			if index < localOffset {
				continue
			}

			text := item.GetTitle()

			lineWidth := width

			style := tcell.StyleDefault.Foreground(list.mainTextColor)
			if tag == list.selectedTag && item == list.selected {
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
		y++
	}
}
