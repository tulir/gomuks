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

	flag "maunium.net/go/mauflag"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/initialize"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
)

// Information to find out exactly which commit gomuks was built from.
// These are filled at build time with the -X linker flag.
var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var (
	// Version is the version number of gomuks. Changed manually when making a release.
	Version = "0.3.0"
	// VersionString is the gomuks version, plus commit information. Filled in init() using the build-time values.
	VersionString = ""
)

func init() {
	if len(Tag) > 0 && Tag[0] == 'v' {
		Tag = Tag[1:]
	}
	if Tag != Version {
		suffix := ""
		if !strings.HasSuffix(Version, "+dev") {
			suffix = "+dev"
		}
		if len(Commit) > 8 {
			Version = fmt.Sprintf("%s%s.%s", Version, suffix, Commit[:8])
		} else {
			Version = fmt.Sprintf("%s%s.unknown", Version, suffix)
		}
	}
	VersionString = fmt.Sprintf("gomuks %s (%s with %s)", Version, BuildTime, runtime.Version())
}

var MainUIProvider ifc.UIProvider = ui.NewGomuksUI

var wantVersion = flag.MakeFull("v", "version", "Show the version of gomuks", "false").Bool()
var clearCache = flag.MakeFull("c", "clear-cache", "Clear the cache directory instead of starting", "false").Bool()
var skipVersionCheck = flag.MakeFull("s", "skip-version-check", "Skip the homeserver version checks at startup and login", "false").Bool()
var clearData = flag.Make().LongKey("clear-all-data").Usage("Clear all data instead of starting").Default("false").Bool()
var logInForTransfer = flag.Make().LongKey("log-in-for-transfer").Usage("Log in and generate packaged data for transfer").Default("false").Bool()
var wantHelp, _ = flag.MakeHelpFlag()

func main() {
	flag.SetHelpTitles(
		"gomuks - A terminal Matrix client written in Go.",
		"gomuks [-vcsh] [--clear-all-data|--log-in-for-transfer]",
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
	if *logInForTransfer {
		if currentDir, err := os.Getwd(); err == nil {
			pack := filepath.Join(currentDir, "transfer")
			if _, err := os.Stat(pack); err == nil {
				fmt.Println("with the --log-in-for-transfer flag, gomuks packs your data up into")
				fmt.Println("the transfer/ directory so you can move it around easily. please make")
				fmt.Println("sure there is nothing there already, and then run it again.")
				os.Exit(1)
			}

			keys := filepath.Join(currentDir, "keys.txt")
			if _, err := os.Stat(keys); err != nil {
				fmt.Println("with the --log-in-for-transfer flag, gomuks packs your data up so")
				fmt.Println("you can move it around easily. please export your existing client")
				fmt.Println("keys to the file keys.txt, and then run gomuks again.")
				os.Exit(1)
			}

			os.Setenv("GOMUKS_ROOT", pack)
		}
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

	configDir, err = initialize.UserConfigDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get config directory:", err)
		os.Exit(3)
	}
	dataDir, err = initialize.UserDataDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get data directory:", err)
		os.Exit(3)
	}
	cacheDir, err = initialize.UserCacheDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get cache directory:", err)
		os.Exit(3)
	}
	downloadDir, err = initialize.UserDownloadDir()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to get download directory:", err)
		os.Exit(3)
	}

	debug.Print("Config directory:", configDir)
	debug.Print("Data directory:", dataDir)
	debug.Print("Cache directory:", cacheDir)
	debug.Print("Download directory:", downloadDir)

	matrix.SkipVersionCheck = *skipVersionCheck
	gmx := initialize.NewGomuks(MainUIProvider, configDir, dataDir, cacheDir, downloadDir, *logInForTransfer)

	if *clearCache {
		debug.Print("Clearing cache as requested by CLI flag")
		gmx.Config().Clear()
		fmt.Printf("Cleared cache at %s\n", gmx.Config().CacheDir)
		return
	} else if *clearData {
		debug.Print("Clearing all data as requested by CLI flag")
		gmx.Config().Clear()
		gmx.Config().ClearData()
		_ = os.RemoveAll(gmx.Config().Dir)
		fmt.Printf("Cleared cache at %s, data at %s and config at %s\n", gmx.Config().CacheDir, gmx.Config().DataDir, gmx.Config().Dir)
		return
	} else if *logInForTransfer {
		debug.Print("Initializing in headless mode as requested by CLI flag")
	}

	gmx.Start()

	// We use os.Exit() everywhere, so exiting by returning from Start() shouldn't happen.
	time.Sleep(5 * time.Second)
	fmt.Println("Unexpected exit by return from gmx.Start().")
	os.Exit(2)
}
