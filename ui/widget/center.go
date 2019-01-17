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

package widget

import (
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

// Center wraps the given tview primitive into a Flex element in order to
// vertically and horizontally center the given primitive.
func Center(width, height int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

type transparentCenter struct {
	*tview.Box
	prefWidth, prefHeight int
	p                     tview.Primitive
}

func TransparentCenter(width, height int, p tview.Primitive) tview.Primitive {
	return &transparentCenter{
		Box:        tview.NewBox(),
		prefWidth:  width,
		prefHeight: height,
		p:          p,
	}
}

func (tc *transparentCenter) Draw(screen tcell.Screen) {
	x, y, width, height := tc.GetRect()
	if width > tc.prefWidth {
		x += (width - tc.prefWidth) / 2
		width = tc.prefWidth
	}
	if height > tc.prefHeight {
		y += (height - tc.prefHeight) / 2
		height = tc.prefHeight
	}
	tc.p.SetRect(x, y, width, height)
	tc.p.Draw(screen)
}

func (tc *transparentCenter) Focus(delegate func(p tview.Primitive)) {
	if delegate != nil {
		delegate(tc.p)
	}
}
