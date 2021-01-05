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

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix"
)

// Gomuks is the wrapper for everything.
type Gomuks struct {
	ui     ifc.GomuksUI
	matrix *matrix.Container
	config *config.Config
	stop   chan bool
}

// NewGomuks creates a new Gomuks instance with everything initialized,
// but does not start it.
func NewGomuks(uiProvider ifc.UIProvider, configDir, dataDir, cacheDir, downloadDir string) *Gomuks {
	gmx := &Gomuks{
		stop: make(chan bool, 1),
	}

	gmx.config = config.NewConfig(configDir, dataDir, cacheDir, downloadDir)
	gmx.ui = uiProvider(gmx)
	gmx.matrix = matrix.NewContainer(gmx)

	gmx.config.LoadAll()
	gmx.ui.Init()

	debug.OnRecover = gmx.ui.Finish

	return gmx
}

func (gmx *Gomuks) Version() string {
	return "v0.2.2"
}

// Save saves the active session and message history.
func (gmx *Gomuks) Save() {
	gmx.config.SaveAll()
}

// StartAutosave calls Save() every minute until it receives a stop signal
// on the Gomuks.stop channel.
func (gmx *Gomuks) StartAutosave() {
	defer debug.Recover()
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			if gmx.config.AuthCache.InitialSyncDone {
				gmx.Save()
			}
		case val := <-gmx.stop:
			if val {
				return
			}
		}
	}
}

// Stop stops the Matrix syncer, the tview app and the autosave goroutine,
// then saves everything and calls os.Exit(0).
func (gmx *Gomuks) Stop(save bool) {
	debug.Print("Disconnecting from Matrix...")
	gmx.matrix.Stop()
	debug.Print("Cleaning up UI...")
	gmx.ui.Stop()
	gmx.stop <- true
	if save {
		gmx.Save()
	}
	os.Exit(0)
}

// Start opens a goroutine for the autosave loop and starts the tview app.
//
// If the tview app returns an error, it will be passed into panic(), which
// will be recovered as specified in Recover().
func (gmx *Gomuks) Start() {
	_ = gmx.matrix.InitClient()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		gmx.Stop(true)
	}()

	go gmx.StartAutosave()
	if err := gmx.ui.Start(); err != nil {
		panic(err)
	}
}

// Matrix returns the MatrixContainer instance.
func (gmx *Gomuks) Matrix() ifc.MatrixContainer {
	return gmx.matrix
}

// Config returns the Gomuks config instance.
func (gmx *Gomuks) Config() *config.Config {
	return gmx.config
}

// UI returns the Gomuks UI instance.
func (gmx *Gomuks) UI() ifc.GomuksUI {
	return gmx.ui
}
