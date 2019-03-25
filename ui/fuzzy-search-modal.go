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
	"fmt"
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
)

type FuzzySearchModal struct {
	mauview.Component

	search  *mauview.InputField
	results *mauview.TextView

	matches  fuzzy.Ranks
	selected int

	roomList   []*rooms.Room
	roomTitles []string

	parent *MainView
}

func NewFuzzySearchModal(mainView *MainView, width int, height int) *FuzzySearchModal {
	fs := &FuzzySearchModal{
		parent: mainView,
	}

	fs.InitList(mainView.rooms)

	fs.search = mauview.NewInputField().SetChangedFunc(fs.changeHandler)
	wrappedSearch := mauview.NewBox(fs.search).SetKeyCaptureFunc(fs.keyHandler)
	searchLabel := mauview.NewTextField().SetText("Room")
	combinedSearch := mauview.NewFlex().
		SetDirection(mauview.FlexColumn).
		AddFixedComponent(searchLabel, 5).
		AddProportionalComponent(wrappedSearch, 1)

	fs.results = mauview.NewTextView().SetRegions(true)

	// Flex widget containing input box and results
	container := mauview.NewBox(mauview.NewFlex().
		SetDirection(mauview.FlexRow).
		AddFixedComponent(combinedSearch, 1).
		AddProportionalComponent(fs.results, 1)).
		SetBorder(true).
		SetTitle("Quick Room Switcher").
		SetBlurCaptureFunc(func() bool {
			fs.parent.HideModal()
			return true
		})

	fs.Component = mauview.Center(container, width, height)

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
		fs.parent.parent.Render()
		fs.results.Highlight(strconv.Itoa(fs.matches[0].OriginalIndex))
		fs.results.ScrollToBeginning()
	} else {
		fs.results.Clear()
		fs.results.Highlight()
	}
}

func (fs *FuzzySearchModal) keyHandler(event mauview.KeyEvent) mauview.KeyEvent {
	highlights := fs.results.GetHighlights()
	switch event.Key() {
	case tcell.KeyEsc:
		// Close room finder
		fs.parent.HideModal()
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
			fs.parent.SwitchRoom(fs.roomList[fs.matches[fs.selected].OriginalIndex].Tags()[0].Tag, fs.roomList[fs.matches[fs.selected].OriginalIndex])
		}
		fs.parent.HideModal()
		fs.results.Clear()
		fs.search.SetText("")
		return nil
	}
	return event
}
