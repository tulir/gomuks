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

package debug

import (
	"fmt"
	"io"
	"os"
)

var writer io.Writer

func EnableExternal() {
	var err error
	writer, err = os.OpenFile("/tmp/gomuks-debug.log", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		writer = nil
	}
}

func ExtPrintf(text string, args ...interface{}) {
	if writer != nil {
		fmt.Fprintf(writer, text+"\n", args...)
	}
}

func ExtPrint(text ...interface{}) {
	if writer != nil {
		fmt.Fprintln(writer, text...)
	}
}
