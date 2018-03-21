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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/tview"
)

type Gomuks struct {
	app       *tview.Application
	ui        *ui.GomuksUI
	matrix    *matrix.Container
	debug     *debug.Pane
	debugMode bool
	config    *config.Config
}

func NewGomuks(enableDebug bool) *Gomuks {
	configDir := filepath.Join(os.Getenv("HOME"), ".config/gomuks")
	gmx := &Gomuks{
		app: tview.NewApplication(),
	}

	gmx.debug = debug.NewPane()
	gmx.debug.SetChangedFunc(func() {
		gmx.ui.Render()
	})
	debug.Default = gmx.debug

	gmx.config = config.NewConfig(configDir)
	gmx.ui = ui.NewGomuksUI(gmx)
	gmx.matrix = matrix.NewContainer(gmx)

	gmx.config.Load()
	if len(gmx.config.UserID) > 0 {
		gmx.config.LoadSession(gmx.config.UserID)
	}

	gmx.matrix.InitClient()

	main := gmx.ui.InitViews()
	if enableDebug {
		debug.EnableExternal()
		main = gmx.debug.Wrap(main)
		gmx.debugMode = true
	}
	gmx.app.SetRoot(main, true)

	return gmx
}

func (gmx *Gomuks) Stop() {
	gmx.debug.Print("Disconnecting from Matrix...")
	gmx.matrix.Stop()
	gmx.debug.Print("Cleaning up UI...")
	gmx.app.Stop()
	if gmx.config.Session != nil {
		gmx.debug.Print("Saving session...")
		gmx.config.Session.Save()
	}
	os.Exit(0)
}

func (gmx *Gomuks) Recover() {
	if p := recover(); p != nil {
		if gmx.App().GetScreen() != nil {
			gmx.App().GetScreen().Fini()
		}
		if gmx.debugMode {
			panic(p)
		} else {
			debug.PrettyPanic()
		}
	}
}

func (gmx *Gomuks) Start() {
	defer gmx.Recover()
	if err := gmx.app.Run(); err != nil {
		panic(err)
	}
}

func (gmx *Gomuks) Matrix() *gomatrix.Client {
	return gmx.matrix.Client()
}

func (gmx *Gomuks) MatrixContainer() ifc.MatrixContainer {
	return gmx.matrix
}

func (gmx *Gomuks) App() *tview.Application {
	return gmx.app
}

func (gmx *Gomuks) Config() *config.Config {
	return gmx.config
}

func (gmx *Gomuks) UI() ifc.GomuksUI {
	return gmx.ui
}

func main() {
	enableDebug := len(os.Getenv("DEBUG")) > 0
	NewGomuks(enableDebug).Start()

	// We use os.Exit() everywhere, so exiting by returning from Start() shouldn't happen.
	time.Sleep(5 * time.Second)
	fmt.Println("Unexpected exit by return from Gomuks#Start().")
	os.Exit(2)
}
