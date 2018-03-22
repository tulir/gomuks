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

	"github.com/gdamore/tcell"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/room"
	"maunium.net/go/gomuks/notification"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/widget"
)

// Container is a wrapper for a gomatrix Client and some other stuff.
//
// It is used for all Matrix calls from the UI and Matrix event handlers.
type Container struct {
	client  *gomatrix.Client
	gmx     ifc.Gomuks
	ui      ifc.GomuksUI
	config  *config.Config
	running bool
	stop    chan bool

	typing int64
}

// NewContainer creates a new Container for the given Gomuks instance.
func NewContainer(gmx ifc.Gomuks) *Container {
	c := &Container{
		config: gmx.Config(),
		ui:     gmx.UI(),
		gmx:    gmx,
	}

	return c
}

// InitClient initializes the gomatrix client and connects to the homeserver specified in the config.
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
		mxid = c.config.UserID
	}

	var err error
	c.client, err = gomatrix.NewClient(c.config.HS, mxid, accessToken)
	if err != nil {
		return err
	}

	c.stop = make(chan bool, 1)

	if c.config.Session != nil && len(accessToken) > 0 {
		go c.Start()
	}
	return nil
}

// Initialized returns whether or not the gomatrix client is initialized (see InitClient())
func (c *Container) Initialized() bool {
	return c.client != nil
}

// Login sends a password login request with the given username and password.
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
	c.config.UserID = resp.UserID
	c.config.Save()

	c.config.Session = c.config.NewSession(resp.UserID)
	c.config.Session.AccessToken = resp.AccessToken
	c.config.Session.Save()

	go c.Start()

	return nil
}

// Stop stops the Matrix syncer.
func (c *Container) Stop() {
	if c.running {
		c.stop <- true
		c.client.StopSync()
	}
}

// Client returns the underlying gomatrix client object.
func (c *Container) Client() *gomatrix.Client {
	return c.client
}

// UpdatePushRules fetches the push notification rules from the server and stores them in the current Session object.
func (c *Container) UpdatePushRules() {
	debug.Print("Updating push rules...")
	resp, err := pushrules.GetPushRules(c.client)
	if err != nil {
		debug.Print("Failed to fetch push rules:", err)
	}
	c.config.Session.PushRules = resp
}

// PushRules returns the push notification rules. If no push rules are cached, UpdatePushRules() will be called first.
func (c *Container) PushRules() *pushrules.PushRuleset {
	if c.config.Session.PushRules == nil {
		c.UpdatePushRules()
	}
	return c.config.Session.PushRules
}

// UpdateRoomList fetches the list of rooms the user has joined and sends them to the UI.
func (c *Container) UpdateRoomList() {
	resp, err := c.client.JoinedRooms()
	if err != nil {
		respErr, _ := err.(gomatrix.HTTPError).WrappedError.(gomatrix.RespError)
		if respErr.ErrCode == "M_UNKNOWN_TOKEN" {
			c.OnLogout()
			return
		}
		debug.Print("Error fetching room list:", err)
		return
	}

	c.ui.MainView().SetRooms(resp.JoinedRooms)
}

// OnLogout stops the syncer and moves the UI back to the login view.
func (c *Container) OnLogout() {
	c.Stop()
	c.ui.SetView(ifc.ViewLogin)
}

// OnLogin initializes the syncer and updates the room list.
func (c *Container) OnLogin() {
	c.client.Store = c.config.Session

	syncer := NewGomuksSyncer(c.config.Session)
	syncer.OnEventType("m.room.message", c.HandleMessage)
	syncer.OnEventType("m.room.member", c.HandleMembership)
	syncer.OnEventType("m.typing", c.HandleTyping)
	syncer.OnEventType("m.push_rules", c.HandlePushRules)
	c.client.Syncer = syncer

	c.UpdateRoomList()
}

// Start moves the UI to the main view, calls OnLogin() and runs the syncer forever until stopped with Stop()
func (c *Container) Start() {
	defer c.gmx.Recover()

	c.ui.SetView(ifc.ViewMain)
	c.OnLogin()

	if c.client == nil {
		return
	}

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

// NotifyMessage sends a desktop notification of the message with the given details.
func (c *Container) NotifyMessage(room *rooms.Room, sender, text string, critical bool) {
	if room.GetTitle() != sender {
		sender = fmt.Sprintf("%s (%s)", sender, room.GetTitle())
	}
	notification.Send(sender, text, critical)
}

// HandleMessage is the event handler for the m.room.message timeline event.
func (c *Container) HandleMessage(evt *gomatrix.Event) {
	room, message := c.ui.MainView().ProcessMessageEvent(evt)
	if room != nil {
		pushRules := c.PushRules().GetActions(room.Room, evt).Should()
		if (pushRules.Notify || !pushRules.NotifySpecified) && evt.Sender != c.config.Session.UserID {
			c.NotifyMessage(room.Room, message.Sender, message.Text, pushRules.Highlight)
		}
		if pushRules.Highlight {
			message.TextColor = tcell.ColorYellow
		}
		if pushRules.PlaySound {
			// TODO play sound
		}
		room.AddMessage(message, widget.AppendMessage)
		c.ui.Render()
	}
}

// HandlePushRules is the event handler for the m.push_rules account data event.
func (c *Container) HandlePushRules(evt *gomatrix.Event) {
	debug.Print("Received updated push rules")
	var err error
	c.config.Session.PushRules, err = pushrules.EventToPushRules(evt)
	if err != nil {
		debug.Print("Failed to convert event to push rules:", err)
	}
}

// HandleMembership is the event handler for the m.room.membership state event.
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
		c.ui.Render()
	}
}

// HandleTyping is the event handler for the m.typing event.
func (c *Container) HandleTyping(evt *gomatrix.Event) {
	users := evt.Content["user_ids"].([]interface{})

	strUsers := make([]string, len(users))
	for i, user := range users {
		strUsers[i] = user.(string)
	}
	c.ui.MainView().SetTyping(evt.RoomID, strUsers)
}

// SendMessage sends a message with the given text to the given room.
func (c *Container) SendMessage(roomID, msgtype, text string) (string, error) {
	defer c.gmx.Recover()
	c.SendTyping(roomID, false)
	resp, err := c.client.SendMessageEvent(roomID, "m.room.message",
		gomatrix.TextMessage{MsgType: msgtype, Body: text})
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

// SendTyping sets whether or not the user is typing in the given room.
func (c *Container) SendTyping(roomID string, typing bool) {
	defer c.gmx.Recover()
	ts := time.Now().Unix()
	if c.typing > ts && typing {
		return
	}

	if typing {
		c.client.UserTyping(roomID, true, 20000)
		c.typing = ts + 15
	} else {
		c.client.UserTyping(roomID, false, 0)
		c.typing = 0
	}
}

// JoinRoom makes the current user try to join the given room.
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

	return nil
}

// LeaveRoom makes the current user leave the given room.
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

// getState requests the state of the given room.
func (c *Container) getState(roomID string) []*gomatrix.Event {
	content := make([]*gomatrix.Event, 0)
	err := c.client.StateEvent(roomID, "", "", &content)
	if err != nil {
		debug.Print("Error getting state of", roomID, err)
		return nil
	}
	return content
}

// GetHistory fetches room history.
func (c *Container) GetHistory(roomID, prevBatch string, limit int) ([]gomatrix.Event, string, error) {
	resp, err := c.client.Messages(roomID, prevBatch, "", 'b', limit)
	if err != nil {
		return nil, "", err
	}
	return resp.Chunk, resp.End, nil
}

// GetRoom gets the room instance stored in the session.
//
// If the room doesn't have any state events stored, its state will be updated.
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
