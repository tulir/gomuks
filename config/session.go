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
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/debug"
)

type Session struct {
	UserID      string `json:"-"`
	path        string
	AccessToken string
	NextBatch   string
	FilterID    string
	Rooms       map[string]*rooms.Room
	PushRules   *pushrules.PushRuleset
}

func (config *Config) LoadSession(mxid string) error {
	config.Session = config.NewSession(mxid)
	return config.Session.Load()
}

func (config *Config) NewSession(mxid string) *Session {
	return &Session{
		UserID: mxid,
		path:   filepath.Join(config.Dir, mxid+".session"),
		Rooms:  make(map[string]*rooms.Room),
	}
}

func (s *Session) Clear() {
	s.Rooms = make(map[string]*rooms.Room)
	s.PushRules = nil
	s.NextBatch = ""
	s.FilterID = ""
	s.Save()
}

func (s *Session) Load() error {
	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		debug.Printf("Failed to read session from %s: %v", s.path, err)
		return err
	}

	err = json.Unmarshal(data, s)
	if err != nil {
		debug.Printf("Failed to parse session at %s: %v", s.path, err)
		return err
	}
	return nil
}

func (s *Session) Save() error {
	data, err := json.Marshal(s)
	if err != nil {
		debug.Printf("Failed to marshal session of %s: %v", s.UserID, err)
		return err
	}

	err = ioutil.WriteFile(s.path, data, 0600)
	if err != nil {
		debug.Printf("Failed to write session of %s to %s: %v", s.UserID, s.path, err)
		return err
	}
	return nil
}

func (s *Session) LoadFilterID(_ string) string {
	return s.FilterID
}

func (s *Session) LoadNextBatch(_ string) string {
	return s.NextBatch
}

func (s *Session) GetRoom(mxid string) *rooms.Room {
	room, _ := s.Rooms[mxid]
	if room == nil {
		room = rooms.NewRoom(mxid, s.UserID)
		s.Rooms[room.ID] = room
	}
	return room
}

func (s *Session) PutRoom(room *rooms.Room) {
	s.Rooms[room.ID] = room
	s.Save()
}

func (s *Session) SaveFilterID(_, filterID string) {
	s.FilterID = filterID
	s.Save()
}

func (s *Session) SaveNextBatch(_, nextBatch string) {
	s.NextBatch = nextBatch
	s.Save()
}

func (s *Session) LoadRoom(mxid string) *gomatrix.Room {
	return s.GetRoom(mxid).Room
}

func (s *Session) SaveRoom(room *gomatrix.Room) {
	s.GetRoom(room.ID).Room = room
	s.Save()
}
