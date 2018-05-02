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

package config_test

import (
	"testing"
	"maunium.net/go/gomuks/config"
	"github.com/stretchr/testify/assert"
	"os"
)

func TestConfig_NewSession(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-7")

	cfg := config.NewConfig("/tmp/gomuks-test-7", "/tmp/gomuks-test-7")
	cfg.Load()
	session := cfg.NewSession("@tulir:maunium.net")
	assert.Equal(t, session.GetUserID(), "@tulir:maunium.net")
	assert.Empty(t, session.Rooms)

	_, err1 := os.Stat("/tmp/gomuks-test-7/@tulir:maunium.net.session")
	assert.True(t, os.IsNotExist(err1))
	assert.Nil(t, session.Save())
	_, err2 := os.Stat("/tmp/gomuks-test-7/@tulir:maunium.net.session")
	assert.Nil(t, err2)
}

func TestSession_Load(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-8")

	cfg := config.NewConfig("/tmp/gomuks-test-8", "/tmp/gomuks-test-8")
	cfg.Load()
	session := cfg.NewSession("@tulir:maunium.net")
	session.SaveNextBatch("@tulir:maunium.net", "foobar")
	session.SaveFilterID("@tulir:maunium.net", "1234")

	cfg = config.NewConfig("/tmp/gomuks-test-8", "/tmp/gomuks-test-8")
	cfg.LoadSession("@tulir:maunium.net")
	assert.NotNil(t, cfg.Session)
	assert.Equal(t, "foobar", cfg.Session.LoadNextBatch("@tulir:maunium.net"))
	assert.Equal(t, "1234", cfg.Session.LoadFilterID("@tulir:maunium.net"))
}

func TestSession_Clear(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-9")

	cfg := config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.Load()
	session := cfg.NewSession("@tulir:maunium.net")
	session.SaveNextBatch("@tulir:maunium.net", "foobar")
	session.SaveFilterID("@tulir:maunium.net", "1234")

	cfg = config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.LoadSession("@tulir:maunium.net")
	assert.NotNil(t, cfg.Session)
	assert.Equal(t, "foobar", cfg.Session.LoadNextBatch("@tulir:maunium.net"))
	assert.Equal(t, "1234", cfg.Session.LoadFilterID("@tulir:maunium.net"))

	cfg.Session.Clear()
	assert.Empty(t, cfg.Session.FilterID)
	assert.Empty(t, cfg.Session.NextBatch)
	assert.Empty(t, cfg.Session.Rooms)

	cfg = config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.LoadSession("@tulir:maunium.net")
	assert.Empty(t, cfg.Session.FilterID)
	assert.Empty(t, cfg.Session.NextBatch)
	assert.Empty(t, cfg.Session.Rooms)
}

func TestConfig_ClearWithSession(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-9")

	cfg := config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.Load()
	session := cfg.NewSession("@tulir:maunium.net")
	session.SaveNextBatch("@tulir:maunium.net", "foobar")
	session.SaveFilterID("@tulir:maunium.net", "1234")

	cfg = config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.LoadSession("@tulir:maunium.net")
	assert.NotNil(t, cfg.Session)
	assert.Equal(t, "foobar", cfg.Session.LoadNextBatch("@tulir:maunium.net"))
	assert.Equal(t, "1234", cfg.Session.LoadFilterID("@tulir:maunium.net"))

	cfg.Clear()
	assert.Empty(t, cfg.Session.FilterID)
	assert.Empty(t, cfg.Session.NextBatch)
	assert.Empty(t, cfg.Session.Rooms)

	cfg = config.NewConfig("/tmp/gomuks-test-9", "/tmp/gomuks-test-9")
	cfg.LoadSession("@tulir:maunium.net")
	assert.Empty(t, cfg.Session.FilterID)
	assert.Empty(t, cfg.Session.NextBatch)
	assert.Empty(t, cfg.Session.Rooms)
}

func TestSession_GetRoom(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-10")

	cfg := config.NewConfig("/tmp/gomuks-test-10", "/tmp/gomuks-test-10")
	cfg.Session = cfg.NewSession("@tulir:maunium.net")
	room := cfg.Session.GetRoom("!foo:maunium.net")
	assert.NotNil(t, room)
	assert.Equal(t, room.Room, cfg.Session.LoadRoom("!foo:maunium.net"))
}

func TestSession_PutRoom(t *testing.T) {
	defer os.RemoveAll("/tmp/gomuks-test-11")

	cfg := config.NewConfig("/tmp/gomuks-test-11", "/tmp/gomuks-test-11")
	cfg.Load()
	cfg.LoadSession("@tulir:maunium.net")
	room := cfg.Session.GetRoom("!foo:maunium.net")
	room.PrevBatch = "foobar"
	room.HasLeft = true
	cfg.Session.PutRoom(room)

	cfg = config.NewConfig("/tmp/gomuks-test-11", "/tmp/gomuks-test-11")
	cfg.LoadSession("@tulir:maunium.net")
	reloadedRoom := cfg.Session.GetRoom("!foo:maunium.net")
	assert.Equal(t, "foobar", reloadedRoom.PrevBatch, "%v %v", room, reloadedRoom)
	assert.True(t, reloadedRoom.HasLeft, "%v %v", room, reloadedRoom)
}
