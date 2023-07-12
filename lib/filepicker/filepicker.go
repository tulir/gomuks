// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2022 Tulir Asokan
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

package filepicker

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// var zenity string
var file_browser string

func init() {
	// zenity, _ = exec.LookPath("zenity")

	// TODO: read config from user to try and load file-browser
	file_browser, _ = exec.LookPath("ranger")
}

func IsSupported() bool {
	return len(file_browser) > 0
}

func Open(confDir string) (string, error) {

	// TODO: what priority rules should we have between zenity (gui
	// browser) and ranger (terminal browser)?
	// cmd := exec.Command(zenity, "--file-selection")

	// TODO: upload dir config:
	// - allow user to set a `upload_dir string` and if not defined
	//   we just boot `ranger` in user's $HOME.
	// - also offer config option to allow custom bootup-path?
	default_upload_dir := os.Getenv("HOME")
	upload_history_file := "_last_upload_file.txt"
	upload_history_path := filepath.Join(confDir, upload_history_file)

	flag := fmt.Sprintf(
		"--choosefile=%s",
		upload_history_path,
	)

	cmd := exec.Command(
		file_browser,
		default_upload_dir,
		flag,
	)

	// var output bytes.Buffer
	var errout bytes.Buffer
	cmd.Stderr = &errout
	// cmd.Stdout = &output
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	// var last_path string
	bpath, err := ioutil.ReadFile(
		upload_history_path,
	)
	if err != nil {
		panic(err)
	}
	last_path := string(bpath[:])
	// if err != nil {
	// 	return "", err
	// }
	// defer file.Close()

	// nbytes, err := file.Read(output)
	// if err != nil {
	// 	return "", err
	// }
	// output, err := strings.TrimSpace(output.String()), nil
	fmt.Print(last_path)
	return last_path, err
}
