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
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
	"os"
)

type View string

// Allowed views in GomuksUI
const (
	ViewLogin View = "login"
	ViewMain  View = "main"
)

type GomuksUI struct {
	gmx   ifc.Gomuks
	app   *tview.Application
	views *tview.Pages

	mainView  *MainView
	loginView *LoginView
}

func init() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDarkGreen
	if tcellDB := os.Getenv("TCELLDB"); len(tcellDB) == 0 {
		if info, err := os.Stat("/usr/share/tcell/database"); err == nil && info.IsDir() {
			os.Setenv("TCELLDB", "/usr/share/tcell/database")
		}
	}
}

func NewGomuksUI(gmx ifc.Gomuks) ifc.GomuksUI {
	ui := &GomuksUI{
		gmx:   gmx,
		app:   tview.NewApplication(),
		views: tview.NewPages(),
	}
	ui.views.SetChangedFunc(ui.Render)
	return ui
}

func (ui *GomuksUI) Init() {
	ui.app.SetRoot(ui.InitViews(), true)
}

func (ui *GomuksUI) Start() error {
	return ui.app.Run()
}

func (ui *GomuksUI) Stop() {
	ui.app.Stop()
}

func (ui *GomuksUI) Finish() {
	if ui.app.GetScreen() != nil {
		ui.app.GetScreen().Fini()
	}
}

func (ui *GomuksUI) Render() {
	ui.app.Draw()
}

func (ui *GomuksUI) OnLogin() {
	ui.SetView(ViewMain)
	ui.app.SetFocus(ui.mainView)
}

func (ui *GomuksUI) OnLogout() {
	ui.SetView(ViewLogin)
	ui.app.SetFocus(ui.loginView)
}

func (ui *GomuksUI) HandleNewPreferences() {
	ui.Render()
}

func (ui *GomuksUI) SetView(name View) {
	ui.views.SwitchToPage(string(name))
}

func (ui *GomuksUI) InitViews() tview.Primitive {
	ui.views.AddPage(string(ViewLogin), ui.NewLoginView(), true, true)
	ui.views.AddPage(string(ViewMain), ui.NewMainView(), true, false)
	return ui.views
}

func (ui *GomuksUI) MainView() ifc.MainView {
	return ui.mainView
}
