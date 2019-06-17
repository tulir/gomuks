// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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

package matrix

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
	dbg "runtime/debug"

	"maunium.net/go/gomuks/matrix/event"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/format"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
)

// Container is a wrapper for a mautrix Client and some other stuff.
//
// It is used for all Matrix calls from the UI and Matrix event handlers.
type Container struct {
	client  *mautrix.Client
	syncer  *GomuksSyncer
	gmx     ifc.Gomuks
	ui      ifc.GomuksUI
	config  *config.Config
	history *HistoryManager
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

// Client returns the underlying mautrix Client.
func (c *Container) Client() *mautrix.Client {
	return c.client
}

type mxLogger struct{}

func (log mxLogger) Debugfln(message string, args ...interface{}) {
	debug.Printf("[Matrix] "+message, args...)
}

// InitClient initializes the mautrix client and connects to the homeserver specified in the config.
func (c *Container) InitClient() error {
	if len(c.config.HS) == 0 {
		return fmt.Errorf("no homeserver in config")
	}

	if c.client != nil {
		c.Stop()
		c.client = nil
	}

	var mxid, accessToken string
	if len(c.config.AccessToken) > 0 {
		accessToken = c.config.AccessToken
		mxid = c.config.UserID
	}

	var err error
	c.client, err = mautrix.NewClient(c.config.HS, mxid, accessToken)
	if err != nil {
		return err
	}
	c.client.Logger = mxLogger{}

	c.history, err = NewHistoryManager(c.config.HistoryPath)
	if err != nil {
		return err
	}

	allowInsecure := len(os.Getenv("GOMUKS_ALLOW_INSECURE_CONNECTIONS")) > 0
	if allowInsecure {
		c.client.Client = &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		}
	}

	c.stop = make(chan bool, 1)

	if len(accessToken) > 0 {
		go c.Start()
	}
	return nil
}

// Initialized returns whether or not the mautrix client is initialized (see InitClient())
func (c *Container) Initialized() bool {
	return c.client != nil
}

// Login sends a password login request with the given username and password.
func (c *Container) Login(user, password string) error {
	resp, err := c.client.Login(&mautrix.ReqLogin{
		Type:                     "m.login.password",
		User:                     user,
		Password:                 password,
		InitialDeviceDisplayName: "gomuks",
	})
	if err != nil {
		return err
	}
	c.client.SetCredentials(resp.UserID, resp.AccessToken)
	c.config.UserID = resp.UserID
	c.config.AccessToken = resp.AccessToken
	c.config.Save()

	go c.Start()

	return nil
}

// Logout revokes the access token, stops the syncer and calls the OnLogout() method of the UI.
func (c *Container) Logout() {
	c.client.Logout()
	c.config.DeleteSession()
	c.Stop()
	c.client = nil
	c.ui.OnLogout()
}

// Stop stops the Matrix syncer.
func (c *Container) Stop() {
	if c.running {
		debug.Print("Stopping Matrix container...")
		c.stop <- true
		c.client.StopSync()
		debug.Print("Closing history manager...")
		err := c.history.Close()
		if err != nil {
			debug.Print("Error closing history manager:", err)
		}
	}
}

// UpdatePushRules fetches the push notification rules from the server and stores them in the current Session object.
func (c *Container) UpdatePushRules() {
	debug.Print("Updating push rules...")
	resp, err := pushrules.GetPushRules(c.client)
	if err != nil {
		debug.Print("Failed to fetch push rules:", err)
		c.config.PushRules = &pushrules.PushRuleset{}
	} else {
		c.config.PushRules = resp
	}
	c.config.SavePushRules()
}

// PushRules returns the push notification rules. If no push rules are cached, UpdatePushRules() will be called first.
func (c *Container) PushRules() *pushrules.PushRuleset {
	if c.config.PushRules == nil {
		c.UpdatePushRules()
	}
	return c.config.PushRules
}

var AccountDataGomuksPreferences = mautrix.NewEventType("net.maunium.gomuks.preferences")

// OnLogin initializes the syncer and updates the room list.
func (c *Container) OnLogin() {
	c.ui.OnLogin()

	c.client.Store = c.config

	debug.Print("Initializing syncer")
	c.syncer = NewGomuksSyncer(c.config)
	c.syncer.OnEventType(mautrix.EventMessage, c.HandleMessage)
	// Just pass encrypted events as messages, they'll show up with an encryption unsupported message.
	c.syncer.OnEventType(mautrix.EventEncrypted, c.HandleMessage)
	c.syncer.OnEventType(mautrix.EventSticker, c.HandleMessage)
	c.syncer.OnEventType(mautrix.EventRedaction, c.HandleRedaction)
	c.syncer.OnEventType(mautrix.StateAliases, c.HandleMessage)
	c.syncer.OnEventType(mautrix.StateCanonicalAlias, c.HandleMessage)
	c.syncer.OnEventType(mautrix.StateTopic, c.HandleMessage)
	c.syncer.OnEventType(mautrix.StateRoomName, c.HandleMessage)
	c.syncer.OnEventType(mautrix.StateMember, c.HandleMembership)
	c.syncer.OnEventType(mautrix.EphemeralEventReceipt, c.HandleReadReceipt)
	c.syncer.OnEventType(mautrix.EphemeralEventTyping, c.HandleTyping)
	c.syncer.OnEventType(mautrix.AccountDataDirectChats, c.HandleDirectChatInfo)
	c.syncer.OnEventType(mautrix.AccountDataPushRules, c.HandlePushRules)
	c.syncer.OnEventType(mautrix.AccountDataRoomTags, c.HandleTag)
	c.syncer.OnEventType(AccountDataGomuksPreferences, c.HandlePreferences)
	c.syncer.InitDoneCallback = func() {
		debug.Print("Initial sync done")
		c.config.AuthCache.InitialSyncDone = true
		debug.Print("Updating title caches")
		for _, room := range c.config.Rooms.Map {
			room.GetTitle()
		}
		debug.Print("Cleaning cached rooms from memory")
		c.config.Rooms.ForceClean()
		debug.Print("Saving all data")
		c.config.SaveAll()
		debug.Print("Adding rooms to UI")
		c.ui.MainView().SetRooms(c.config.Rooms)
		c.ui.Render()
		// The initial sync can be a bit heavy, so we force run the GC here
		// after cleaning up rooms from memory above.
		debug.Print("Running GC")
		runtime.GC()
		dbg.FreeOSMemory()
	}
	c.client.Syncer = c.syncer

	debug.Print("Setting existing rooms")
	c.ui.MainView().SetRooms(c.config.Rooms)

	debug.Print("OnLogin() done.")
}

// Start moves the UI to the main view, calls OnLogin() and runs the syncer forever until stopped with Stop()
func (c *Container) Start() {
	defer debug.Recover()

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
				if httpErr, ok := err.(mautrix.HTTPError); ok && httpErr.Code == http.StatusUnauthorized {
					debug.Print("Sync() errored with ", err, " -> logging out")
					c.Logout()
				} else {
					debug.Print("Sync() errored", err)
				}
			} else {
				debug.Print("Sync() returned without error")
			}
		}
	}
}

func (c *Container) HandlePreferences(source EventSource, evt *mautrix.Event) {
	if source&EventSourceAccountData == 0 {
		return
	}
	orig := c.config.Preferences
	err := json.Unmarshal(evt.Content.VeryRaw, &c.config.Preferences)
	if err != nil {
		debug.Print("Failed to parse updated preferences:", err)
		return
	}
	debug.Print("Updated preferences:", orig, "->", c.config.Preferences)
	if c.config.AuthCache.InitialSyncDone {
		c.ui.HandleNewPreferences()
	}
}

func (c *Container) SendPreferencesToMatrix() {
	defer debug.Recover()
	debug.Print("Sending updated preferences:", c.config.Preferences)
	u := c.client.BuildURL("user", c.config.UserID, "account_data", "net.maunium.gomuks.preferences")
	_, err := c.client.MakeRequest("PUT", u, &c.config.Preferences, nil)
	if err != nil {
		debug.Print("Failed to update preferences:", err)
	}
}

func (c *Container) HandleRedaction(source EventSource, evt *mautrix.Event) {
	room := c.GetOrCreateRoom(evt.RoomID)
	var redactedEvt *event.Event
	err := c.history.Update(room, evt.Redacts, func(redacted *event.Event) error {
		redacted.Unsigned.RedactedBy = evt.ID
		redacted.Unsigned.RedactedBecause = evt
		redactedEvt = redacted
		return nil
	})
	if err != nil {
		debug.Print("Failed to mark", evt.Redacts, "as redacted:", err)
	}

	if !room.Loaded() || redactedEvt == nil {
		return
	}

	mainView := c.ui.MainView()

	roomView := mainView.GetRoom(evt.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle event %v: No room view found.", evt)
		return
	}

	// TODO make this less hacky?
	message := roomView.ParseEvent(redactedEvt)
	if message != nil {
		roomView.AddMessage(message)
		if c.syncer.FirstSyncDone {
			c.ui.Render()
		}
	}
}

// HandleMessage is the event handler for the m.room.message timeline event.
func (c *Container) HandleMessage(source EventSource, mxEvent *mautrix.Event) {
	room := c.GetOrCreateRoom(mxEvent.RoomID)
	if source&EventSourceLeave != 0 {
		room.HasLeft = true
		return
	} else if source&EventSourceState != 0 {
		return
	}

	events, err := c.history.Append(room, []*mautrix.Event{mxEvent})
	if err != nil {
		debug.Printf("Failed to add event %s to history: %v", mxEvent.ID, err)
	}
	evt := events[0]

	if !c.config.AuthCache.InitialSyncDone {
		room.LastReceivedMessage = time.Unix(evt.Timestamp/1000, evt.Timestamp%1000*1000)
		return
	}

	mainView := c.ui.MainView()

	roomView := mainView.GetRoom(evt.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle event %v: No room view found.", evt)
		return
	}

	if !room.Loaded() {
		pushRules := c.PushRules().GetActions(room, evt.Event).Should()
		shouldNotify := pushRules.Notify || !pushRules.NotifySpecified
		if !shouldNotify {
			room.LastReceivedMessage = time.Unix(evt.Timestamp/1000, evt.Timestamp%1000*1000)
			room.AddUnread(evt.ID, shouldNotify, pushRules.Highlight)
			mainView.Bump(room)
			return
		}
	}

	// TODO switch to roomView.AddEvent
	message := roomView.ParseEvent(evt)
	if message != nil {
		roomView.AddMessage(message)
		roomView.MxRoom().LastReceivedMessage = message.Time()
		if c.syncer.FirstSyncDone {
			pushRules := c.PushRules().GetActions(roomView.MxRoom(), evt.Event).Should()
			mainView.NotifyMessage(roomView.MxRoom(), message, pushRules)
			c.ui.Render()
		}
	} else {
		debug.Printf("Parsing event %s type %s %v from %s in %s failed (ParseEvent() returned nil).", evt.ID, evt.Type.String(), evt.Content.Raw, evt.Sender, evt.RoomID)
	}
}

// HandleMembership is the event handler for the m.room.member state event.
func (c *Container) HandleMembership(source EventSource, evt *mautrix.Event) {
	isLeave := source&EventSourceLeave != 0
	isTimeline := source&EventSourceTimeline != 0
	if isLeave {
		c.GetOrCreateRoom(evt.RoomID).HasLeft = true
	}
	isNonTimelineLeave := isLeave && !isTimeline
	if !c.config.AuthCache.InitialSyncDone && isNonTimelineLeave {
		return
	} else if evt.StateKey != nil && *evt.StateKey == c.config.UserID {
		c.processOwnMembershipChange(evt)
	} else if !isTimeline && (!c.config.AuthCache.InitialSyncDone || isLeave) {
		// We don't care about other users' membership events in the initial sync or chats we've left.
		return
	}

	c.HandleMessage(source, evt)
}

func (c *Container) processOwnMembershipChange(evt *mautrix.Event) {
	membership := evt.Content.Membership
	prevMembership := mautrix.MembershipLeave
	if evt.Unsigned.PrevContent != nil {
		prevMembership = evt.Unsigned.PrevContent.Membership
	}
	debug.Printf("Processing own membership change: %s->%s in %s", prevMembership, membership, evt.RoomID)
	if membership == prevMembership {
		return
	}
	room := c.GetRoom(evt.RoomID)
	switch membership {
	case "join":
		if c.config.AuthCache.InitialSyncDone {
			c.ui.MainView().AddRoom(room)
		}
		room.HasLeft = false
	case "leave":
		if c.config.AuthCache.InitialSyncDone {
			c.ui.MainView().RemoveRoom(room)
		}
		room.HasLeft = true
		room.Unload()
	case "invite":
		// TODO handle
		debug.Printf("%s invited the user to %s", evt.Sender, evt.RoomID)
	}
}

func (c *Container) parseReadReceipt(evt *mautrix.Event) (largestTimestampEvent string) {
	var largestTimestamp int64
	for eventID, rawContent := range evt.Content.Raw {
		content, ok := rawContent.(map[string]interface{})
		if !ok {
			continue
		}

		mRead, ok := content["m.read"].(map[string]interface{})
		if !ok {
			continue
		}

		myInfo, ok := mRead[c.config.UserID].(map[string]interface{})
		if !ok {
			continue
		}

		ts, ok := myInfo["ts"].(float64)
		if int64(ts) > largestTimestamp {
			largestTimestamp = int64(ts)
			largestTimestampEvent = eventID
		}
	}
	return
}

func (c *Container) HandleReadReceipt(source EventSource, evt *mautrix.Event) {
	if source&EventSourceLeave != 0 {
		return
	}

	lastReadEvent := c.parseReadReceipt(evt)
	if len(lastReadEvent) == 0 {
		return
	}

	room := c.GetRoom(evt.RoomID)
	if room != nil {
		room.MarkRead(lastReadEvent)
		if c.config.AuthCache.InitialSyncDone {
			c.ui.Render()
		}
	}
}

func (c *Container) parseDirectChatInfo(evt *mautrix.Event) map[*rooms.Room]bool {
	directChats := make(map[*rooms.Room]bool)
	for _, rawRoomIDList := range evt.Content.Raw {
		roomIDList, ok := rawRoomIDList.([]interface{})
		if !ok {
			continue
		}

		for _, rawRoomID := range roomIDList {
			roomID, ok := rawRoomID.(string)
			if !ok {
				continue
			}

			room := c.GetOrCreateRoom(roomID)
			if room != nil && !room.HasLeft {
				directChats[room] = true
			}
		}
	}
	return directChats
}

func (c *Container) HandleDirectChatInfo(source EventSource, evt *mautrix.Event) {
	directChats := c.parseDirectChatInfo(evt)
	for _, room := range c.config.Rooms.Map {
		shouldBeDirect := directChats[room]
		if shouldBeDirect != room.IsDirect {
			room.IsDirect = shouldBeDirect
			if c.config.AuthCache.InitialSyncDone {
				c.ui.MainView().UpdateTags(room)
			}
		}
	}
}

// HandlePushRules is the event handler for the m.push_rules account data event.
func (c *Container) HandlePushRules(source EventSource, evt *mautrix.Event) {
	debug.Print("Received updated push rules")
	var err error
	c.config.PushRules, err = pushrules.EventToPushRules(evt)
	if err != nil {
		debug.Print("Failed to convert event to push rules:", err)
		return
	}
	c.config.SavePushRules()
}

// HandleTag is the event handler for the m.tag account data event.
func (c *Container) HandleTag(source EventSource, evt *mautrix.Event) {
	room := c.GetOrCreateRoom(evt.RoomID)

	newTags := make([]rooms.RoomTag, len(evt.Content.RoomTags))
	index := 0
	for tag, info := range evt.Content.RoomTags {
		order := "0.5"
		if len(info.Order) > 0 {
			order = info.Order.String()
		}
		newTags[index] = rooms.RoomTag{
			Tag:   tag,
			Order: order,
		}
		index++
	}
	room.RawTags = newTags

	if c.config.AuthCache.InitialSyncDone {
		mainView := c.ui.MainView()
		mainView.UpdateTags(room)
	}
}

// HandleTyping is the event handler for the m.typing event.
func (c *Container) HandleTyping(source EventSource, evt *mautrix.Event) {
	if !c.config.AuthCache.InitialSyncDone {
		return
	}
	c.ui.MainView().SetTyping(evt.RoomID, evt.Content.TypingUserIDs)
}

func (c *Container) MarkRead(roomID, eventID string) {
	urlPath := c.client.BuildURL("rooms", roomID, "receipt", "m.read", eventID)
	c.client.MakeRequest("POST", urlPath, struct{}{}, nil)
}

var mentionRegex = regexp.MustCompile("\\[(.+?)]\\(https://matrix.to/#/@.+?:.+?\\)")
var roomRegex = regexp.MustCompile("\\[.+?]\\(https://matrix.to/#/(#.+?:[^/]+?)\\)")

func (c *Container) PrepareMarkdownMessage(roomID string, msgtype mautrix.MessageType, text string) *event.Event {
	content := format.RenderMarkdown(text)
	content.MsgType = msgtype

	// Remove markdown link stuff from plaintext mentions and room links
	content.Body = mentionRegex.ReplaceAllString(content.Body, "$1")
	content.Body = roomRegex.ReplaceAllString(content.Body, "$1")

	txnID := c.client.TxnID()
	localEcho := event.Wrap(&mautrix.Event{
		ID:        txnID,
		Sender:    c.config.UserID,
		Type:      mautrix.EventMessage,
		Timestamp: time.Now().UnixNano() / 1e6,
		RoomID:    roomID,
		Content:   content,
		Unsigned: mautrix.Unsigned{
			TransactionID: txnID,
		},
	})
	localEcho.Gomuks.OutgoingState = event.StateLocalEcho
	return localEcho
}

// SendMarkdownMessage sends a message with the given markdown text to the given room.
func (c *Container) SendEvent(event *event.Event) (string, error) {
	defer debug.Recover()

	c.SendTyping(event.RoomID, false)
	resp, err := c.client.SendMessageEvent(event.RoomID, event.Type, event.Content, mautrix.ReqSendEvent{TransactionID: event.Unsigned.TransactionID})
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

// SendTyping sets whether or not the user is typing in the given room.
func (c *Container) SendTyping(roomID string, typing bool) {
	defer debug.Recover()
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

// CreateRoom attempts to create a new room and join the user.
func (c *Container) CreateRoom(req *mautrix.ReqCreateRoom) (*rooms.Room, error) {
	resp, err := c.client.CreateRoom(req)
	if err != nil {
		return nil, err
	}
	room := c.GetOrCreateRoom(resp.RoomID)
	return room, nil
}

// JoinRoom makes the current user try to join the given room.
func (c *Container) JoinRoom(roomID, server string) (*rooms.Room, error) {
	resp, err := c.client.JoinRoom(roomID, server, nil)
	if err != nil {
		return nil, err
	}

	room := c.GetRoom(resp.RoomID)
	room.HasLeft = false
	return room, nil
}

// LeaveRoom makes the current user leave the given room.
func (c *Container) LeaveRoom(roomID string) error {
	_, err := c.client.LeaveRoom(roomID)
	if err != nil {
		return err
	}

	node := c.GetOrCreateRoom(roomID)
	node.HasLeft = true
	node.Unload()
	return nil
}

// GetHistory fetches room history.
func (c *Container) GetHistory(room *rooms.Room, limit int) ([]*event.Event, error) {
	events, err := c.history.Load(room, limit)
	if err != nil {
		return nil, err
	}
	if len(events) > 0 {
		debug.Printf("Loaded %d events for %s from local cache", len(events), room.ID)
		return events, nil
	}
	resp, err := c.client.Messages(room.ID, room.PrevBatch, "", 'b', limit)
	if err != nil {
		return nil, err
	}
	debug.Printf("Loaded %d events for %s from server from %s to %s", len(resp.Chunk), room.ID, resp.Start, resp.End)
	room.PrevBatch = resp.End
	c.config.Rooms.Put(room)
	if len(resp.Chunk) == 0 {
		return []*event.Event{}, nil
	}
	events, err = c.history.Prepend(room, resp.Chunk)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Container) GetEvent(room *rooms.Room, eventID string) (*event.Event, error) {
	evt, err := c.history.Get(room, eventID)
	if evt != nil || err != nil {
		debug.Printf("Found event %s in local cache", eventID)
		return evt, err
	}
	mxEvent, err := c.client.GetEvent(room.ID, eventID)
	if err != nil {
		return nil, err
	}
	evt = event.Wrap(mxEvent)
	debug.Printf("Loaded event %s from server", eventID)
	return evt, nil
}

// GetOrCreateRoom gets the room instance stored in the session.
func (c *Container) GetOrCreateRoom(roomID string) *rooms.Room {
	return c.config.Rooms.GetOrCreate(roomID)
}

// GetRoom gets the room instance stored in the session.
func (c *Container) GetRoom(roomID string) *rooms.Room {
	return c.config.Rooms.Get(roomID)
}

var mxcRegex = regexp.MustCompile("mxc://(.+)/(.+)")

// Download fetches the given Matrix content (mxc) URL and returns the data, homeserver, file ID and potential errors.
//
// The file will be either read from the media cache (if found) or downloaded from the server.
func (c *Container) Download(mxcURL string) (data []byte, hs, id string, err error) {
	parts := mxcRegex.FindStringSubmatch(mxcURL)
	if parts == nil || len(parts) != 3 {
		err = fmt.Errorf("invalid matrix content URL")
		return
	}

	hs = parts[1]
	id = parts[2]

	cacheFile := c.GetCachePath(hs, id)
	var info os.FileInfo
	if info, err = os.Stat(cacheFile); err == nil && !info.IsDir() {
		data, err = ioutil.ReadFile(cacheFile)
		if err == nil {
			return
		}
	}

	data, err = c.download(hs, id, cacheFile)
	return
}

func (c *Container) GetDownloadURL(hs, id string) string {
	dlURL, _ := url.Parse(c.client.HomeserverURL.String())
	if dlURL.Scheme == "" {
		dlURL.Scheme = "https"
	}
	dlURL.Path = path.Join(dlURL.Path, "/_matrix/media/v1/download", hs, id)
	return dlURL.String()
}

func (c *Container) download(hs, id, cacheFile string) (data []byte, err error) {
	var resp *http.Response
	resp, err = c.client.Client.Get(c.GetDownloadURL(hs, id))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return
	}

	data = buf.Bytes()

	err = ioutil.WriteFile(cacheFile, data, 0600)
	return
}

// GetCachePath gets the path to the cached version of the given homeserver:fileID combination.
// The file may or may not exist, use Download() to ensure it has been cached.
func (c *Container) GetCachePath(homeserver, fileID string) string {
	dir := filepath.Join(c.config.MediaDir, homeserver)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return ""
	}

	return filepath.Join(dir, fileID)
}
