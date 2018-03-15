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

package main

import (
	"fmt"
	"time"

	"maunium.net/go/gomatrix"
)

type MatrixContainer struct {
	client  *gomatrix.Client
	gmx     Gomuks
	ui      *GomuksUI
	debug   DebugPrinter
	config  *Config
	running bool
	stop    chan bool

	typing int64
}

func NewMatrixContainer(gmx Gomuks) *MatrixContainer {
	c := &MatrixContainer{
		config: gmx.Config(),
		debug:  gmx.Debug(),
		ui:     gmx.UI(),
		gmx:    gmx,
	}

	return c
}

func (c *MatrixContainer) InitClient() error {
	if len(c.config.HS) == 0 {
		return fmt.Errorf("no homeserver in config")
	}

	if c.client != nil {
		c.Stop()
		c.client = nil
	}

	var mxid, accessToken string
	if c.config.Session != nil {
		accessToken = c.config.Session.AccessToken
		mxid = c.config.MXID
	}

	var err error
	c.client, err = gomatrix.NewClient(c.config.HS, mxid, accessToken)
	if err != nil {
		return err
	}

	c.stop = make(chan bool, 1)

	if c.config.Session != nil {
		go c.Start()
	}
	return nil
}

func (c *MatrixContainer) Initialized() bool {
	return c.client != nil
}

func (c *MatrixContainer) Login(user, password string) error {
	resp, err := c.client.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     user,
		Password: password,
	})
	if err != nil {
		return err
	}
	c.client.SetCredentials(resp.UserID, resp.AccessToken)
	c.config.MXID = resp.UserID
	c.config.Save()

	c.config.Session = c.config.NewSession(resp.UserID)
	c.config.Session.AccessToken = resp.AccessToken
	c.config.Session.Save()

	go c.Start()

	return nil
}

func (c *MatrixContainer) Stop() {
	if c.running {
		c.stop <- true
		c.client.StopSync()
	}
}

func (c *MatrixContainer) UpdateRoomList() {
	rooms, err := c.client.JoinedRooms()
	if err != nil {
		c.debug.Print(err)
	}

	c.ui.MainView().SetRoomList(rooms.JoinedRooms)
}

func (c *MatrixContainer) Start() {
	defer c.gmx.Recover()
	c.debug.Print("Starting sync...")
	c.running = true
	c.ui.SetView(ViewMain)
	c.client.Store = c.config.Session

	c.UpdateRoomList()

	syncer := c.client.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", c.HandleMessage)
	syncer.OnEventType("m.typing", c.HandleTyping)

	for {
		select {
		case <-c.stop:
			c.debug.Print("Stopping sync...")
			c.running = false
			return
		default:
			if err := c.client.Sync(); err != nil {
				c.debug.Print("Sync() errored", err)
			} else {
				c.debug.Print("Sync() returned without error")
			}
		}
	}
}

func (c *MatrixContainer) HandleMessage(evt *gomatrix.Event) {
	message, _ := evt.Content["body"].(string)
	c.ui.MainView().Append(evt.RoomID, evt.Sender, message)
}

func (c *MatrixContainer) HandleTyping(evt *gomatrix.Event) {
	users := evt.Content["user_ids"].([]interface{})
	c.debug.Print(users, "are typing")

	strUsers := make([]string, len(users))
	for i, user := range users {
		strUsers[i] = user.(string)
	}
	c.ui.MainView().SetTyping(evt.RoomID, strUsers)
}

func (c *MatrixContainer) SendMessage(roomID, message string) {
	c.SendTyping(roomID, false)
	c.client.SendText(roomID, message)
}

func (c *MatrixContainer) SendTyping(roomID string, typing bool) {
	time := time.Now().Unix()
	if c.typing > time && typing {
		return
	}

	if typing {
		c.client.UserTyping(roomID, true, 5000)
		c.typing = time + 5
	} else {
		c.client.UserTyping(roomID, false, 0)
		c.typing = 0
	}
}

func (c *MatrixContainer) GetState(roomID string) []*gomatrix.Event {
	content := make([]*gomatrix.Event, 0)
	err := c.client.StateEvent(roomID, "", "", &content)
	if err != nil {
		c.debug.Print(err)
		return nil
	}
	return content
}

func (c *MatrixContainer) UpdateState(roomID string) {
	room := c.client.Store.LoadRoom(roomID)
	if room == nil {
		return
	}
	events := c.GetState(room.ID)
	if events != nil {
		for _, event := range events {
			room.UpdateState(event)
		}
	}
}
