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

	flag "maunium.net/go/mauflag"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/version"
	"go.mau.fi/gomuks/web"
)

var wantHelp, _ = flag.MakeHelpFlag()
var wantVersion = flag.MakeFull("v", "version", "View gomuks version and quit.", "false").Bool()

func main() {
	hicli.InitialDeviceDisplayName = "gomuks web"
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
	} else if *wantVersion {
		fmt.Println(version.Description)
		os.Exit(0)
	}

	gmx := gomuks.NewGomuks()
	gmx.Version = version.Version
	gmx.Commit = version.Commit
	gmx.LinkifiedVersion = version.LinkifiedVersion
	gmx.BuildTime = version.ParsedBuildTime
	gmx.FrontendFS = web.Frontend
	gmx.Run()
}
