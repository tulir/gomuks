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

package edit

import (
	"os"
	"os/exec"
	"io/ioutil"
)

const file = "/tmp/gomuks-draft.md"

func GetText(text string) string {
	visual, exists := os.LookupEnv("VISUAL")
	if ! exists {
		visual = "vi"
	}

	f, _ := os.Create("/tmp/gomuks-draft.md")
	f.WriteString(text)

	exec.Command(os.Getenv("TERM"), visual, file).Run()

	content, _ := ioutil.ReadFile(file)

	return string(content)
}

