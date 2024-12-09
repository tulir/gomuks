// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

package gomuks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chzyer/readline"
	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"go.mau.fi/zeroconfig"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Web     WebConfig         `yaml:"web"`
	Matrix  MatrixConfig      `yaml:"matrix"`
	Push    PushConfig        `yaml:"push"`
	Media   MediaConfig       `yaml:"media"`
	Logging zeroconfig.Config `yaml:"logging"`
}

type MatrixConfig struct {
	DisableHTTP2 bool `yaml:"disable_http2"`
}

type PushConfig struct {
	FCMGateway string `yaml:"fcm_gateway"`
}

type MediaConfig struct {
	ThumbnailSize int `yaml:"thumbnail_size"`
}

type WebConfig struct {
	ListenAddress   string   `yaml:"listen_address"`
	Username        string   `yaml:"username"`
	PasswordHash    string   `yaml:"password_hash"`
	TokenKey        string   `yaml:"token_key"`
	DebugEndpoints  bool     `yaml:"debug_endpoints"`
	EventBufferSize int      `yaml:"event_buffer_size"`
	OriginPatterns  []string `yaml:"origin_patterns"`
}

var defaultFileWriter = zeroconfig.WriterConfig{
	Type:   zeroconfig.WriterTypeFile,
	Format: "json",
	FileConfig: zeroconfig.FileConfig{
		Filename:   "",
		MaxSize:    100,
		MaxBackups: 10,
	},
}

func makeDefaultConfig() Config {
	return Config{
		Web: WebConfig{
			ListenAddress: "localhost:29325",
		},
		Matrix: MatrixConfig{
			DisableHTTP2: false,
		},
		Media: MediaConfig{
			ThumbnailSize: 120,
		},
		Logging: zeroconfig.Config{
			MinLevel: ptr.Ptr(zerolog.DebugLevel),
			Writers: []zeroconfig.WriterConfig{{
				Type:   zeroconfig.WriterTypeStdout,
				Format: zeroconfig.LogFormatPrettyColored,
			}, defaultFileWriter},
		},
	}
}

func (gmx *Gomuks) LoadConfig() error {
	file, err := os.Open(filepath.Join(gmx.ConfigDir, "config.yaml"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	gmx.Config = makeDefaultConfig()
	changed := false
	if file != nil {
		err = yaml.NewDecoder(file).Decode(&gmx.Config)
		if err != nil {
			return err
		}
	} else {
		changed = true
	}
	if gmx.Config.Web.TokenKey == "" {
		gmx.Config.Web.TokenKey = random.String(64)
		changed = true
	}
	if !gmx.DisableAuth && gmx.TUI == nil && (gmx.Config.Web.Username == "" || gmx.Config.Web.PasswordHash == "") {
		fmt.Println("Please create a username and password for authenticating the web app")
		fmt.Println("This is only used for gomuks and is NOT your Matrix account")
		gmx.Config.Web.Username, err = readline.Line("Username: ")
		if err != nil {
			return fmt.Errorf("failed to read username: %w", err)
		} else if len(gmx.Config.Web.Username) == 0 || len(gmx.Config.Web.Username) > 32 {
			return fmt.Errorf("username must be 1-32 characters long")
		}
		passwd, err := readline.Password("Password: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		hash, err := bcrypt.GenerateFromPassword(passwd, 12)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		gmx.Config.Web.PasswordHash = string(hash)
		changed = true
	}
	if gmx.Config.Web.EventBufferSize <= 0 {
		gmx.Config.Web.EventBufferSize = 512
		changed = true
	}
	if gmx.Config.Push.FCMGateway == "" {
		gmx.Config.Push.FCMGateway = "https://push.gomuks.app"
		changed = true
	}
	if gmx.Config.Media.ThumbnailSize == 0 {
		gmx.Config.Media.ThumbnailSize = 120
		changed = true
	}
	if len(gmx.Config.Web.OriginPatterns) == 0 {
		gmx.Config.Web.OriginPatterns = []string{"localhost:*", "*.localhost:*"}
		changed = true
	}
	if changed {
		err = gmx.SaveConfig()
		if err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}
	gmx.EventBuffer = NewEventBuffer(gmx.Config.Web.EventBufferSize)
	return nil
}

func (gmx *Gomuks) SaveConfig() error {
	file, err := os.OpenFile(filepath.Join(gmx.ConfigDir, "config.yaml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	return yaml.NewEncoder(file).Encode(&gmx.Config)
}
