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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"runtime/debug"
)

var writer io.Writer
var RecoverPrettyPanic bool
var OnRecover func()

func init() {
	var err error
	writer, err = os.OpenFile("/tmp/gomuks-debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		writer = nil
	}
}

func Printf(text string, args ...interface{}) {
	if writer != nil {
		fmt.Fprintf(writer, time.Now().Format("[2006-01-02 15:04:05] "))
		fmt.Fprintf(writer, text+"\n", args...)
	}
}

func Print(text ...interface{}) {
	if writer != nil {
		fmt.Fprintf(writer, time.Now().Format("[2006-01-02 15:04:05] "))
		fmt.Fprintln(writer, text...)
	}
}

func PrintStack() {
	if writer != nil {
		data := debug.Stack()
		writer.Write(data)
	}
}

// Recover recovers a panic, runs the OnRecover handler and either re-panics or
// shows an user-friendly message about the panic depending on whether or not
// the pretty panic mode is enabled.
func Recover() {
	if p := recover(); p != nil {
		if OnRecover != nil {
			OnRecover()
		}
		if RecoverPrettyPanic {
			PrettyPanic(p)
		} else {
			panic(p)
		}
	}
}

const Oops = ` __________
< Oh noes! >
 ‾‾‾\‾‾‾‾‾‾
     \   ^__^
      \  (XX)\_______
         (__)\       )\/\
          U  ||----W |
             ||     ||`

func PrettyPanic(panic interface{}) {
	fmt.Println(Oops)
	fmt.Println("")
	fmt.Println("A fatal error has occurred.")
	fmt.Println("")
	traceFile := fmt.Sprintf("/tmp/gomuks-panic-%s.txt", time.Now().Format("2006-01-02--15-04-05"))

	var buf bytes.Buffer
	fmt.Fprintln(&buf, panic)
	buf.Write(debug.Stack())
	err := ioutil.WriteFile(traceFile, buf.Bytes(), 0644)

	if err != nil {
		fmt.Println("Saving the stack trace to", traceFile, "failed:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(err)
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("")
		fmt.Println("You can file an issue at https://github.com/tulir/gomuks/issues.")
		fmt.Println("Please provide the file save error (above) and the stack trace of the original error (below) when filing an issue.")
		fmt.Println("")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(panic)
		debug.PrintStack()
		fmt.Println("--------------------------------------------------------------------------------")
	} else {
		fmt.Println("The stack trace has been saved to", traceFile)
		fmt.Println("")
		fmt.Println("You can file an issue at https://github.com/tulir/gomuks/issues.")
		fmt.Println("Please provide the contents of that file when filing an issue.")
	}
	os.Exit(1)
}
