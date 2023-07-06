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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	flag "maunium.net/go/mauflag"

	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
)

var MainUIProvider ifc.UIProvider = ui.NewGomuksUI

var wantVersion = flag.MakeFull("v", "version", "Show the version of gomuks", "false").Bool()
var clearCache = flag.MakeFull("c", "clear-cache", "Clear the cache directory instead of starting", "false").Bool()
var clearData = flag.Make().LongKey("clear-all-data").Usage("Clear all data instead of starting").Default("false").Bool()
var skipVersionCheck = flag.MakeFull("s", "skip-version-check", "Skip the homeserver version checks at startup and login", "false").Bool()
var wantHelp, _ = flag.MakeHelpFlag()

func main() {
	flag.SetHelpTitles(
		"gomuks - A terminal Matrix client written in Go.",
		"gomuks [-vch] [--clear-all-data]",
	)
	err := flag.Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else if *wantHelp {
		flag.PrintHelp()
		return
	} else if *wantVersion {
		fmt.Println(VersionString)
		return
	}
	debugDir := os.Getenv("DEBUG_DIR")
	if len(debugDir) > 0 {
		debug.LogDirectory = debugDir
	}
	debugLevel := strings.ToLower(os.Getenv("DEBUG"))
	if debugLevel == "1" || debugLevel == "t" || debugLevel == "true" {
		debug.RecoverPrettyPanic = false
		debug.DeadlockDetection = true
		debug.WriteLogs = true
	}
	debug.Initialize()
	defer debug.Recover()

	var configDir, dataDir, cacheDir, downloadDir string

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

	debug.Print("Config directory:", configDir)
	debug.Print("Data directory:", dataDir)
	debug.Print("Cache directory:", cacheDir)
	debug.Print("Download directory:", downloadDir)

	matrix.SkipVersionCheck = *skipVersionCheck
	gmx := NewGomuks(MainUIProvider, configDir, dataDir, cacheDir, downloadDir)

	if *clearCache {
		debug.Print("Clearing cache as requested by CLI flag")
		gmx.config.Clear()
		fmt.Printf("Cleared cache at %s\n", gmx.config.CacheDir)
		return
	} else if *clearData {
		debug.Print("Clearing all data as requested by CLI flag")
		gmx.config.Clear()
		gmx.config.ClearData()
		_ = os.RemoveAll(gmx.config.Dir)
		fmt.Printf("Cleared cache at %s, data at %s and config at %s\n", gmx.config.CacheDir, gmx.config.DataDir, gmx.config.Dir)
		return
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

func UserCacheDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CACHE_HOME")
	if dir == "" {
		dir = getRootDir("cache")
	}
	if dir == "" {
		dir, err = os.UserCacheDir()
		dir = filepath.Join(dir, "gomuks")
	}
	return
}

func UserDataDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_DATA_HOME")
	if dir != "" {
		return
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return UserConfigDir()
	}
	dir = getRootDir("data")
	if dir == "" {
		dir = os.Getenv("XDG_DATA_HOME")
	}
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_DATA_HOME nor $HOME are defined")
		}
		dir = filepath.Join(dir, ".local", "share")
	}
	dir = filepath.Join(dir, "gomuks")
	return
}

func getXDGUserDir(name string) (dir string, err error) {
	cmd := exec.Command("xdg-user-dir", name)
	var out strings.Builder
	cmd.Stdout = &out
	err = cmd.Run()
	dir = strings.TrimSpace(out.String())
	return
}

func UserDownloadDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_DOWNLOAD_HOME")
	if dir != "" {
		return
	}
	dir, _ = getXDGUserDir("DOWNLOAD")
	if dir != "" {
		return
	}
	dir, err = os.UserHomeDir()
	dir = filepath.Join(dir, "Downloads")
	return
}

func UserConfigDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CONFIG_HOME")
	if dir == "" {
		dir = getRootDir("config")
	}
	if dir == "" {
		dir, err = os.UserConfigDir()
		dir = filepath.Join(dir, "gomuks")
	}
	return
}
