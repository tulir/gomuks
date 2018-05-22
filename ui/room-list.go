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

	"math"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type orderedRoom struct {
	*rooms.Room
	order string
}

func newOrderedRoom(order string, room *rooms.Room) *orderedRoom {
	return &orderedRoom{
		Room:  room,
		order: order,
	}
}

func convertRoom(room *rooms.Room) *orderedRoom {
	return newOrderedRoom("0.5", room)
}

type tagRoomList struct {
	rooms    []*orderedRoom
	maxShown int
}

func newTagRoomList(rooms ...*orderedRoom) *tagRoomList {
	return &tagRoomList{
		maxShown: 10,
		rooms:    rooms,
	}
}

func (trl *tagRoomList) Visible() []*orderedRoom {
	return trl.rooms[len(trl.rooms)-trl.Length():]
}

func (trl *tagRoomList) FirstVisible() *rooms.Room {
	visible := trl.Visible()
	if len(visible) > 0 {
		return visible[len(visible)-1].Room
	}
	return nil
}

func (trl *tagRoomList) LastVisible() *rooms.Room {
	visible := trl.Visible()
	if len(visible) > 0 {
		return visible[0].Room
	}
	return nil
}

func (trl *tagRoomList) All() []*orderedRoom {
	return trl.rooms
}

func (trl *tagRoomList) Length() int {
	if len(trl.rooms) < trl.maxShown {
		return len(trl.rooms)
	}
	return trl.maxShown
}

func (trl *tagRoomList) TotalLength() int {
	return len(trl.rooms)
}

func (trl *tagRoomList) IsEmpty() bool {
	return len(trl.rooms) == 0
}

func (trl *tagRoomList) IsCollapsed() bool {
	return trl.maxShown == 0
}

func (trl *tagRoomList) ToggleCollapse() {
	if trl.IsCollapsed() {
		trl.maxShown = 10
	} else {
		trl.maxShown = 0
	}
}

func (trl *tagRoomList) HasInvisibleRooms() bool {
	return trl.maxShown < trl.TotalLength()
}

func (trl *tagRoomList) HasVisibleRooms() bool {
	return !trl.IsEmpty() && trl.maxShown > 0
}

// ShouldBeBefore returns if the first room should be after the second room in the room list.
// The manual order and last received message timestamp are considered.
func (trl *tagRoomList) ShouldBeAfter(room1 *orderedRoom, room2 *orderedRoom) bool {
	orderComp := strings.Compare(room1.order, room2.order)
	return orderComp == 1 || (orderComp == 0 && room2.LastReceivedMessage.After(room1.LastReceivedMessage))
}

func (trl *tagRoomList) Insert(order string, mxRoom *rooms.Room) {
	room := newOrderedRoom(order, mxRoom)
	trl.rooms = append(trl.rooms, nil)
	// The default insert index is the newly added slot.
	// That index will be used if all other rooms in the list have the same LastReceivedMessage timestamp.
	insertAt := len(trl.rooms) - 1
	// Find the spot where the new room should be put according to the last received message timestamps.
	for i := 0; i < len(trl.rooms)-1; i++ {
		if trl.ShouldBeAfter(room, trl.rooms[i]) {
			insertAt = i
			break
		}
	}
	// Move newer rooms forward in the array.
	for i := len(trl.rooms) - 1; i > insertAt; i-- {
		trl.rooms[i] = trl.rooms[i-1]
	}
	// Insert room.
	trl.rooms[insertAt] = room
}

func (trl *tagRoomList) String() string {
	var str strings.Builder
	fmt.Fprintln(&str, "&tagRoomList{")
	fmt.Fprintf(&str, "    maxShown: %d,\n", trl.maxShown)
	fmt.Fprint(&str, "    rooms: {")
	for i, room := range trl.rooms {
		if room == nil {
			fmt.Fprintf(&str, "<<NIL>>")
		} else {
			fmt.Fprint(&str, room.ID)
		}
		if i != len(trl.rooms)-1 {
			fmt.Fprint(&str, ", ")
		}
	}
	fmt.Fprintln(&str, "},")
	fmt.Fprintln(&str, "}")
	return str.String()
}

func (trl *tagRoomList) Bump(mxRoom *rooms.Room) {
	var found *orderedRoom
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

func (trl *tagRoomList) Remove(room *rooms.Room) {
	trl.RemoveIndex(trl.Index(room))
}

func (trl *tagRoomList) RemoveIndex(index int) {
	if index < 0 || index > len(trl.rooms) {
		return
	}
	trl.rooms = append(trl.rooms[0:index], trl.rooms[index+1:]...)
}

func (trl *tagRoomList) Index(room *rooms.Room) int {
	return trl.indexInList(trl.All(), room)
}

func (trl *tagRoomList) IndexVisible(room *rooms.Room) int {
	return trl.indexInList(trl.Visible(), room)
}

func (trl *tagRoomList) indexInList(list []*orderedRoom, room *rooms.Room) int {
	for index, entry := range list {
		if entry.Room == room {
			return index
		}
	}
	return -1
}

type RoomList struct {
	*tview.Box

	// The list of tags in display order.
	tags []string
	// The list of rooms, in reverse order.
	items map[string]*tagRoomList
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
	list := &RoomList{
		Box:   tview.NewBox(),
		items: make(map[string]*tagRoomList),
		tags:  []string{"m.favourite", "net.maunium.gomuks.fake.direct", "", "m.lowpriority"},

		scrollOffset: 0,

		mainTextColor:           tcell.ColorWhite,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
	for _, tag := range list.tags {
		list.items[tag] = newTagRoomList()
	}
	return list
}

func (list *RoomList) Contains(roomID string) bool {
	for _, tagRoomList := range list.items {
		for _, room := range tagRoomList.All() {
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

func (list *RoomList) CheckTag(tag string) {
	index := list.IndexTag(tag)

	tagRoomList, ok := list.items[tag]

	if ok && tagRoomList.IsEmpty() {
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
	tagRoomList, ok := list.items[tag.Tag]
	if !ok {
		list.items[tag.Tag] = newTagRoomList(convertRoom(room))
		return
	}

	tagRoomList.Insert(tag.Order, room)
	list.CheckTag(tag.Tag)
}

func (list *RoomList) Remove(room *rooms.Room) {
	for _, tag := range list.tags {
		list.RemoveFromTag(tag, room)
	}
}

func (list *RoomList) RemoveFromTag(tag string, room *rooms.Room) {
	tagRoomList, ok := list.items[tag]
	if !ok {
		return
	}

	index := tagRoomList.Index(room)
	if index == -1 {
		return
	}

	tagRoomList.RemoveIndex(index)

	if tagRoomList.IsEmpty() {
		// delete(list.items, tag)
	}

	if room == list.selected {
		if index > 0 {
			list.selected = tagRoomList.All()[index-1].Room
		} else if tagRoomList.Length() > 0 {
			list.selected = tagRoomList.Visible()[0].Room
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
	list.CheckTag(tag)
}

func (list *RoomList) Bump(room *rooms.Room) {
	for _, tag := range room.Tags() {
		tagRoomList, ok := list.items[tag.Tag]
		if !ok {
			return
		}
		tagRoomList.Bump(room)
	}
}

func (list *RoomList) Clear() {
	list.items = make(map[string]*tagRoomList)
	list.tags = []string{"m.favourite", "net.maunium.gomuks.fake.direct", "", "m.lowpriority"}
	for _, tag := range list.tags {
		list.items[tag] = newTagRoomList()
	}
	list.selected = nil
	list.selectedTag = ""
}

func (list *RoomList) SetSelected(tag string, room *rooms.Room) {
	list.selected = room
	list.selectedTag = tag
	pos := list.index(tag, room)
	_, _, _, height := list.GetRect()
	if pos <= list.scrollOffset {
		list.scrollOffset = pos - 1
	} else if pos >= list.scrollOffset+height {
		list.scrollOffset = pos - height + 1
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
	_, _, _, viewHeight := list.GetRect()
	contentHeight := list.ContentHeight()
	if list.scrollOffset > contentHeight-viewHeight {
		list.scrollOffset = contentHeight - viewHeight
	}
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
}

func (list *RoomList) First() (string, *rooms.Room) {
	for _, tag := range list.tags {
		tagRoomList := list.items[tag]
		if tagRoomList.HasVisibleRooms() {
			return tag, tagRoomList.FirstVisible()
		}
	}
	return "", nil
}

func (list *RoomList) Last() (string, *rooms.Room) {
	for tagIndex := len(list.tags) - 1; tagIndex >= 0; tagIndex-- {
		tag := list.tags[tagIndex]
		tagRoomList := list.items[tag]
		if tagRoomList.HasVisibleRooms() {
			return tag, tagRoomList.LastVisible()
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

	tagRoomList := list.items[list.selectedTag]
	index := tagRoomList.IndexVisible(list.selected)
	indexInvisible := tagRoomList.Index(list.selected)
	if index == -1 && indexInvisible >= 0 {
		num := tagRoomList.TotalLength() - indexInvisible
		tagRoomList.maxShown = int(math.Ceil(float64(num)/10.0) * 10.0)
		index = tagRoomList.IndexVisible(list.selected)
	}

	if index == tagRoomList.Length()-1 {
		tagIndex := list.IndexTag(list.selectedTag)
		tagIndex--
		for ; tagIndex >= 0; tagIndex-- {
			prevTag := list.tags[tagIndex]
			prevTagRoomList := list.items[prevTag]
			if prevTagRoomList.HasVisibleRooms() {
				return prevTag, prevTagRoomList.LastVisible()
			}
		}
		return list.Last()
	} else if index >= 0 {
		return list.selectedTag, tagRoomList.Visible()[index+1].Room
	}
	return list.First()
}

func (list *RoomList) Next() (string, *rooms.Room) {
	if len(list.items) == 0 {
		return "", nil
	} else if list.selected == nil {
		return list.First()
	}

	tagRoomList := list.items[list.selectedTag]
	index := tagRoomList.IndexVisible(list.selected)
	indexInvisible := tagRoomList.Index(list.selected)
	if index == -1 && indexInvisible >= 0 {
		num := tagRoomList.TotalLength() - indexInvisible + 1
		tagRoomList.maxShown = int(math.Ceil(float64(num)/10.0) * 10.0)
		index = tagRoomList.IndexVisible(list.selected)
	}

	if index == 0 {
		tagIndex := list.IndexTag(list.selectedTag)
		tagIndex++
		for ; tagIndex < len(list.tags); tagIndex++ {
			nextTag := list.tags[tagIndex]
			nextTagRoomList := list.items[nextTag]
			if nextTagRoomList.HasVisibleRooms() {
				return nextTag, nextTagRoomList.FirstVisible()
			}
		}
		return list.First()
	} else if index > 0 {
		return list.selectedTag, tagRoomList.Visible()[index-1].Room
	}
	return list.Last()
}

func (list *RoomList) index(tag string, room *rooms.Room) int {
	tagIndex := list.IndexTag(tag)
	if tagIndex == -1 {
		return -1
	}

	tagRoomList, ok := list.items[tag]
	localIndex := -1
	if ok {
		localIndex = tagRoomList.IndexVisible(room)
	}
	if localIndex == -1 {
		return -1
	}
	localIndex = tagRoomList.Length() - 1 - localIndex

	// Tag header
	localIndex += 1

	if tagIndex > 0 {
		for i := 0; i < tagIndex; i++ {
			previousTag := list.tags[i]
			previousTagRoomList := list.items[previousTag]

			tagDisplayName := list.GetTagDisplayName(previousTag)
			if len(tagDisplayName) > 0 {
				if previousTagRoomList.IsCollapsed() {
					localIndex++
					continue
				}
				// Previous tag header + space
				localIndex += 2
				if previousTagRoomList.HasInvisibleRooms() {
					// Previous tag "Show more" button
					localIndex++
				}
				// Previous tag items
				localIndex += previousTagRoomList.Length()
			}
		}
	}

	return localIndex
}

func (list *RoomList) ContentHeight() (height int) {
	for _, tag := range list.tags {
		tagRoomList := list.items[tag]
		tagDisplayName := list.GetTagDisplayName(tag)
		if len(tagDisplayName) == 0 {
			continue
		}
		if tagRoomList.IsCollapsed() {
			height++
			continue
		}
		height += 2 + tagRoomList.Length()
		if tagRoomList.HasInvisibleRooms() {
			height++
		}
	}
	return
}

func (list *RoomList) HandleClick(column, line int, mod bool) (string, *rooms.Room) {
	line += list.scrollOffset
	if line < 0 {
		return "", nil
	}
	for _, tag := range list.tags {
		tagRoomList := list.items[tag]
		if line--; line == -1 {
			tagRoomList.ToggleCollapse()
			return "", nil
		}

		if tagRoomList.IsCollapsed() {
			continue
		}

		if line < 0 {
			return "", nil
		} else if line < tagRoomList.Length() {
			return tag, tagRoomList.Visible()[tagRoomList.Length()-1-line].Room
		}

		// Tag items
		line -= tagRoomList.Length()

		hasMore := tagRoomList.HasInvisibleRooms()
		hasLess := tagRoomList.maxShown > 10
		if hasMore || hasLess {
			if line--; line == -1 {
				diff := 10
				if mod {
					diff = 100
				}
				_, _, width, _ := list.GetRect()
				if column <= 6 && hasLess {
					tagRoomList.maxShown -= diff
				} else if column >= width-6 && hasMore {
					tagRoomList.maxShown += diff
				}
				if tagRoomList.maxShown < 10 {
					tagRoomList.maxShown = 10
				}
				return "", nil
			}
		}
		// Tag footer
		line--
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
		tagRoomList := list.items[tag]
		tagDisplayName := list.GetTagDisplayName(tag)
		if len(tagDisplayName) == 0 {
			continue
		}

		localOffset := 0

		if handledOffset < list.scrollOffset {
			if handledOffset+tagRoomList.Length() < list.scrollOffset {
				if tagRoomList.IsCollapsed() {
					handledOffset++
				} else {
					handledOffset += tagRoomList.Length() + 2
					if tagRoomList.HasInvisibleRooms() || tagRoomList.maxShown > 10 {
						handledOffset++
					}
				}
				continue
			} else {
				localOffset = list.scrollOffset - handledOffset
				handledOffset += localOffset
			}
		}

		roomCount := strconv.Itoa(tagRoomList.TotalLength())
		widget.WriteLine(screen, tview.AlignLeft, tagDisplayName, x, y, width-1-len(roomCount), tcell.StyleDefault.Underline(true).Bold(true))
		widget.WriteLine(screen, tview.AlignLeft, roomCount, x+len(tagDisplayName)+1, y, width-2-len(tagDisplayName), tcell.StyleDefault.Italic(true))

		items := tagRoomList.Visible()

		if tagRoomList.IsCollapsed() {
			screen.SetCell(x+width-1, y, tcell.StyleDefault, '▶')
			y++
			continue
		}
		screen.SetCell(x+width-1, y, tcell.StyleDefault, '▼')
		y++

		for i := tagRoomList.Length() - 1; i >= 0; i-- {
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
			if tag == list.selectedTag && item.Room == list.selected {
				style = style.Foreground(list.selectedTextColor).Background(list.selectedBackgroundColor)
			}
			if item.HasNewMessages() {
				style = style.Bold(true)
			}

			unreadCount := item.UnreadCount()
			if unreadCount > 0 {
				unreadMessageCount := "99+"
				if unreadCount < 100 {
					unreadMessageCount = strconv.Itoa(unreadCount)
				}
				if item.Highlighted() {
					unreadMessageCount += "!"
				}
				unreadMessageCount = fmt.Sprintf("(%s)", unreadMessageCount)
				widget.WriteLine(screen, tview.AlignRight, unreadMessageCount, x+lineWidth-7, y, 7, style)
				lineWidth -= len(unreadMessageCount)
			}

			widget.WriteLinePadded(screen, tview.AlignLeft, text, x, y, lineWidth, style)
			y++

			if y >= bottomLimit {
				break
			}
		}
		hasLess := tagRoomList.maxShown > 10
		hasMore := tagRoomList.HasInvisibleRooms()
		if hasLess || hasMore {
			if hasMore {
				widget.WriteLine(screen, tview.AlignRight, "More ↓", x, y, width, tcell.StyleDefault)
			}
			if hasLess {
				widget.WriteLine(screen, tview.AlignLeft, "↑ Less", x, y, width, tcell.StyleDefault)
			}
			y++
		}

		y++
	}
}
