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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"maunium.net/go/gomuks/debug"
)

// Config contains the main config of gomuks.
type Config struct {
	UserID string `yaml:"mxid"`
	HS     string `yaml:"homeserver"`

	Dir        string   `yaml:"-"`
	HistoryDir string   `yaml:"history_dir"`
	MediaDir   string   `yaml:"media_dir"`
	Session    *Session `yaml:"-"`
}

// NewConfig creates a config that loads data from the given directory.
func NewConfig(dir string) *Config {
	return &Config{
		Dir:        dir,
		HistoryDir: filepath.Join(dir, "history"),
		MediaDir:   filepath.Join(dir, "media"),
	}
}

// Clear clears the session cache and removes all history.
func (config *Config) Clear() {
	if config.Session != nil {
		config.Session.Clear()
	}
	os.RemoveAll(config.HistoryDir)
}

// Load loads the config from config.yaml in the directory given to the config struct.
func (config *Config) Load() {
	os.MkdirAll(config.Dir, 0700)
	os.MkdirAll(config.HistoryDir, 0700)
	configPath := filepath.Join(config.Dir, "config.yaml")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		} else {
			fmt.Println("Failed to read config from", configPath)
			panic(err)
		}
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Failed to parse config at", configPath)
		panic(err)
	}
}

// Save saves this config to config.yaml in the directory given to the config struct.
func (config *Config) Save() {
	os.MkdirAll(config.Dir, 0700)
	data, err := yaml.Marshal(&config)
	if err != nil {
		debug.Print("Failed to marshal config")
		panic(err)
	}

	path := filepath.Join(config.Dir, "config.yaml")
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write config to", path)
		panic(err)
	}
}
