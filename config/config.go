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

package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/cbind"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
)

type AuthCache struct {
	NextBatch       string `yaml:"next_batch"`
	FilterID        string `yaml:"filter_id"`
	FilterVersion   int    `yaml:"filter_version"`
	InitialSyncDone bool   `yaml:"initial_sync_done"`
}

type UserPreferences struct {
	HideUserList         bool `yaml:"hide_user_list"`
	HideRoomList         bool `yaml:"hide_room_list"`
	HideTimestamp        bool `yaml:"hide_timestamp"`
	BareMessageView      bool `yaml:"bare_message_view"`
	DisableImages        bool `yaml:"disable_images"`
	DisableTypingNotifs  bool `yaml:"disable_typing_notifs"`
	DisableEmojis        bool `yaml:"disable_emojis"`
	DisableMarkdown      bool `yaml:"disable_markdown"`
	DisableHTML          bool `yaml:"disable_html"`
	DisableDownloads     bool `yaml:"disable_downloads"`
	DisableNotifications bool `yaml:"disable_notifications"`
	DisableShowURLs      bool `yaml:"disable_show_urls"`

	InlineURLMode string `yaml:"inline_url_mode"`
}

var InlineURLsProbablySupported bool

func init() {
	vteVersion, _ := strconv.Atoi(os.Getenv("VTE_VERSION"))
	term := os.Getenv("TERM")
	// Enable inline URLs by default on VTE 0.50.0+
	InlineURLsProbablySupported = vteVersion > 5000 ||
		os.Getenv("TERM_PROGRAM") == "iTerm.app" ||
		term == "foot" ||
		term == "xterm-kitty"
}

func (up *UserPreferences) EnableInlineURLs() bool {
	return up.InlineURLMode == "enable" || (InlineURLsProbablySupported && up.InlineURLMode != "disable")
}

type Keybind struct {
	Mod tcell.ModMask
	Key tcell.Key
	Ch  rune
}

type ParsedKeybindings struct {
	Main   map[Keybind]string
	Room   map[Keybind]string
	Modal  map[Keybind]string
	Visual map[Keybind]string
}

type RawKeybindings struct {
	Main   map[string]string `yaml:"main,omitempty"`
	Room   map[string]string `yaml:"room,omitempty"`
	Modal  map[string]string `yaml:"modal,omitempty"`
	Visual map[string]string `yaml:"visual,omitempty"`
}

// Config contains the main config of gomuks.
type Config struct {
	UserID      id.UserID   `yaml:"mxid"`
	DeviceID    id.DeviceID `yaml:"device_id"`
	AccessToken string      `yaml:"access_token"`
	HS          string      `yaml:"homeserver"`

	RoomCacheSize int   `yaml:"room_cache_size"`
	RoomCacheAge  int64 `yaml:"room_cache_age"`

	NotifySound        bool `yaml:"notify_sound"`
	SendToVerifiedOnly bool `yaml:"send_to_verified_only"`

	Backspace1RemovesWord bool `yaml:"backspace1_removes_word"`
	Backspace2RemovesWord bool `yaml:"backspace2_removes_word"`

	AlwaysClearScreen bool `yaml:"always_clear_screen"`

	Dir          string `yaml:"-"`
	DataDir      string `yaml:"data_dir"`
	CacheDir     string `yaml:"cache_dir"`
	HistoryPath  string `yaml:"history_path"`
	RoomListPath string `yaml:"room_list_path"`
	MediaDir     string `yaml:"media_dir"`
	DownloadDir  string `yaml:"download_dir"`
	StateDir     string `yaml:"state_dir"`

	Preferences UserPreferences        `yaml:"-"`
	AuthCache   AuthCache              `yaml:"-"`
	Rooms       *rooms.RoomCache       `yaml:"-"`
	PushRules   *pushrules.PushRuleset `yaml:"-"`
	Keybindings ParsedKeybindings      `yaml:"-"`

	nosave bool
}

// NewConfig creates a config that loads data from the given directory.
func NewConfig(configDir, dataDir, cacheDir, downloadDir string) *Config {
	return &Config{
		Dir:          configDir,
		DataDir:      dataDir,
		CacheDir:     cacheDir,
		DownloadDir:  downloadDir,
		HistoryPath:  filepath.Join(cacheDir, "history.db"),
		RoomListPath: filepath.Join(cacheDir, "rooms.gob.gz"),
		StateDir:     filepath.Join(cacheDir, "state"),
		MediaDir:     filepath.Join(cacheDir, "media"),

		RoomCacheSize: 32,
		RoomCacheAge:  1 * 60,

		NotifySound:           true,
		SendToVerifiedOnly:    false,
		Backspace1RemovesWord: true,
		AlwaysClearScreen:     true,
	}
}

// Clear clears the session cache and removes all history.
func (config *Config) Clear() {
	_ = os.Remove(config.HistoryPath)
	_ = os.Remove(config.RoomListPath)
	_ = os.RemoveAll(config.StateDir)
	_ = os.RemoveAll(config.MediaDir)
	_ = os.RemoveAll(config.CacheDir)
	config.nosave = true
}

// ClearData clears non-temporary session data.
func (config *Config) ClearData() {
	_ = os.RemoveAll(config.DataDir)
}

func (config *Config) CreateCacheDirs() {
	_ = os.MkdirAll(config.CacheDir, 0700)
	_ = os.MkdirAll(config.DataDir, 0700)
	_ = os.MkdirAll(config.StateDir, 0700)
	_ = os.MkdirAll(config.MediaDir, 0700)
}

func (config *Config) DeleteSession() {
	config.AuthCache.NextBatch = ""
	config.AuthCache.InitialSyncDone = false
	config.AccessToken = ""
	config.DeviceID = ""
	config.Rooms = rooms.NewRoomCache(config.RoomListPath, config.StateDir, config.RoomCacheSize, config.RoomCacheAge, config.GetUserID)
	config.PushRules = nil

	config.ClearData()
	config.Clear()
	config.nosave = false
	config.CreateCacheDirs()
}

func (config *Config) LoadAll() {
	config.Load()
	config.Rooms = rooms.NewRoomCache(config.RoomListPath, config.StateDir, config.RoomCacheSize, config.RoomCacheAge, config.GetUserID)
	config.LoadAuthCache()
	config.LoadPushRules()
	config.LoadPreferences()
	config.LoadKeybindings()
	err := config.Rooms.LoadList()
	if err != nil {
		panic(err)
	}
}

// Load loads the config from config.yaml in the directory given to the config struct.
func (config *Config) Load() {
	err := config.load("config", config.Dir, "config.yaml", config)
	if err != nil {
		panic(fmt.Errorf("failed to load config.yaml: %w", err))
	}
	config.CreateCacheDirs()
}

func (config *Config) SaveAll() {
	config.Save()
	config.SaveAuthCache()
	config.SavePushRules()
	config.SavePreferences()
	err := config.Rooms.SaveList()
	if err != nil {
		panic(err)
	}
	config.Rooms.SaveLoadedRooms()
}

// Save saves this config to config.yaml in the directory given to the config struct.
func (config *Config) Save() {
	config.save("config", config.Dir, "config.yaml", config)
}

func (config *Config) LoadPreferences() {
	_ = config.load("user preferences", config.CacheDir, "preferences.yaml", &config.Preferences)
}

func (config *Config) SavePreferences() {
	config.save("user preferences", config.CacheDir, "preferences.yaml", &config.Preferences)
}

//go:embed keybindings.yaml
var DefaultKeybindings string

func parseKeybindings(input map[string]string) (output map[Keybind]string) {
	output = make(map[Keybind]string, len(input))
	for shortcut, action := range input {
		mod, key, ch, err := cbind.Decode(shortcut)
		if err != nil {
			panic(fmt.Errorf("failed to parse keybinding %s -> %s: %w", shortcut, action, err))
		}
		// TODO find out if other keys are parsed incorrectly like this
		if key == tcell.KeyEscape {
			ch = 0
		}
		parsedShortcut := Keybind{
			Mod: mod,
			Key: key,
			Ch:  ch,
		}
		output[parsedShortcut] = action
	}
	return
}

func (config *Config) LoadKeybindings() {
	var inputConfig RawKeybindings

	err := yaml.Unmarshal([]byte(DefaultKeybindings), &inputConfig)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal default keybindings: %w", err))
	}
	_ = config.load("keybindings", config.Dir, "keybindings.yaml", &inputConfig)

	config.Keybindings.Main = parseKeybindings(inputConfig.Main)
	config.Keybindings.Room = parseKeybindings(inputConfig.Room)
	config.Keybindings.Modal = parseKeybindings(inputConfig.Modal)
	config.Keybindings.Visual = parseKeybindings(inputConfig.Visual)
}

func (config *Config) SaveKeybindings() {
	config.save("keybindings", config.Dir, "keybindings.yaml", &config.Keybindings)
}

func (config *Config) LoadAuthCache() {
	err := config.load("auth cache", config.CacheDir, "auth-cache.yaml", &config.AuthCache)
	if err != nil {
		panic(fmt.Errorf("failed to load auth-cache.yaml: %w", err))
	}
}

func (config *Config) SaveAuthCache() {
	config.save("auth cache", config.CacheDir, "auth-cache.yaml", &config.AuthCache)
}

func (config *Config) LoadPushRules() {
	_ = config.load("push rules", config.CacheDir, "pushrules.json", &config.PushRules)

}

func (config *Config) SavePushRules() {
	if config.PushRules == nil {
		return
	}
	config.save("push rules", config.CacheDir, "pushrules.json", &config.PushRules)
}

func (config *Config) load(name, dir, file string, target interface{}) error {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		debug.Print("Failed to create", dir)
		return err
	}

	path := filepath.Join(dir, file)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		debug.Print("Failed to read", name, "from", path)
		return err
	}

	if strings.HasSuffix(file, ".yaml") {
		err = yaml.Unmarshal(data, target)
	} else {
		err = json.Unmarshal(data, target)
	}
	if err != nil {
		debug.Print("Failed to parse", name, "at", path)
		return err
	}
	return nil
}

func (config *Config) save(name, dir, file string, source interface{}) {
	if config.nosave {
		return
	}

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		debug.Print("Failed to create", dir)
		panic(err)
	}
	var data []byte
	if strings.HasSuffix(file, ".yaml") {
		data, err = yaml.Marshal(source)
	} else {
		data, err = json.Marshal(source)
	}
	if err != nil {
		debug.Print("Failed to marshal", name)
		panic(err)
	}

	path := filepath.Join(dir, file)
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write", name, "to", path)
		panic(err)
	}
}

func (config *Config) GetUserID() id.UserID {
	return config.UserID
}

const FilterVersion = 1

func (config *Config) SaveFilterID(_ id.UserID, filterID string) {
	config.AuthCache.FilterID = filterID
	config.AuthCache.FilterVersion = FilterVersion
	config.SaveAuthCache()
}

func (config *Config) LoadFilterID(_ id.UserID) string {
	if config.AuthCache.FilterVersion != FilterVersion {
		return ""
	}
	return config.AuthCache.FilterID
}

func (config *Config) SaveNextBatch(_ id.UserID, nextBatch string) {
	config.AuthCache.NextBatch = nextBatch
	config.SaveAuthCache()
}

func (config *Config) LoadNextBatch(_ id.UserID) string {
	return config.AuthCache.NextBatch
}

func (config *Config) SaveRoom(_ *mautrix.Room) {
	panic("SaveRoom is not supported")
}

func (config *Config) LoadRoom(_ id.RoomID) *mautrix.Room {
	panic("LoadRoom is not supported")
}
