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

	sync "github.com/sasha-s/go-deadlock"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix/event"
)

const beeperBridgeSuffix = ":beeper.local"

type split struct {
	name, tag string
	collapsed bool
	rooms     []*rooms.Room
}

func (splt *split) title(selected bool) string {
	char := "▼"
	if splt.collapsed {
		if selected {
			char = "▷"
		} else {
			char = "▶"
		}
	}
	return splt.name + " " + char
}

type RosterView struct {
	mauview.Component
	sync.RWMutex

	split *split
	room  *rooms.Room

	splits      []*split
	splitLookup map[string]*split

	height, width,
	splitOffset, roomOffset int
	focused bool

	parent *MainView
}

func NewRosterView(mainView *MainView) *RosterView {
	splts := make([]*split, 0)
	splts = append(splts, &split{
		name:  "Favorites",
		tag:   "m.favourite",
		rooms: make([]*rooms.Room, 0),
	})
	splts = append(splts, &split{
		name:  "Inbox",
		tag:   "",
		rooms: make([]*rooms.Room, 0),
	})
	splts = append(splts, &split{
		name:      "Low Priority",
		tag:       "m.lowpriority",
		collapsed: true,
		rooms:     make([]*rooms.Room, 0),
	})

	rstr := &RosterView{
		parent:      mainView,
		splits:      splts,
		splitLookup: make(map[string]*split, 0),
	}

	for _, splt := range rstr.splits {
		rstr.splitLookup[splt.tag] = splt
	}

	return rstr
}

// splitForRoom returns the corresponding split for a given room.
func (rstr *RosterView) splitForRoom(room *rooms.Room, create bool) *split {
	if room == nil {
		return nil
	}

	if strings.HasSuffix(room.ID.String(), beeperBridgeSuffix) {
		splt, sortByTag := rstr.splitForDiscordAndSlackRooms(room, create)
		if !sortByTag {
			return splt
		}
	}

	for _, tag := range room.Tags() {
		if splt, ok := rstr.splitLookup[tag.Tag]; ok {
			return splt
		}
	}

	return nil
}

// splitForDiscordAndSlackRooms returns the corresponding split for
// passed bridged rooms from the Discord and Slack networks. If the room
// is not bridged, or is not from Discord or Slack, it returns (nil, true).
// If the split does not yet exist, it is created.
func (rstr *RosterView) splitForDiscordAndSlackRooms(room *rooms.Room, create bool) (*split, bool) {
	bridgeEvent := room.MostRecentStateEventOfType(event.StateBridge)
	if bridgeEvent == nil {
		return nil, true
	}

	if _, server, err := bridgeEvent.Sender.Parse(); err != nil || server != beeperBridgeSuffix[1:] {
		return nil, true
	}

	content := bridgeEvent.Content
	bridge := content.AsBridge()
	if bridge.Protocol.DisplayName != "Discord" && bridge.Protocol.DisplayName != "Slack" {
		return nil, true
	}

	if bridge.Network == nil {
		// Need to check account data for "show in inbox" settings, which
		// govern the display of DMs.
		if _, ok := content.Raw["com.beeper.room_type"]; ok && bridge.Protocol.DisplayName == "Discord" {
			bridge.Network = &event.BridgeInfoSection{
				ID:          "discord-dms",
				DisplayName: "Discord DMs",
			}
		} else {
			return nil, true
		}
	}

	if splt, ok := rstr.splitLookup[bridge.Network.ID]; ok {
		return splt, false
	}

	if create {
		splt := &split{
			name:      bridge.Network.DisplayName,
			tag:       bridge.Network.ID,
			collapsed: true,
			rooms:     make([]*rooms.Room, 0),
		}
		rstr.splits = append(rstr.splits, splt)
		rstr.splitLookup[splt.tag] = splt
		return splt, false
	}

	return nil, true
}

func (rstr *RosterView) Add(room *rooms.Room) {
	if room.IsReplaced() {
		return
	}

	rstr.Lock()
	defer rstr.Unlock()

	splt := rstr.splitForRoom(room, true)
	if splt == nil {
		return
	}

	insertAt := len(splt.rooms)
	for i := 0; i < len(splt.rooms); i++ {
		if splt.rooms[i] == room {
			return
		} else if room.LastReceivedMessage.After(splt.rooms[i].LastReceivedMessage) {
			insertAt = i
			break
		}
	}
	splt.rooms = append(splt.rooms, nil)
	copy(splt.rooms[insertAt+1:], splt.rooms[insertAt:len(splt.rooms)-1])
	splt.rooms[insertAt] = room
}

func (rstr *RosterView) Remove(room *rooms.Room) {
	rstr.Lock()
	defer rstr.Unlock()

	splt, index := rstr.indexOfRoom(room)
	if index < 0 || index > len(splt.rooms) {
		return
	}

	last := len(splt.rooms) - 1
	if index < last {
		copy(splt.rooms[index:], splt.rooms[index+1:])
	}
	splt.rooms[last] = nil
	splt.rooms = splt.rooms[:last]
}

func (rstr *RosterView) Bump(room *rooms.Room) {
	rstr.Remove(room)
	rstr.Add(room)
}

func (rstr *RosterView) indexOfRoom(room *rooms.Room) (*split, int) {
	if room == nil {
		return nil, -1
	}

	splt := rstr.splitForRoom(room, false)
	if splt == nil {
		return nil, -1
	}

	for index, entry := range splt.rooms {
		if entry == room {
			return splt, index
		}
	}

	return nil, -1
}

func (rstr *RosterView) indexOfSplit(split *split) int {
	for index, entry := range rstr.splits {
		if entry == split {
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

func (rstr *RosterView) first() (*split, *rooms.Room) {
	for _, splt := range rstr.splits {
		if !splt.collapsed && len(splt.rooms) > 0 {
			return splt, splt.rooms[0]
		}
	}
	return rstr.splits[0], rstr.splits[0].rooms[0]
}

func (rstr *RosterView) Last() (*split, *rooms.Room) {
	rstr.Lock()
	defer rstr.Unlock()

	for index := len(rstr.splits) - 1; index >= 0; index-- {
		if rstr.splits[index].collapsed || len(rstr.splits[index].rooms) == 0 {
			continue
		}
		splt := rstr.splits[index]
		return splt, splt.rooms[len(splt.rooms)-1]
	}

	return rstr.splits[len(rstr.splits)-1], rstr.splits[len(rstr.splits)-1].rooms[0]
}

func (rstr *RosterView) MatchOffsetsToSelection() {
	rstr.Lock()
	defer rstr.Unlock()

	var splt *split
	splt, rstr.roomOffset = rstr.indexOfRoom(rstr.room)
	rstr.splitOffset = rstr.indexOfSplit(splt)
}

func (rstr *RosterView) ScrollNext() {
	rstr.Lock()

	if splt, index := rstr.indexOfRoom(rstr.room); splt == nil || index == -1 {
		rstr.split, rstr.room = rstr.first()
	} else if index < len(splt.rooms)-1 && !splt.collapsed {
		rstr.room = splt.rooms[index+1]
	} else {
		idx := -1
		for i, s := range rstr.splits {
			if s == rstr.split {
				idx = i
			}
		}
		for i := idx + 1; i < len(rstr.splits); i++ {
			if len(rstr.splits[i].rooms) > 0 {
				rstr.split = rstr.splits[i]
				rstr.room = rstr.splits[i].rooms[0]
				break
			}
		}
	}

	rstr.Unlock()

	if rstr.HeightThroughSelection() > rstr.height {
		rstr.MatchOffsetsToSelection()
	}
}

func (rstr *RosterView) ScrollPrev() {
	rstr.Lock()
	defer rstr.Unlock()

	if splt, index := rstr.indexOfRoom(rstr.room); splt == nil || index == -1 {
		return
	} else if index > 0 && !splt.collapsed {
		rstr.room = splt.rooms[index-1]
		if index == rstr.roomOffset {
			rstr.roomOffset--
		}
	} else {
		for idx := len(rstr.splits) - 1; idx > 0; idx-- {
			if rstr.splits[idx] == rstr.split {
				rstr.split = rstr.splits[idx-1]
				rstr.splitOffset = idx - 1

				if len(rstr.split.rooms) > 0 {
					if rstr.split.collapsed {
						rstr.room = rstr.split.rooms[0]
						rstr.roomOffset = 0
					} else {
						rstr.room = rstr.split.rooms[len(rstr.split.rooms)-1]
						rstr.roomOffset = len(rstr.split.rooms) - 1
					}
				}
				return
			}
		}
	}
}

func (rstr *RosterView) HeightThroughSelection() int {
	height := 3
	for _, splt := range rstr.splits[rstr.splitOffset:] {
		if len(splt.rooms) == 0 {
			continue
		}

		height++
		if splt.collapsed {
			continue
		}

		for _, r := range splt.rooms[rstr.roomOffset:] {
			height += 2
			if r == rstr.room {
				return height
			}
		}
	}
	return -1
}

func (rstr *RosterView) Draw(screen mauview.Screen) {
	if rstr.focused {
		if roomView, ok := rstr.parent.getRoomView(rstr.room.ID, true); ok {
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
	for _, splt := range rstr.splits[rstr.splitOffset:] {

		if len(splt.rooms) == 0 {
			continue
		}

		name := splt.title(splt == rstr.split)
		halfWidth := (rstr.width - 5 - len(name)) / 2
		widget.WriteLineColor(screen, mauview.AlignCenter, name, halfWidth, y, halfWidth, tcell.ColorGray)
		y++

		if splt.collapsed {
			continue
		}

		iter := splt.rooms
		if splt == rstr.split {
			iter = iter[rstr.roomOffset:]
		}

		for _, room := range iter {
			if room.IsReplaced() {
				continue
			}

			renderHeight := 2
			if y+renderHeight >= rstr.height {
				renderHeight = rstr.height - y
			}

			isSelected := room == rstr.room

			style := tcell.StyleDefault.
				Foreground(tcell.ColorDefault).
				Bold(room.HasNewMessages())
			if isSelected {
				style = style.
					Foreground(tcell.ColorBlack).
					Background(tcell.ColorWhite).
					Italic(true)
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
			startingX := 2

			if isSelected {
				lastMessage = "  " + lastMessage
				msgStyle = msgStyle.Background(tcell.ColorWhite).Italic(true)
				startingX += 2

				widget.WriteLine(screen, mauview.AlignLeft, string(tcell.RuneDiamond)+" ", 2, y, 4, style)
			}

			tmX := rstr.width - 3 - len(tm)
			widget.WriteLinePadded(screen, mauview.AlignLeft, room.GetTitle(), startingX, y, tmX, style)
			widget.WriteLine(screen, mauview.AlignLeft, tm, tmX, y, startingX+len(tm), style)
			widget.WriteLinePadded(screen, mauview.AlignLeft, lastMessage, 2, y+1, rstr.width-5, msgStyle)

			y += renderHeight
			if y >= rstr.height {
				break
			}
		}
	}
}

func (rstr *RosterView) OnKeyEvent(event mauview.KeyEvent) bool {
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}

	if rstr.focused {
		if rstr.parent.config.Keybindings.Roster[kb] == "clear" {
			rstr.focused = false
			rstr.split = nil
			rstr.room = nil
		} else {
			if roomView, ok := rstr.parent.getRoomView(rstr.room.ID, true); ok {
				return roomView.OnKeyEvent(event)
			}
		}
	}

	switch rstr.parent.config.Keybindings.Roster[kb] {
	case "next_room":
		rstr.ScrollNext()
	case "prev_room":
		rstr.ScrollPrev()
		rstr.MatchOffsetsToSelection()
	case "top":
		rstr.Lock()
		rstr.split, rstr.room = rstr.first()
		rstr.Unlock()
		rstr.MatchOffsetsToSelection()
	case "bottom":
		rstr.split, rstr.room = rstr.Last()
		if rstr.HeightThroughSelection() > rstr.height {
			rstr.MatchOffsetsToSelection()
		}
	case "clear":
		rstr.split = nil
		rstr.room = nil
	case "quit":
		rstr.parent.gmx.Stop(true)
	case "enter":
		if rstr.split != nil && !rstr.split.collapsed {
			rstr.focused = rstr.room != nil
		}
	case "toggle_split":
		if rstr.split != nil {
			rstr.split.collapsed = !rstr.split.collapsed
		}
	default:
		return false
	}
	return true
}

func (rstr *RosterView) OnMouseEvent(event mauview.MouseEvent) bool {
	if rstr.focused {
		if roomView, ok := rstr.parent.getRoomView(rstr.room.ID, true); ok {
			return roomView.OnMouseEvent(event)
		}
	}

	if event.HasMotion() {
		return false
	}

	switch event.Buttons() {
	case tcell.WheelUp:
		rstr.ScrollPrev()
		return true
	case tcell.WheelDown:
		rstr.ScrollNext()
		return true
	}

	return false
}
