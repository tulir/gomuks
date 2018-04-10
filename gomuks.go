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

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
	"maunium.net/go/tview"
)

// Gomuks is the wrapper for everything.
type Gomuks struct {
	app       *tview.Application
	ui        *ui.GomuksUI
	matrix    *matrix.Container
	debugMode bool
	config    *config.Config
	stop      chan bool
}

// NewGomuks creates a new Gomuks instance with everything initialized,
// but does not start it.
func NewGomuks(enableDebug bool) *Gomuks {
	configDir := filepath.Join(os.Getenv("HOME"), ".config/gomuks")
	gmx := &Gomuks{
		app:       tview.NewApplication(),
		stop:      make(chan bool, 1),
		debugMode: enableDebug,
	}

	gmx.config = config.NewConfig(configDir)
	gmx.ui = ui.NewGomuksUI(gmx)
	gmx.matrix = matrix.NewContainer(gmx)

	gmx.config.Load()
	if len(gmx.config.UserID) > 0 {
		_ = gmx.config.LoadSession(gmx.config.UserID)
	}

	_ = gmx.matrix.InitClient()

	main := gmx.ui.InitViews()
	gmx.app.SetRoot(main, true)

	return gmx
}

// Save saves the active session and message history.
func (gmx *Gomuks) Save() {
	if gmx.config.Session != nil {
		debug.Print("Saving session...")
		_ = gmx.config.Session.Save()
	}
	debug.Print("Saving history...")
	gmx.ui.MainView().SaveAllHistory()
}

// StartAutosave calls Save() every minute until it receives a stop signal
// on the Gomuks.stop channel.
func (gmx *Gomuks) StartAutosave() {
	defer gmx.Recover()
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			gmx.Save()
		case val := <-gmx.stop:
			if val {
				return
			}
		}
	}
}

// Stop stops the Matrix syncer, the tview app and the autosave goroutine,
// then saves everything and calls os.Exit(0).
func (gmx *Gomuks) Stop() {
	debug.Print("Disconnecting from Matrix...")
	gmx.matrix.Stop()
	debug.Print("Cleaning up UI...")
	gmx.app.Stop()
	gmx.stop <- true
	gmx.Save()
	os.Exit(0)
}

// Recover recovers a panic, closes the tcell screen and either re-panics or
// shows an user-friendly message about the panic depending on whether or not
// the debug mode is enabled.
func (gmx *Gomuks) Recover() {
	if p := recover(); p != nil {
		if gmx.App().GetScreen() != nil {
			gmx.App().GetScreen().Fini()
		}
		if gmx.debugMode {
			panic(p)
		} else {
			debug.PrettyPanic(p)
		}
	}
}

// Start opens a goroutine for the autosave loop and starts the tview app.
//
// If the tview app returns an error, it will be passed into panic(), which
// will be recovered as specified in Recover().
func (gmx *Gomuks) Start() {
	defer gmx.Recover()
	go gmx.StartAutosave()
	if err := gmx.app.Run(); err != nil {
		panic(err)
	}
}

// Matrix returns the MatrixContainer instance.
func (gmx *Gomuks) Matrix() ifc.MatrixContainer {
	return gmx.matrix
}

// App returns the tview Application instance.
func (gmx *Gomuks) App() *tview.Application {
	return gmx.app
}

// Config returns the Gomuks config instance.
func (gmx *Gomuks) Config() *config.Config {
	return gmx.config
}

// UI returns the Gomuks UI instance.
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
