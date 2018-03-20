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
	"maunium.net/go/gomuks/ui/debug"
)

type Config struct {
	MXID string `yaml:"mxid"`
	HS   string `yaml:"homeserver"`

	dir     string   `yaml:"-"`
	Session *Session `yaml:"-"`
}

func NewConfig(dir string) *Config {
	return &Config{
		dir: dir,
	}
}

func (config *Config) Load() {
	os.MkdirAll(config.dir, 0700)
	configPath := filepath.Join(config.dir, "config.yaml")
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

func (config *Config) Save() {
	os.MkdirAll(config.dir, 0700)
	data, err := yaml.Marshal(&config)
	if err != nil {
		debug.Print("Failed to marshal config")
		panic(err)
	}

	path := filepath.Join(config.dir, "config.yaml")
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write config to", path)
		panic(err)
	}
}
