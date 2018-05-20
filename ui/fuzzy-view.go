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

	"github.com/evidlo/fuzzysearch/fuzzy"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type FuzzyView struct {
	*tview.Grid
	matches  fuzzy.Ranks
	selected int
}

func NewFuzzyView(view *MainView, width int, height int) *FuzzyView {

	rooms := []*rooms.Room{}
	roomtitles := []string{}
	for _, tag := range view.roomList.tags {
		for _, room := range view.roomList.items[tag].rooms {
			rooms = append(rooms, room.Room)
			roomtitles = append(roomtitles, room.GetTitle())
		}
	}
	// search box for fuzzy search
	fuzzySearch := tview.NewInputField().
		SetLabel("Room: ")

	// list of rooms matching fuzzy search
	fuzzyResults := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true)

	fuzzyResults.
		SetBorderPadding(1, 0, 0, 0)

	// flexbox containing input box and results
	fuzzyFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fuzzySearch, 1, 0, true).
		AddItem(fuzzyResults, 0, 1, false)

	fuzzyFlex.SetBorder(true).
		SetBorderPadding(1, 1, 1, 1).
		SetTitle("Fuzzy Room Finder")

	var matches fuzzy.Ranks
	var selected int
	fuzz := &FuzzyView{
		Grid: tview.NewGrid().
			SetColumns(0, width, 0).
			SetRows(0, height, 0).
			AddItem(fuzzyFlex, 1, 1, 1, 1, 0, 0, true),
		matches:  matches,
		selected: selected,
	}

	// callback to update search box
	fuzzySearch.SetChangedFunc(func(str string) {
		// get matches and display in fuzzyResults
		fuzz.matches = fuzzy.RankFindFold(str, roomtitles)
		if len(str) > 0 && len(fuzz.matches) > 0 {
			sort.Sort(fuzz.matches)
			fuzzyResults.Clear()
			for _, match := range fuzz.matches {
				fmt.Fprintf(fuzzyResults, "[\"%d\"]%s[\"\"]\n", match.Index, match.Target)
			}
			view.parent.app.Draw()
			fuzzyResults.Highlight(strconv.Itoa(fuzz.matches[0].Index))
			fuzzyResults.ScrollToBeginning()
		} else {
			fuzzyResults.Clear()
			fuzzyResults.Highlight()
		}
	})

	// callback to handle key events on fuzzy search
	fuzzySearch.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		highlights := fuzzyResults.GetHighlights()
		if event.Key() == tcell.KeyEsc {
			view.parent.views.RemovePage("fuzzy")
			return nil
		} else if event.Key() == tcell.KeyTab {
			// cycle highlighted area to next fuzzy match
			if len(highlights) > 0 {
				fuzz.selected = (fuzz.selected + 1) % len(fuzz.matches)
				fuzzyResults.Highlight(strconv.Itoa(fuzz.matches[fuzz.selected].Index))
				fuzzyResults.ScrollToHighlight()
			}
			return nil
		} else if event.Key() == tcell.KeyEnter {
			// switch room to currently selected room
			if len(highlights) > 0 {
				debug.Print("Fuzzy Selected Room:", rooms[fuzz.matches[fuzz.selected].Index].GetTitle())
				view.SwitchRoom(rooms[fuzz.matches[fuzz.selected].Index].Tags()[0].Tag, rooms[fuzz.matches[fuzz.selected].Index])
			}
			view.parent.views.RemovePage("fuzzy")
			fuzzyResults.Clear()
			fuzzySearch.SetText("")
			return nil
		}
		return event
	})

	return fuzz
}
