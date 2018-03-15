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

package main

import (
	"github.com/gdamore/tcell"
	"maunium.net/go/tview"
)

// Allowed views in GomuksUI
const (
	ViewLogin = "login"
	ViewMain  = "main"
)

type GomuksUI struct {
	gmx    Gomuks
	app    *tview.Application
	matrix *MatrixContainer
	debug  DebugPrinter
	config *Config
	views  *tview.Pages

	mainView         *MainView
}

func init() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDefault
}

func NewGomuksUI(gmx Gomuks) (ui *GomuksUI) {
	ui = &GomuksUI{
		gmx:    gmx,
		app:    gmx.App(),
		matrix: gmx.MatrixContainer(),
		debug:  gmx.Debug(),
		config: gmx.Config(),
		views:  tview.NewPages(),
	}
	ui.views.SetChangedFunc(ui.Render)
	return
}

func (ui *GomuksUI) Render() {
	ui.app.Draw()
}

func (ui *GomuksUI) SetView(name string) {
	ui.views.SwitchToPage(name)
}

func (ui *GomuksUI) InitViews() tview.Primitive {
	ui.mainView = ui.NewMainView()
	ui.views.AddPage(ViewLogin, ui.MakeLoginUI(), true, true)
	ui.views.AddPage(ViewMain, ui.mainView, true, false)
	return ui.views
}

func (ui *GomuksUI) MainView() *MainView {
	return ui.mainView
}
