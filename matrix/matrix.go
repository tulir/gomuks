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

package matrix

import (
	"fmt"
	"strings"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	rooms "maunium.net/go/gomuks/matrix/room"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/widget"
)

type Container struct {
	client  *gomatrix.Client
	gmx     ifc.Gomuks
	ui      ifc.GomuksUI
	config  *config.Config
	running bool
	stop    chan bool

	typing int64
}

func NewMatrixContainer(gmx ifc.Gomuks) *Container {
	c := &Container{
		config: gmx.Config(),
		ui:     gmx.UI(),
		gmx:    gmx,
	}

	return c
}

func (c *Container) InitClient() error {
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

func (c *Container) Initialized() bool {
	return c.client != nil
}

func (c *Container) Login(user, password string) error {
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

func (c *Container) Stop() {
	if c.running {
		c.stop <- true
		c.client.StopSync()
	}
}

func (c *Container) Client() *gomatrix.Client {
	return c.client
}

func (c *Container) UpdateRoomList() {
	resp, err := c.client.JoinedRooms()
	if err != nil {
		debug.Print("Error fetching room list:", err)
		return
	}

	c.ui.MainView().SetRooms(resp.JoinedRooms)
}

func (c *Container) OnLogin() {
	c.client.Store = c.config.Session

	syncer := NewGomuksSyncer(c.config.Session)
	syncer.OnEventType("m.room.message", c.HandleMessage)
	syncer.OnEventType("m.room.member", c.HandleMembership)
	syncer.OnEventType("m.typing", c.HandleTyping)
	c.client.Syncer = syncer

	c.UpdateRoomList()
}

func (c *Container) Start() {
	defer c.gmx.Recover()

	c.ui.SetView(ifc.ViewMain)
	c.OnLogin()

	debug.Print("Starting sync...")
	c.running = true
	for {
		select {
		case <-c.stop:
			debug.Print("Stopping sync...")
			c.running = false
			return
		default:
			if err := c.client.Sync(); err != nil {
				debug.Print("Sync() errored", err)
			} else {
				debug.Print("Sync() returned without error")
			}
		}
	}
}

func (c *Container) HandleMessage(evt *gomatrix.Event) {
	room, message := c.ui.MainView().ProcessMessageEvent(evt)
	if room != nil {
		room.AddMessage(message, widget.AppendMessage)
	}
}

func (c *Container) HandleMembership(evt *gomatrix.Event) {
	const Hour = 1 * 60 * 60 * 1000
	if evt.Unsigned.Age > Hour {
		return
	}

	room, message := c.ui.MainView().ProcessMembershipEvent(evt, true)
	if room != nil {
		// TODO this shouldn't be necessary
		room.Room.UpdateState(evt)
		// TODO This should probably also be in a different place
		room.UpdateUserList()

		room.AddMessage(message, widget.AppendMessage)
	}
}

func (c *Container) HandleTyping(evt *gomatrix.Event) {
	users := evt.Content["user_ids"].([]interface{})

	strUsers := make([]string, len(users))
	for i, user := range users {
		strUsers[i] = user.(string)
	}
	c.ui.MainView().SetTyping(evt.RoomID, strUsers)
}

func (c *Container) SendMessage(roomID, message string) {
	c.gmx.Recover()
	c.SendTyping(roomID, false)
	c.client.SendText(roomID, message)
}

func (c *Container) SendTyping(roomID string, typing bool) {
	c.gmx.Recover()
	ts := time.Now().Unix()
	if c.typing > ts && typing {
		return
	}

	if typing {
		c.client.UserTyping(roomID, true, 5000)
		c.typing = ts + 5
	} else {
		c.client.UserTyping(roomID, false, 0)
		c.typing = 0
	}
}

func (c *Container) JoinRoom(roomID string) error {
	if len(roomID) == 0 {
		return fmt.Errorf("invalid room ID")
	}

	server := ""
	if roomID[0] == '!' {
		server = roomID[strings.Index(roomID, ":")+1:]
	}

	_, err := c.client.JoinRoom(roomID, server, nil)
	if err != nil {
		return err
	}

	// TODO probably safe to remove
	// c.ui.MainView().AddRoom(resp.RoomID)
	return nil
}

func (c *Container) LeaveRoom(roomID string) error {
	if len(roomID) == 0 {
		return fmt.Errorf("invalid room ID")
	}

	_, err := c.client.LeaveRoom(roomID)
	if err != nil {
		return err
	}

	return nil
}

func (c *Container) getState(roomID string) []*gomatrix.Event {
	content := make([]*gomatrix.Event, 0)
	err := c.client.StateEvent(roomID, "", "", &content)
	if err != nil {
		debug.Print("Error getting state of", roomID, err)
		return nil
	}
	return content
}

func (c *Container) GetHistory(roomID, prevBatch string, limit int) ([]gomatrix.Event, string, error) {
	resp, err := c.client.Messages(roomID, prevBatch, "", 'b', limit)
	if err != nil {
		return nil, "", err
	}
	return resp.Chunk, resp.End, nil
}

func (c *Container) GetRoom(roomID string) *rooms.Room {
	room := c.config.Session.GetRoom(roomID)
	if room != nil && len(room.State) == 0 {
		events := c.getState(room.ID)
		if events != nil {
			for _, event := range events {
				room.UpdateState(event)
			}
		}
	}
	return room
}
