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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/adrg/xdg"

	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/util"
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
	}
	if debugLevel == "1" || debugLevel == "t" || debugLevel == "true" {
		debug.RecoverPrettyPanic = false
		debug.DeadlockDetection = true
	}
	debug.Initialize()
	defer debug.Recover()

	userDirs, err := xdg.SearchConfigFile("user-dirs.dirs")
	if err != nil {
		debug.Print("user-dirs.dirs not found")
	}

	if userDirs == "" {
		userDirs, err = xdg.SearchConfigFile("user-dirs.defaults")
		if err != nil {
			debug.Print("user-dirs.defaults not found")
		}
	}

	if userDirs != "" {
		err := util.LoadEnvFile(userDirs)

		if err != nil {
			debug.Print("Failed to load user-dirs file")
		}
	}

	xdg.Reload()

	configDir := UserConfigDir()
	dataDir := UserDataDir()
	cacheDir := UserCacheDir()
	downloadDir := UserDownloadDir()
	debug.Print(os.Getenv("HOME"))
	debug.Print(os.Getenv("XDG_DOWNLOAD_DIR"))
	debug.Print(downloadDir)

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
	return filepath.Join(rootDir, subdir)
}

func UserCacheDir() (dir string) {
	dir = os.Getenv("GOMUKS_CACHE_HOME")
	if dir == "" {
		dir = getRootDir("cache")
	}
	if dir == "" {
		dir = filepath.Join(xdg.CacheHome, "gomuks")
	}
	return
}

func UserDataDir() (dir string) {
	dir = os.Getenv("GOMUKS_DATA_HOME")
	if dir != "" {
		return
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return UserConfigDir()
	}

	dir = filepath.Join(xdg.DataHome, "gomuks")
	return
}

func UserDownloadDir() (dir string) {
	dir = os.Getenv("GOMUKS_DOWNLOAD_DIR")
	if dir != "" {
		return
	}

	dir = xdg.UserDirs.Download
	return
}

func UserConfigDir() (dir string) {
	dir = os.Getenv("GOMUKS_CONFIG_HOME")
	if dir == "" {
		dir = getRootDir("config")
	}
	if dir == "" {
		dir = filepath.Join(xdg.ConfigHome, "gomuks")
	}
	return
}
