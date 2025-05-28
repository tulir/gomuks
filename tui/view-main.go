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

package tui

import (
	"go.mau.fi/gomuks/tui/widget"

	"go.mau.fi/mauview"
	"time"

	"maunium.net/go/mautrix/id"
)

type MainView struct {
	flex *mauview.Flex

	roomView     *mauview.Box
	focused      mauview.Focusable

	modal mauview.Component

	lastFocusTime time.Time

	parent *GomuksTUI
}

func (gt *GomuksTUI) NewMainView() mauview.Component {
	mainView := &MainView{
		flex:     mauview.NewFlex().SetDirection(mauview.FlexColumn),
		roomView: mauview.NewBox(nil).SetBorder(false),
		rooms:    make(map[id.RoomID]*RoomView),

		parent: gt,
	}
	mainView.roomList = NewRoomList(mainView)
	mainView.cmdProcessor = NewCommandProcessor(mainView)

	mainView.flex.
		AddFixedComponent(mainView.roomList, 25).
		AddFixedComponent(widget.NewBorder(), 1).
		AddProportionalComponent(mainView.roomView, 1)

	gt.mainView = mainView

	return mainView
}
