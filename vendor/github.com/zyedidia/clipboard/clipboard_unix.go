// Copyright 2013 @atotto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build freebsd linux netbsd openbsd solaris

package clipboard

import "os/exec"

const (
	xsel  = "xsel"
	xclip = "xclip"
)

var (
	internalClipboards map[string]string
)

func init() {
	if _, err := exec.LookPath(xclip); err == nil {
		if err := exec.Command("xclip", "-o").Run(); err == nil {
			return
		}
	}
	if _, err := exec.LookPath(xsel); err == nil {
		if err := exec.Command("xsel").Run(); err == nil {
			return
		}
	}

	internalClipboards = make(map[string]string)
	Unsupported = true
}

func copyCommand(register string) []string {
	if _, err := exec.LookPath(xclip); err == nil {
		return []string{xclip, "-in", "-selection", register}
	}

	if _, err := exec.LookPath(xsel); err == nil {
		return []string{xsel, "--input", "--" + register}
	}

	return []string{}
}
func pasteCommand(register string) []string {
	if _, err := exec.LookPath(xclip); err == nil {
		return []string{xclip, "-out", "-selection", register}
	}

	if _, err := exec.LookPath(xsel); err == nil {
		return []string{xsel, "--output", "--" + register}
	}

	return []string{}
}

func getPasteCommand(register string) *exec.Cmd {
	pasteCmdArgs := pasteCommand(register)
	return exec.Command(pasteCmdArgs[0], pasteCmdArgs[1:]...)
}

func getCopyCommand(register string) *exec.Cmd {
	copyCmdArgs := copyCommand(register)
	return exec.Command(copyCmdArgs[0], copyCmdArgs[1:]...)
}

func readAll(register string) (string, error) {
	if Unsupported {
		if text, ok := internalClipboards[register]; ok {
			return text, nil
		}
		return "", nil
	}
	pasteCmd := getPasteCommand(register)
	out, err := pasteCmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func writeAll(text string, register string) error {
	if Unsupported {
		internalClipboards[register] = text
		return nil
	}
	copyCmd := getCopyCommand(register)
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write([]byte(text)); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}
