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
	"os/exec"

	"github.com/zyedidia/clipboard"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	ifc "maunium.net/go/gomuks/interface"
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
	mauview.Styles.PrimaryTextColor = tcell.ColorDefault
	mauview.Styles.BorderColor = tcell.ColorDefault
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
	mauview.Backspace2RemovesWord = ui.gmx.Config().Backspace2RemovesWord
	mauview.Backspace1RemovesWord = ui.gmx.Config().Backspace1RemovesWord
	ui.app.SetAlwaysClear(ui.gmx.Config().AlwaysClearScreen)
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
	ui.app.ForceStop()
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
	ui.app.SetRoot(ui.views[name])
}

func (ui *GomuksUI) MainView() ifc.MainView {
	return ui.mainView
}

func (ui *GomuksUI) RunExternal(executablePath string, args ...string) error {
	callback := make(chan error)
	ui.app.Suspend(func() {
		cmd := exec.Command(executablePath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()
		callback <- cmd.Run()
	})
	return <-callback
}
