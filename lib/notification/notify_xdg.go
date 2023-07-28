//go:build !windows && !darwin

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

package notification

import (
	"os"
	"os/exec"
)

var notifySendPath string
var audioCommand string
var tryAudioCommands = []string{"ogg123", "paplay"}
var soundNormal = "/usr/share/sounds/freedesktop/stereo/message-new-instant.oga"
var soundCritical = "/usr/share/sounds/freedesktop/stereo/complete.oga"

func getSoundPath(env, defaultPath string) string {
	if path, ok := os.LookupEnv(env); ok {
		// Sound file overriden by environment
		return path
	} else if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		// Sound file doesn't exist, disable it
		return ""
	} else {
		// Default sound file exists and wasn't overridden by environment
		return defaultPath
	}
}

func init() {
	var err error

	if notifySendPath, err = exec.LookPath("notify-send"); err != nil {
		return
	}

	for _, cmd := range tryAudioCommands {
		if audioCommand, err = exec.LookPath(cmd); err == nil {
			break
		}
	}
	soundNormal = getSoundPath("GOMUKS_SOUND_NORMAL", soundNormal)
	soundCritical = getSoundPath("GOMUKS_SOUND_CRITICAL", soundCritical)
}

func Send(title, text string, critical, sound bool) error {
	if len(notifySendPath) == 0 {
		return nil
	}

	args := []string{"-a", "gomuks"}
	if !critical {
		args = append(args, "-u", "low")
	}
	//if iconPath {
	//	args = append(args, "-i", iconPath)
	//}
	args = append(args, title, text)
	if sound && len(audioCommand) > 0 && len(soundNormal) > 0 {
		audioFile := soundNormal
		if critical && len(soundCritical) > 0 {
			audioFile = soundCritical
		}
		go func() {
			_ = exec.Command(audioCommand, audioFile).Run()
		}()
	}
	return exec.Command(notifySendPath, args...).Run()
}
