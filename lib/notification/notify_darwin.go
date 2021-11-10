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
)

var terminalNotifierAvailable = false

func init() {
	if err := exec.Command("which", "terminal-notifier").Run(); err != nil {
		terminalNotifierAvailable = false
	}
	terminalNotifierAvailable = true
}

const sendScript = `on run {notifText, notifTitle}
	display notification notifText with title "gomuks" subtitle notifTitle
end run`

func Send(title, text string, critical, sound bool) error {
	if terminalNotifierAvailable {
		args := []string{"-title", "gomuks", "-subtitle", title, "-message", text}
		if critical {
			args = append(args, "-timeout", "15")
		} else {
			args = append(args, "-timeout", "4")
		}
		if sound {
			args = append(args, "-sound", "default")
		}
		//if len(iconPath) > 0 {
		//	args = append(args, "-appIcon", iconPath)
		//}
		return exec.Command("terminal-notifier", args...).Run()
	}
	cmd := exec.Command("osascript", "-", text, title)
	if stdin, err := cmd.StdinPipe(); err != nil {
		return fmt.Errorf("failed to get stdin pipe for osascript: %w", err)
	} else if _, err = stdin.Write([]byte(sendScript)); err != nil {
		return fmt.Errorf("failed to write notification script to osascript: %w", err)
	} else if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to run notification script: %w", err)
	} else if !cmd.ProcessState.Success() {
		return fmt.Errorf("notification script exited unsuccessfully")
	} else {
		return nil
	}
}
