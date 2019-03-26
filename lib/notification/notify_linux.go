// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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

import "os/exec"

func Send(title, text string, critical, sound bool) error {
	args := []string{"-a", "gomuks"}
	if !critical {
		args = append(args, "-u", "low")
	}
	// 	if iconPath {
	// 		args = append(args, "-i", iconPath)
	// 	}
	args = append(args, title, text)
	if sound {
		soundName := "message-new-instant"
		if critical {
			soundName = "complete"
		}
		exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/"+soundName+".oga").Run()
	}
	return exec.Command("notify-send", args...).Run()
}
