// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package notification

import "os/exec"

func Send(title, text string, critical bool) error {
	args := []string{"-a", "gomuks"}
	if critical {
		args = append(args, "-p", "critical")
	}
// 	if iconPath {
// 		args = append(args, "-i", iconPath)
// 	}
	args = append(args, title, text)
	return exec.Command("notify-send", args...).Run()
}
