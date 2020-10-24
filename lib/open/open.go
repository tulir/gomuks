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

package open

import (
	"os/exec"

	"maunium.net/go/gomuks/debug"
)

func Open(input string) error {
	cmd := exec.Command(Command, append(Args, input)...)
	err := cmd.Start()
	if err != nil {
		debug.Printf("Failed to start %s: %v", Command, err)
	} else {
		go func() {
			waitErr := cmd.Wait()
			if waitErr != nil {
				debug.Printf("Failed to run %s: %v", Command, err)
			}
		}()
	}
	return err
}
