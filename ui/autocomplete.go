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

package ui

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func autocompleteFile(cmd *CommandAutocomplete) (completions []string, newText string) {
	inputPath, err := filepath.Abs(cmd.RawArgs)
	if err != nil {
		return
	}

	var searchNamePrefix, searchDir string
	if strings.HasSuffix(cmd.RawArgs, "/") {
		searchDir = inputPath
	} else {
		searchNamePrefix = filepath.Base(inputPath)
		searchDir = filepath.Dir(inputPath)
	}
	files, err := ioutil.ReadDir(searchDir)
	if err != nil {
		return
	}
	for _, file := range files {
		name := file.Name()
		if !strings.HasPrefix(name, searchNamePrefix) || (name[0] == '.' && searchNamePrefix == "") {
			continue
		}
		fullPath := filepath.Join(searchDir, name)
		if file.IsDir() {
			fullPath += "/"
		}
		completions = append(completions, fullPath)
	}
	if len(completions) == 1 {
		newText = fmt.Sprintf("/%s %s", cmd.OrigCommand, completions[0])
	}
	return
}

func autocompleteToggle(cmd *CommandAutocomplete) (completions []string, newText string) {
	completions = make([]string, 0, len(toggleMsg))
	for k := range toggleMsg {
		if strings.HasPrefix(k, cmd.RawArgs) {
			completions = append(completions, k)
		}
	}
	if len(completions) == 1 {
		newText = fmt.Sprintf("/%s %s", cmd.OrigCommand, completions[0])
	}
	return
}

var staticPowerLevelKeys = []string{"ban", "kick", "redact", "invite", "state_default", "events_default", "users_default"}

func autocompletePowerLevel(cmd *CommandAutocomplete) (completions []string, newText string) {
	if len(cmd.Args) > 1 {
		return
	}
	for _, staticKey := range staticPowerLevelKeys {
		if strings.HasPrefix(staticKey, cmd.RawArgs) {
			completions = append(completions, staticKey)
		}
	}
	for _, cpl := range cmd.Room.AutocompleteUser(cmd.RawArgs) {
		completions = append(completions, cpl.id)
	}
	return
}
