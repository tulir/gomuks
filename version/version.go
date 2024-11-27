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

package version

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"maunium.net/go/mautrix"
)

const StaticVersion = "0.4.0"
const URL = "https://github.com/tulir/gomuks"

var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var (
	Version          string
	Description      string
	LinkifiedVersion string
	ParsedBuildTime  time.Time
)

func init() {
	tagWithoutV := strings.TrimPrefix(Tag, "v")
	if tagWithoutV != StaticVersion {
		suffix := "+dev"
		if len(Commit) > 8 {
			Version = fmt.Sprintf("%s%s.%s", StaticVersion, suffix, Commit[:8])
		} else {
			Version = fmt.Sprintf("%s%s.unknown", StaticVersion, suffix)
		}
	} else {
		Version = StaticVersion
	}

	LinkifiedVersion = fmt.Sprintf("v%s", Version)
	if tagWithoutV == Version {
		LinkifiedVersion = fmt.Sprintf("[v%s](%s/releases/v%s)", Version, URL, tagWithoutV)
	} else if len(Commit) > 8 {
		LinkifiedVersion = strings.Replace(LinkifiedVersion, Commit[:8], fmt.Sprintf("[%s](%s/commit/%s)", Commit[:8], URL, Commit), 1)
	}
	if BuildTime != "unknown" {
		ParsedBuildTime, _ = time.Parse(time.RFC3339, BuildTime)
	}
	var builtWith string
	if ParsedBuildTime.IsZero() {
		BuildTime = "unknown"
		builtWith = runtime.Version()
	} else {
		BuildTime = ParsedBuildTime.Format(time.RFC1123)
		builtWith = fmt.Sprintf("built at %s with %s", BuildTime, runtime.Version())
	}
	mautrix.DefaultUserAgent = fmt.Sprintf("gomuks/%s %s", Version, mautrix.DefaultUserAgent)
	Description = fmt.Sprintf("gomuks %s (%s)", Version, builtWith)
}
