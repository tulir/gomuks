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
	"io/ioutil"
	"os"
	"path/filepath"

	"encoding/json"
	"gopkg.in/yaml.v2"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
	"strings"
)

// Config contains the main config of gomuks.
type Config struct {
	UserID      string `yaml:"mxid"`
	AccessToken string `yaml:"access_token"`
	HS          string `yaml:"homeserver"`

	Dir        string `yaml:"-"`
	CacheDir   string `yaml:"cache_dir"`
	HistoryDir string `yaml:"history_dir"`
	MediaDir   string `yaml:"media_dir"`
	StateDir   string `yaml:"state_dir"`

	AuthCache struct {
		NextBatch       string `yaml:"next_batch"`
		FilterID        string `yaml:"filter_id"`
		InitialSyncDone bool   `yaml:"initial_sync_done"`
	} `yaml:"-"`

	Rooms     map[string]*rooms.Room `yaml:"-"`
	PushRules *pushrules.PushRuleset `yaml:"-"`

	nosave bool
}

// NewConfig creates a config that loads data from the given directory.
func NewConfig(configDir, cacheDir string) *Config {
	return &Config{
		Dir:        configDir,
		CacheDir:   cacheDir,
		HistoryDir: filepath.Join(cacheDir, "history"),
		StateDir:   filepath.Join(cacheDir, "state"),
		MediaDir:   filepath.Join(cacheDir, "media"),

		Rooms: make(map[string]*rooms.Room),
	}
}

// Clear clears the session cache and removes all history.
func (config *Config) Clear() {
	os.RemoveAll(config.HistoryDir)
	os.RemoveAll(config.StateDir)
	os.RemoveAll(config.MediaDir)
	os.RemoveAll(config.CacheDir)
	config.nosave = true
}

func (config *Config) CreateCacheDirs() {
	os.MkdirAll(config.CacheDir, 0700)
	os.MkdirAll(config.HistoryDir, 0700)
	os.MkdirAll(config.StateDir, 0700)
	os.MkdirAll(config.MediaDir, 0700)
}

func (config *Config) DeleteSession() {
	config.AuthCache.NextBatch = ""
	config.AuthCache.InitialSyncDone = false
	config.Rooms = make(map[string]*rooms.Room)
	config.PushRules = nil

	config.Clear()
	config.nosave = false
	config.CreateCacheDirs()
}

func (config *Config) LoadAll() {
	config.Load()
	config.LoadAuthCache()
	config.LoadPushRules()
	config.LoadRooms()
}

// Load loads the config from config.yaml in the directory given to the config struct.
func (config *Config) Load() {
	os.MkdirAll(config.Dir, 0700)

	configPath := filepath.Join(config.Dir, "config.yaml")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config.CreateCacheDirs()
			return
		}
		debug.Print("Failed to read config from", configPath)
		panic(err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		debug.Print("Failed to parse config at", configPath)
		panic(err)
	}
	config.CreateCacheDirs()
}

func (config *Config) SaveAll() {
	config.Save()
	config.SaveAuthCache()
	config.SavePushRules()
	config.SaveRooms()
}

// Save saves this config to config.yaml in the directory given to the config struct.
func (config *Config) Save() {
	if config.nosave {
		return
	}

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

func (config *Config) LoadAuthCache() {
	os.MkdirAll(config.Dir, 0700)

	configPath := filepath.Join(config.CacheDir, "auth-cache.yaml")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		debug.Print("Failed to read auth cache from", configPath)
		panic(err)
	}

	err = yaml.Unmarshal(data, &config.AuthCache)
	if err != nil {
		debug.Print("Failed to parse auth cache at", configPath)
		panic(err)
	}
}

func (config *Config) SaveAuthCache() {
	if config.nosave {
		return
	}

	os.MkdirAll(config.CacheDir, 0700)
	data, err := yaml.Marshal(&config.AuthCache)
	if err != nil {
		debug.Print("Failed to marshal auth cache")
		panic(err)
	}

	path := filepath.Join(config.CacheDir, "auth-cache.yaml")
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write auth cache to", path)
		panic(err)
	}
}

func (config *Config) LoadPushRules() {
	os.MkdirAll(config.CacheDir, 0700)

	pushRulesPath := filepath.Join(config.CacheDir, "pushrules.json")
	data, err := ioutil.ReadFile(pushRulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		debug.Print("Failed to read push rules from", pushRulesPath)
		return
	}

	config.PushRules = &pushrules.PushRuleset{}
	err = json.Unmarshal(data, &config.PushRules)
	if err != nil {
		debug.Print("Failed to parse push rules at", pushRulesPath)
		return
	}
}

func (config *Config) SavePushRules() {
	if config.nosave || config.PushRules == nil {
		return
	}

	os.MkdirAll(config.CacheDir, 0700)
	data, err := json.Marshal(&config.PushRules)
	if err != nil {
		debug.Print("Failed to marshal push rules")
		return
	}

	path := filepath.Join(config.CacheDir, "pushrules.json")
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write config to", path)
		return
	}
}

func (config *Config) LoadRooms() {
	os.MkdirAll(config.StateDir, 0700)

	roomFiles, err := ioutil.ReadDir(config.StateDir)
	if err != nil {
		debug.Print("Failed to list rooms state caches in", config.StateDir)
		panic(err)
	}

	for _, roomFile := range roomFiles {
		if roomFile.IsDir() || !strings.HasSuffix(roomFile.Name(), ".gmxstate") {
			continue
		}
		path := filepath.Join(config.StateDir, roomFile.Name())
		room := &rooms.Room{}
		err = room.Load(path)
		if err != nil {
			debug.Printf("Failed to load room state cache from %s: %v", path, err)
			continue
		}
		config.Rooms[room.ID] = room
	}
}

func (config *Config) SaveRooms() {
	if config.nosave {
		return
	}

	os.MkdirAll(config.StateDir, 0700)
	for _, room := range config.Rooms {
		path := config.getRoomCachePath(room)
		err := room.Save(path)
		if err != nil {
			debug.Printf("Failed to save room state cache to file %s: %v", path, err)
		}
	}
}

func (config *Config) GetUserID() string {
	return config.UserID
}

func (config *Config) SaveFilterID(_, filterID string) {
	config.AuthCache.FilterID = filterID
	config.SaveAuthCache()
}

func (config *Config) LoadFilterID(_ string) string {
	return config.AuthCache.FilterID
}

func (config *Config) SaveNextBatch(_, nextBatch string) {
	config.AuthCache.NextBatch = nextBatch
	config.SaveAuthCache()
}

func (config *Config) LoadNextBatch(_ string) string {
	return config.AuthCache.NextBatch
}

func (config *Config) GetRoom(roomID string) *rooms.Room {
	room, _ := config.Rooms[roomID]
	if room == nil {
		room = rooms.NewRoom(roomID, config.UserID)
		config.Rooms[room.ID] = room
	}
	return room
}

func (config *Config) getRoomCachePath(room *rooms.Room) string {
	return filepath.Join(config.StateDir, room.ID+".gmxstate")
}

func (config *Config) PutRoom(room *rooms.Room) {
	config.Rooms[room.ID] = room
	room.Save(config.getRoomCachePath(room))
}

func (config *Config) SaveRoom(room *gomatrix.Room) {
	gmxRoom := config.GetRoom(room.ID)
	gmxRoom.Room = room
	gmxRoom.Save(config.getRoomCachePath(gmxRoom))
}

func (config *Config) LoadRoom(roomID string) *gomatrix.Room {
	return config.GetRoom(roomID).Room
}
