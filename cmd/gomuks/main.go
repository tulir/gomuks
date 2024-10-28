// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	_ "go.mau.fi/util/dbutil/litestream"
	flag "maunium.net/go/mauflag"
	"maunium.net/go/mautrix"

	"go.mau.fi/gomuks/pkg/hicli"
)

var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

const StaticVersion = "0.4.0"
const URL = "https://github.com/tulir/gomuks"

var (
	Version          string
	VersionDesc      string
	LinkifiedVersion string
	ParsedBuildTime  time.Time
)

var wantHelp, _ = flag.MakeHelpFlag()
var version = flag.MakeFull("v", "version", "View gomuks version and quit.", "false").Bool()

func main() {
	hicli.InitialDeviceDisplayName = "gomuks web"
	initVersion(Tag, Commit, BuildTime)
	flag.SetHelpTitles(
		"gomuks - A Matrix client written in Go.",
		"gomuks [-hv]",
	)
	err := flag.Parse()

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		flag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		flag.PrintHelp()
		os.Exit(0)
	} else if *version {
		fmt.Println(VersionDesc)
		os.Exit(0)
	}

	gmx := NewGomuks()
	gmx.InitDirectories()
	err = gmx.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(9)
	}
	gmx.SetupLog()
	gmx.Log.Info().
		Str("version", Version).
		Str("go_version", runtime.Version()).
		Time("built_at", ParsedBuildTime).
		Msg("Initializing gomuks")
	gmx.StartServer()
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete")
	gmx.WaitForInterrupt()
	gmx.Log.Info().Msg("Shutting down...")
	gmx.directStop()
	gmx.Log.Info().Msg("Shutdown complete")
	os.Exit(0)
}

func initVersion(tag, commit, rawBuildTime string) {
	if len(tag) > 0 && tag[0] == 'v' {
		tag = tag[1:]
	}
	if tag != StaticVersion {
		suffix := "+dev"
		if len(commit) > 8 {
			Version = fmt.Sprintf("%s%s.%s", StaticVersion, suffix, commit[:8])
		} else {
			Version = fmt.Sprintf("%s%s.unknown", StaticVersion, suffix)
		}
	} else {
		Version = StaticVersion
	}

	LinkifiedVersion = fmt.Sprintf("v%s", Version)
	if tag == Version {
		LinkifiedVersion = fmt.Sprintf("[v%s](%s/releases/v%s)", Version, URL, tag)
	} else if len(commit) > 8 {
		LinkifiedVersion = strings.Replace(LinkifiedVersion, commit[:8], fmt.Sprintf("[%s](%s/commit/%s)", commit[:8], URL, commit), 1)
	}
	if rawBuildTime != "unknown" {
		ParsedBuildTime, _ = time.Parse(time.RFC3339, rawBuildTime)
	}
	var builtWith string
	if ParsedBuildTime.IsZero() {
		rawBuildTime = "unknown"
		builtWith = runtime.Version()
	} else {
		rawBuildTime = ParsedBuildTime.Format(time.RFC1123)
		builtWith = fmt.Sprintf("built at %s with %s", rawBuildTime, runtime.Version())
	}
	mautrix.DefaultUserAgent = fmt.Sprintf("gomuks/%s %s", Version, mautrix.DefaultUserAgent)
	VersionDesc = fmt.Sprintf("gomuks %s (%s)", Version, builtWith)
	BuildTime = rawBuildTime
}
