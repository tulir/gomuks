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

package debug

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/sasha-s/go-deadlock"
)

var writer io.Writer
var RecoverPrettyPanic bool = true
var DeadlockDetection bool
var WriteLogs bool
var OnRecover func()
var LogDirectory = GetUserDebugDir()

func GetUserDebugDir() string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "gomuks-"+getUname())
	}
	// See https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
	if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "gomuks")
	}
	home := os.Getenv("HOME")
	if home == "" {
		fmt.Println("XDG_STATE_HOME and HOME are both unset")
		os.Exit(1)
	}
	return filepath.Join(home, ".local", "state", "gomuks")
}

func getUname() string {
	currUser, err := user.Current()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return currUser.Username
}

func Initialize() {
	err := os.MkdirAll(LogDirectory, 0750)
	if err != nil {
		RecoverPrettyPanic = false
		DeadlockDetection = false
		WriteLogs = false
		return
	}

	if WriteLogs {
		writer, err = os.OpenFile(filepath.Join(LogDirectory, "debug.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			panic(err)
		}
		_, _ = fmt.Fprintf(writer, "======================= Debug init @ %s =======================\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	if DeadlockDetection {
		deadlocks, err := os.OpenFile(filepath.Join(LogDirectory, "deadlock.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			panic(err)
		}
		deadlock.Opts.LogBuf = deadlocks
		deadlock.Opts.OnPotentialDeadlock = func() {
			if OnRecover != nil {
				OnRecover()
			}
			_, _ = fmt.Fprintf(os.Stderr, "Potential deadlock detected. See %s/deadlock.log for more information.", LogDirectory)
			os.Exit(88)
		}
		_, err = fmt.Fprintf(deadlocks, "======================= Debug init @ %s =======================\n", time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			panic(err)
		}
	} else {
		deadlock.Opts.Disable = true
	}
}

func Printf(text string, args ...interface{}) {
	if writer != nil {
		_, _ = fmt.Fprintf(writer, time.Now().Format("[2006-01-02 15:04:05] "))
		_, _ = fmt.Fprintf(writer, text+"\n", args...)
	}
}

func Print(text ...interface{}) {
	if writer != nil {
		_, _ = fmt.Fprintf(writer, time.Now().Format("[2006-01-02 15:04:05] "))
		_, _ = fmt.Fprintln(writer, text...)
	}
}

func PrintStack() {
	if writer != nil {
		_, _ = writer.Write(debug.Stack())
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
             ||     ||

A fatal error has occurred.

`

func PrettyPanic(panic interface{}) {
	fmt.Print(Oops)
	traceFile := fmt.Sprintf(filepath.Join(LogDirectory, "panic-%s.txt"), time.Now().Format("2006-01-02--15-04-05"))

	var buf bytes.Buffer
	_, _ = fmt.Fprintln(&buf, panic)
	buf.Write(debug.Stack())
	err := ioutil.WriteFile(traceFile, buf.Bytes(), 0640)

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
