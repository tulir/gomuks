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
	"sort"
	"strconv"

	"github.com/renstrom/fuzzysearch/fuzzy"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type FuzzySearchModal struct {
	tview.Primitive

	search  *tview.InputField
	results *tview.TextView

	matches  fuzzy.Ranks
	selected int

	roomList   []*rooms.Room
	roomTitles []string

	parent   *GomuksUI
	mainView *MainView
}

func NewFuzzySearchModal(mainView *MainView, width int, height int) *FuzzySearchModal {
	fs := &FuzzySearchModal{
		parent:   mainView.parent,
		mainView: mainView,
	}

	fs.InitList(mainView.rooms)

	fs.search = tview.NewInputField().
		SetLabel("Room: ")
	fs.search.
		SetChangedFunc(fs.changeHandler).
		SetInputCapture(fs.keyHandler)

	fs.results = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true)
	fs.results.SetBorderPadding(1, 0, 0, 0)

	// Flex widget containing input box and results
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fs.search, 1, 0, true).
		AddItem(fs.results, 0, 1, false)
	container.
		SetBorder(true).
		SetBorderPadding(1, 1, 1, 1).
		SetTitle("Quick Room Switcher")

	fs.Primitive = widget.TransparentCenter(width, height, container)

	return fs
}

func (fs *FuzzySearchModal) InitList(rooms map[string]*RoomView) {
	for _, room := range rooms {
		fs.roomList = append(fs.roomList, room.Room)
		fs.roomTitles = append(fs.roomTitles, room.Room.GetTitle())
	}
}

func (fs *FuzzySearchModal) changeHandler(str string) {
	// Get matches and display in result box
	fs.matches = fuzzy.RankFindFold(str, fs.roomTitles)
	if len(str) > 0 && len(fs.matches) > 0 {
		sort.Sort(fs.matches)
		fs.results.Clear()
		for _, match := range fs.matches {
			fmt.Fprintf(fs.results, `["%d"]%s[""]%s`, match.OriginalIndex, match.Target, "\n")
		}
		fs.parent.Render()
		fs.results.Highlight(strconv.Itoa(fs.matches[0].OriginalIndex))
		fs.results.ScrollToBeginning()
	} else {
		fs.results.Clear()
		fs.results.Highlight()
	}
}

func (fs *FuzzySearchModal) keyHandler(event *tcell.EventKey) *tcell.EventKey {
	highlights := fs.results.GetHighlights()
	switch event.Key() {
	case tcell.KeyEsc:
		// Close room finder
		fs.parent.views.RemovePage("fuzzy-search-modal")
		fs.parent.app.SetFocus(fs.parent.views)
		return nil
	case tcell.KeyTab:
		// Cycle highlighted area to next match
		if len(highlights) > 0 {
			fs.selected = (fs.selected + 1) % len(fs.matches)
			fs.results.Highlight(strconv.Itoa(fs.matches[fs.selected].OriginalIndex))
			fs.results.ScrollToHighlight()
		}
		return nil
	case tcell.KeyEnter:
		// Switch room to currently selected room
		if len(highlights) > 0 {
			debug.Print("Fuzzy Selected Room:", fs.roomList[fs.matches[fs.selected].OriginalIndex].GetTitle())
			fs.mainView.SwitchRoom(fs.roomList[fs.matches[fs.selected].OriginalIndex].Tags()[0].Tag, fs.roomList[fs.matches[fs.selected].OriginalIndex])
		}
		fs.parent.views.RemovePage("fuzzy-search-modal")
		fs.parent.app.SetFocus(fs.parent.views)
		fs.results.Clear()
		fs.search.SetText("")
		return nil
	}
	return event
}
