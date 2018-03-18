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
	"os"
	"path/filepath"

	"maunium.net/go/gomatrix"
	"maunium.net/go/tview"
)

type Gomuks interface {
	Debug() DebugPrinter
	Matrix() *gomatrix.Client
	MatrixContainer() *MatrixContainer
	App() *tview.Application
	UI() *GomuksUI
	Config() *Config

	Start()
	Stop()
	Recover()
}

type gomuks struct {
	app    *tview.Application
	ui     *GomuksUI
	matrix *MatrixContainer
	debug  *DebugPane
	config *Config
}

var gdebug DebugPrinter

func NewGomuks(debug bool) *gomuks {
	configDir := filepath.Join(os.Getenv("HOME"), ".config/gomuks")
	gmx := &gomuks{
		app: tview.NewApplication(),
	}
	gmx.debug = NewDebugPane(gmx)
	gdebug = gmx.debug
	gmx.config = NewConfig(gmx, configDir)
	gmx.ui = NewGomuksUI(gmx)
	gmx.matrix = NewMatrixContainer(gmx)
	gmx.ui.matrix = gmx.matrix

	gmx.config.Load()
	if len(gmx.config.MXID) > 0 {
		gmx.config.LoadSession(gmx.config.MXID)
	}

	gmx.matrix.InitClient()

	main := gmx.ui.InitViews()
	if debug {
		main = gmx.debug.Wrap(main)
	}
	gmx.app.SetRoot(main, true)

	return gmx
}

func (gmx *gomuks) Stop() {
	gmx.debug.Print("Disconnecting from Matrix...")
	gmx.matrix.Stop()
	gmx.debug.Print("Cleaning up UI...")
	gmx.app.Stop()
	if gmx.config.Session != nil {
		gmx.debug.Print("Saving session...")
		gmx.config.Session.Save()
	}
}

func (gmx *gomuks) Recover() {
	if p := recover(); p != nil {
		if gmx.App().GetScreen() != nil {
			gmx.App().GetScreen().Fini()
		}
		panic(p)
	}
}

func (gmx *gomuks) Start() {
	if err := gmx.app.Run(); err != nil {
		panic(err)
	}
}

func (gmx *gomuks) Debug() DebugPrinter {
	return gmx.debug
}

func (gmx *gomuks) Matrix() *gomatrix.Client {
	return gmx.matrix.client
}

func (gmx *gomuks) MatrixContainer() *MatrixContainer {
	return gmx.matrix
}

func (gmx *gomuks) App() *tview.Application {
	return gmx.app
}

func (gmx *gomuks) Config() *Config {
	return gmx.config
}

func (gmx *gomuks) UI() *GomuksUI {
	return gmx.ui
}

func main() {
	debug := os.Getenv("DEBUG")
	NewGomuks(len(debug) > 0).Start()
}
