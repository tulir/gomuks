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

package open

import (
	"os/exec"
)

const FileProtocolHandler = "url.dll,FileProtocolHandler"

var RunDLL32 = filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")

func Open(input string) error {
	return exec.Command(RunDLL32, FileProtocolHandler, input).Start()
}
