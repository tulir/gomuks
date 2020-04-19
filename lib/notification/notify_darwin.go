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

package notification

import (
	"fmt"
	"os/exec"
	"strings"
)

var TerminalNotifierAvailable = false

func init() {
	if err := exec.Command("which", "terminal-notifier").Run(); err != nil {
		TerminalNotifierAvailable = false
	}
	TerminalNotifierAvailable = true
}

func Send(title, text string, critical, sound bool) error {
	if TerminalNotifierAvailable {
		args := []string{"-title", "gomuks", "-subtitle", title, "-message", text}
		if critical {
			args = append(args, "-timeout", "15")
		} else {
			args = append(args, "-timeout", "4")
		}
		if sound {
			args = append(args, "-sound", "default")
		}
		// 		if len(iconPath) > 0 {
		// 			args = append(args, "-appIcon", iconPath)
		// 		}
		return exec.Command("terminal-notifier", args...).Run()
	}
	title = strings.Replace(title, `"`, `\"`, -1)
	text = strings.Replace(text, `"`, `\"`, -1)
	notification := fmt.Sprintf("display notification \"%s\" with title \"gomuks\" subtitle \"%s\"", text, title)
	return exec.Command("osascript", "-e", notification).Run()
}
