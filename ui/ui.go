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
	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/tview"
)

type GomuksUI struct {
	gmx   ifc.Gomuks
	app   *tview.Application
	views *tview.Pages

	mainView  *MainView
	loginView *tview.Form
}

func init() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDarkGreen
}

func NewGomuksUI(gmx ifc.Gomuks) (ui *GomuksUI) {
	ui = &GomuksUI{
		gmx:   gmx,
		app:   gmx.App(),
		views: tview.NewPages(),
	}
	ui.views.SetChangedFunc(ui.Render)
	return
}

func (ui *GomuksUI) Render() {
	ui.app.Draw()
}

func (ui *GomuksUI) SetView(name ifc.View) {
	ui.views.SwitchToPage(string(name))
}

func (ui *GomuksUI) InitViews() tview.Primitive {
	ui.views.AddPage(string(ifc.ViewLogin), ui.NewLoginView(), true, true)
	ui.views.AddPage(string(ifc.ViewMain), ui.NewMainView(), true, false)
	return ui.views
}

func (ui *GomuksUI) MainView() ifc.MainView {
	return ui.mainView
}
