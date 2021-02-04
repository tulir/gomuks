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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui"
)

var MainUIProvider ifc.UIProvider = ui.NewGomuksUI

func main() {
	debugDir := os.Getenv("DEBUG_DIR")
	if len(debugDir) > 0 {
		debug.LogDirectory = debugDir
	}
	debugLevel := strings.ToLower(os.Getenv("DEBUG"))
	if debugLevel != "0" && debugLevel != "f" && debugLevel != "false" {
		debug.WriteLogs = true
		debug.RecoverPrettyPanic = true
	}
	if debugLevel == "1" || debugLevel == "t" || debugLevel == "true" {
		debug.RecoverPrettyPanic = false
		debug.DeadlockDetection = true
	}
	debug.Initialize()
	defer debug.Recover()

	var configDir, dataDir, cacheDir, downloadDir string
	var err error

	configDir, err = UserConfigDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get config directory:", err)
		os.Exit(3)
	}
	dataDir, err = UserDataDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get data directory:", err)
		os.Exit(3)
	}
	cacheDir, err = UserCacheDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get cache directory:", err)
		os.Exit(3)
	}
	downloadDir, err = UserDownloadDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get download directory:", err)
		os.Exit(3)
	}

	gmx := NewGomuks(MainUIProvider, configDir, dataDir, cacheDir, downloadDir)

	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("gomuks version %s\n", gmx.Version())
		os.Exit(0)
	}

	gmx.Start()

	// We use os.Exit() everywhere, so exiting by returning from Start() shouldn't happen.
	time.Sleep(5 * time.Second)
	fmt.Println("Unexpected exit by return from gmx.Start().")
	os.Exit(2)
}

func getRootDir(subdir string) string {
	rootDir := os.Getenv("GOMUKS_ROOT")
	if rootDir == "" {
		return ""
	}
	dir := filepath.Join(rootDir, subdir)
	debug.Printf("root dir: %s", dir)
	return dir
}

// UserCacheDir returns GOMUKS_CACHE_HOME
// If the environment variable is blank, it returns the default cache dir /gomuks
func UserCacheDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CACHE_HOME")
	if dir == "" {
		dir = getRootDir("cache")
	}
	if dir == "" {
		dir, err = os.UserCacheDir()
		dir = filepath.Join(dir, "gomuks")
	}
	debug.Printf("cache dir: %s", dir)
	return
}

// UserDataDir returns a home directory GOMUKS_DATA_HOME
// If the environment variable is blank, it returns a default based on OS
func UserDataDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_DATA_HOME")
	if dir != "" {
		return
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return UserConfigDir()
	}
	dir = os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		dir = getRootDir("data")
	}
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
		}
		dir = filepath.Join(dir, ".local", "share")
	}
	dir = filepath.Join(dir, "gomuks")
	debug.Printf("data dir: %s", dir)
	return
}

// UserDownloadDir returns either HOME/Downloads or an error if homedir cannot be found
func UserDownloadDir() (dir string, err error) {
	dir, err = os.UserHomeDir()
	dir = filepath.Join(dir, "Downloads")
	debug.Printf("download dir: %s", dir)
	return
}

// UserConfigDir returns a configuration directory based on the GOMUKS_CONFIG_HOME
// environment variable, or the default.
func UserConfigDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CONFIG_HOME")
	if dir == "" {
		dir = getRootDir("config")
	}
	if dir == "" {
		dir, err = os.UserConfigDir()
		dir = filepath.Join(dir, "gomuks")
	}
	debug.Printf("user config dir: %s", dir)
	return
}
