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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"crypto/tls"
	"encoding/json"

	"gopkg.in/russross/blackfriday.v2"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/bfhtml"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
)

// Container is a wrapper for a gomatrix Client and some other stuff.
//
// It is used for all Matrix calls from the UI and Matrix event handlers.
type Container struct {
	client  *gomatrix.Client
	syncer  *GomuksSyncer
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

// Client returns the underlying gomatrix Client.
func (c *Container) Client() *gomatrix.Client {
	return c.client
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
	if len(c.config.AccessToken) > 0 {
		accessToken = c.config.AccessToken
		mxid = c.config.UserID
	}

	var err error
	c.client, err = gomatrix.NewClient(c.config.HS, mxid, accessToken)
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

// Initialized returns whether or not the gomatrix client is initialized (see InitClient())
func (c *Container) Initialized() bool {
	return c.client != nil
}

// Login sends a password login request with the given username and password.
func (c *Container) Login(user, password string) error {
	resp, err := c.client.Login(&gomatrix.ReqLogin{
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
	}
}

// UpdatePushRules fetches the push notification rules from the server and stores them in the current Session object.
func (c *Container) UpdatePushRules() {
	debug.Print("Updating push rules...")
	resp, err := pushrules.GetPushRules(c.client)
	if err != nil {
		debug.Print("Failed to fetch push rules:", err)
	}
	c.config.PushRules = resp
	c.config.SavePushRules()
}

// PushRules returns the push notification rules. If no push rules are cached, UpdatePushRules() will be called first.
func (c *Container) PushRules() *pushrules.PushRuleset {
	if c.config.PushRules == nil {
		c.UpdatePushRules()
	}
	return c.config.PushRules
}

// OnLogin initializes the syncer and updates the room list.
func (c *Container) OnLogin() {
	c.ui.OnLogin()

	c.client.Store = c.config

	debug.Print("Initializing syncer")
	c.syncer = NewGomuksSyncer(c.config)
	c.syncer.OnEventType("m.room.message", c.HandleMessage)
	c.syncer.OnEventType("m.room.member", c.HandleMembership)
	c.syncer.OnEventType("m.receipt", c.HandleReadReceipt)
	c.syncer.OnEventType("m.typing", c.HandleTyping)
	c.syncer.OnEventType("m.direct", c.HandleDirectChatInfo)
	c.syncer.OnEventType("m.push_rules", c.HandlePushRules)
	c.syncer.OnEventType("m.tag", c.HandleTag)
	c.syncer.OnEventType("net.maunium.gomuks.preferences", c.HandlePreferences)
	c.syncer.InitDoneCallback = func() {
		c.config.AuthCache.InitialSyncDone = true
		c.config.SaveAuthCache()
		c.ui.MainView().InitialSyncDone()
		c.ui.Render()
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
				if httpErr, ok := err.(gomatrix.HTTPError); ok && httpErr.Code == http.StatusUnauthorized {
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

func (c *Container) HandlePreferences(source EventSource, evt *gomatrix.Event) {
	orig := c.config.Preferences
	rt, _ := json.Marshal(&evt.Content)
	json.Unmarshal(rt, &c.config.Preferences)
	debug.Print("Updated preferences:", orig, "->", c.config.Preferences)
	c.ui.HandleNewPreferences()
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

// HandleMessage is the event handler for the m.room.message timeline event.
func (c *Container) HandleMessage(source EventSource, evt *gomatrix.Event) {
	if source&EventSourceLeave != 0 {
		return
	}
	mainView := c.ui.MainView()

	roomView := mainView.GetRoom(evt.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle event %v: No room view found.", evt)
		return
	}

	message := mainView.ParseEvent(roomView, evt)
	if message != nil {
		roomView.AddMessage(message, ifc.AppendMessage)
		roomView.MxRoom().LastReceivedMessage = message.Timestamp()
		if c.syncer.FirstSyncDone {
			pushRules := c.PushRules().GetActions(roomView.MxRoom(), evt).Should()
			mainView.NotifyMessage(roomView.MxRoom(), message, pushRules)
			c.ui.Render()
		}
	} else {
		debug.Printf("Parsing event %v failed (ParseEvent() returned nil).", evt)
	}
}

// HandleMembership is the event handler for the m.room.member state event.
func (c *Container) HandleMembership(source EventSource, evt *gomatrix.Event) {
	isLeave := source&EventSourceLeave != 0
	isTimeline := source&EventSourceTimeline != 0
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

func (c *Container) processOwnMembershipChange(evt *gomatrix.Event) {
	membership, _ := evt.Content["membership"].(string)
	prevMembership := "leave"
	if evt.Unsigned.PrevContent != nil {
		prevMembership, _ = evt.Unsigned.PrevContent["membership"].(string)
	}
	debug.Printf("Processing own membership change: %s->%s in %s", prevMembership, membership, evt.RoomID)
	if membership == prevMembership {
		return
	}
	room := c.GetRoom(evt.RoomID)
	switch membership {
	case "join":
		c.ui.MainView().AddRoom(room)
		room.HasLeft = false
	case "leave":
		c.ui.MainView().RemoveRoom(room)
		room.HasLeft = true
	case "invite":
		// TODO handle
		debug.Printf("%s invited the user to %s", evt.Sender, evt.RoomID)
	}
}

func (c *Container) parseReadReceipt(evt *gomatrix.Event) (largestTimestampEvent string) {
	var largestTimestamp int64
	for eventID, rawContent := range evt.Content {
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

func (c *Container) HandleReadReceipt(source EventSource, evt *gomatrix.Event) {
	if source&EventSourceLeave != 0 {
		return
	}

	lastReadEvent := c.parseReadReceipt(evt)
	if len(lastReadEvent) == 0 {
		return
	}

	room := c.GetRoom(evt.RoomID)
	room.MarkRead(lastReadEvent)
	c.ui.Render()
}

func (c *Container) parseDirectChatInfo(evt *gomatrix.Event) map[*rooms.Room]bool {
	directChats := make(map[*rooms.Room]bool)
	for _, rawRoomIDList := range evt.Content {
		roomIDList, ok := rawRoomIDList.([]interface{})
		if !ok {
			continue
		}

		for _, rawRoomID := range roomIDList {
			roomID, ok := rawRoomID.(string)
			if !ok {
				continue
			}

			room := c.GetRoom(roomID)
			if room != nil && !room.HasLeft {
				directChats[room] = true
			}
		}
	}
	return directChats
}

func (c *Container) HandleDirectChatInfo(source EventSource, evt *gomatrix.Event) {
	directChats := c.parseDirectChatInfo(evt)
	for _, room := range c.config.Rooms {
		shouldBeDirect := directChats[room]
		if shouldBeDirect != room.IsDirect {
			room.IsDirect = shouldBeDirect
			c.ui.MainView().UpdateTags(room)
		}
	}
}

// HandlePushRules is the event handler for the m.push_rules account data event.
func (c *Container) HandlePushRules(source EventSource, evt *gomatrix.Event) {
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
func (c *Container) HandleTag(source EventSource, evt *gomatrix.Event) {
	room := c.config.GetRoom(evt.RoomID)

	tags, _ := evt.Content["tags"].(map[string]interface{})
	newTags := make([]rooms.RoomTag, len(tags))
	index := 0
	for tag, infoifc := range tags {
		info, _ := infoifc.(map[string]interface{})
		order := "0.5"
		rawOrder, ok := info["order"]
		if ok {
			order = fmt.Sprintf("%v", rawOrder)
		}
		newTags[index] = rooms.RoomTag{
			Tag:   tag,
			Order: order,
		}
		index++
	}

	mainView := c.ui.MainView()
	room.RawTags = newTags
	mainView.UpdateTags(room)
}

// HandleTyping is the event handler for the m.typing event.
func (c *Container) HandleTyping(source EventSource, evt *gomatrix.Event) {
	users := evt.Content["user_ids"].([]interface{})

	strUsers := make([]string, len(users))
	for i, user := range users {
		strUsers[i] = user.(string)
	}
	c.ui.MainView().SetTyping(evt.RoomID, strUsers)
}

func (c *Container) MarkRead(roomID, eventID string) {
	urlPath := c.client.BuildURL("rooms", roomID, "receipt", "m.read", eventID)
	c.client.MakeRequest("POST", urlPath, struct{}{}, nil)
}

// SendMessage sends a message with the given text to the given room.
func (c *Container) SendMessage(roomID, msgtype, text string) (string, error) {
	defer debug.Recover()
	c.SendTyping(roomID, false)
	resp, err := c.client.SendMessageEvent(roomID, "m.room.message",
		gomatrix.TextMessage{MsgType: msgtype, Body: text})
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

func (c *Container) renderMarkdown(text string) string {
	parser := blackfriday.New(
		blackfriday.WithExtensions(blackfriday.NoIntraEmphasis |
			blackfriday.Tables |
			blackfriday.FencedCode |
			blackfriday.Strikethrough |
			blackfriday.SpaceHeadings |
			blackfriday.DefinitionLists))
	ast := parser.Parse([]byte(text))

	renderer := bfhtml.HTMLRenderer{
		HTMLRenderer: blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
			Flags: blackfriday.UseXHTML,
		}),
	}

	var buf strings.Builder
	renderer.RenderHeader(&buf, ast)
	ast.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		return renderer.RenderNode(&buf, node, entering)
	})
	renderer.RenderFooter(&buf, ast)
	return buf.String()
}

var mentionRegex = regexp.MustCompile("\\[(.+?)]\\(https://matrix.to/#/@.+?:.+?\\)")
var roomRegex = regexp.MustCompile("\\[.+?]\\(https://matrix.to/#/(#.+?:[^/]+?)\\)")

// SendMarkdownMessage sends a message with the given text to the given room.
//
// If the given text contains markdown formatting symbols, it will be rendered into HTML before sending.
// Otherwise, it will be sent as plain text.
func (c *Container) SendMarkdownMessage(roomID, msgtype, text string) (string, error) {
	defer debug.Recover()

	html := c.renderMarkdown(text)
	if html == text {
		return c.SendMessage(roomID, msgtype, text)
	}

	// Remove markdown link stuff from plaintext mentions and room links
	text = mentionRegex.ReplaceAllString(text, "$1")
	text = roomRegex.ReplaceAllString(text, "$1")

	c.SendTyping(roomID, false)
	resp, err := c.client.SendMessageEvent(roomID, "m.room.message",
		map[string]interface{}{
			"msgtype":        msgtype,
			"body":           text,
			"format":         "org.matrix.custom.html",
			"formatted_body": html,
		})
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

	room := c.GetRoom(roomID)
	room.HasLeft = true
	return nil
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
func (c *Container) GetRoom(roomID string) *rooms.Room {
	return c.config.GetRoom(roomID)
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
