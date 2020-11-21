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
	"os"

	"maunium.net/go/mauview"
	"github.com/zyedidia/clipboard"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/interface"
)

type View string

// Allowed views in GomuksUI
const (
	ViewLogin View = "login"
	ViewMain  View = "main"
)

type GomuksUI struct {
	gmx ifc.Gomuks
	app *mauview.Application

	mainView  *MainView
	loginView *LoginView

	views map[View]mauview.Component
}

func init() {
	mauview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	mauview.Styles.ContrastBackgroundColor = tcell.ColorDarkGreen
	if tcellDB := os.Getenv("TCELLDB"); len(tcellDB) == 0 {
		if info, err := os.Stat("/usr/share/tcell/database"); err == nil && info.IsDir() {
			os.Setenv("TCELLDB", "/usr/share/tcell/database")
		}
	}
}

func NewGomuksUI(gmx ifc.Gomuks) ifc.GomuksUI {
	ui := &GomuksUI{
		gmx: gmx,
		app: mauview.NewApplication(),
	}
	return ui
}

func (ui *GomuksUI) Init() {
	clipboard.Initialize()
	ui.views = map[View]mauview.Component{
		ViewLogin: ui.NewLoginView(),
		ViewMain:  ui.NewMainView(),
	}
	ui.SetView(ViewLogin)
}

func (ui *GomuksUI) Start() error {
	return ui.app.Start()
}

func (ui *GomuksUI) Stop() {
	ui.app.Stop()
}

func (ui *GomuksUI) Finish() {
	if ui.app.Screen() != nil {
		ui.app.Screen().Fini()
	}
}

func (ui *GomuksUI) Render() {
	ui.app.Redraw()
}

func (ui *GomuksUI) OnLogin() {
	ui.SetView(ViewMain)
}

func (ui *GomuksUI) OnLogout() {
	ui.SetView(ViewLogin)
}

func (ui *GomuksUI) HandleNewPreferences() {
	ui.Render()
}

func (ui *GomuksUI) SetView(name View) {
	ui.app.Root = ui.views[name]
	focusable, ok := ui.app.Root.(mauview.Focusable)
	if ok {
		focusable.Focus()
	}
	if ui.app.Screen() != nil {
		ui.app.Screen().Clear()
		ui.Render()
	}
}

func (ui *GomuksUI) MainView() ifc.MainView {
	return ui.mainView
}
