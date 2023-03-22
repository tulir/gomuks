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

package matrix

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	dbg "runtime/debug"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/open"
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/matrix/rooms"
)

// Container is a wrapper for a mautrix Client and some other stuff.
//
// It is used for all Matrix calls from the UI and Matrix event handlers.
type Container struct {
	client  *mautrix.Client
	crypto  ifc.Crypto
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

func (c *Container) Crypto() ifc.Crypto {
	return c.crypto
}

var (
	ErrNoHomeserver   = errors.New("no homeserver entered")
	ErrServerOutdated = errors.New("homeserver is outdated")
)

var MinSpecVersion = mautrix.SpecV11
var SkipVersionCheck = false

// InitClient initializes the mautrix client and connects to the homeserver specified in the config.
func (c *Container) InitClient(isStartup bool) error {
	if len(c.config.HS) == 0 {
		if isStartup {
			return nil
		}
		return ErrNoHomeserver
	}

	if c.client != nil {
		c.Stop()
		c.client = nil
		c.crypto = nil
	}

	var mxid id.UserID
	var accessToken string
	if len(c.config.AccessToken) > 0 {
		accessToken = c.config.AccessToken
		mxid = c.config.UserID
	}

	var err error
	c.client, err = mautrix.NewClient(c.config.HS, mxid, accessToken)
	if err != nil {
		return fmt.Errorf("failed to create mautrix client: %w", err)
	}
	c.client.UserAgent = fmt.Sprintf("gomuks/%s %s", c.gmx.Version(), mautrix.DefaultUserAgent)
	c.client.Logger = mxLogger{}
	c.client.DeviceID = c.config.DeviceID

	err = c.initCrypto()
	if err != nil {
		return fmt.Errorf("failed to initialize crypto: %w", err)
	}

	if c.history == nil {
		c.history, err = NewHistoryManager(c.config.HistoryPath)
		if err != nil {
			return fmt.Errorf("failed to initialize history: %w", err)
		}
	}

	allowInsecure := len(os.Getenv("GOMUKS_ALLOW_INSECURE_CONNECTIONS")) > 0
	if allowInsecure {
		c.client.Client = &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		}
	}

	if !SkipVersionCheck && (!isStartup || len(c.client.AccessToken) > 0) {
		debug.Printf("Checking versions that %s supports", c.client.HomeserverURL)
		resp, err := c.client.Versions()
		if err != nil {
			debug.Print("Error checking supported versions:", err)
			return fmt.Errorf("failed to check server versions: %w", err)
		} else if !resp.ContainsGreaterOrEqual(MinSpecVersion) {
			debug.Print("Server doesn't support modern spec versions")
			bestVersionStr := "nothing"
			bestVersion := mautrix.MustParseSpecVersion("r0.0.0")
			for _, ver := range resp.Versions {
				if ver.GreaterThan(bestVersion) {
					bestVersion = ver
					bestVersionStr = ver.String()
				}
			}
			return fmt.Errorf("%w (it only supports %s, while gomuks requires %s)", ErrServerOutdated, bestVersionStr, MinSpecVersion.String())
		} else {
			debug.Print("Server supports modern spec versions")
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

func (c *Container) PasswordLogin(user, password string) error {
	resp, err := c.client.Login(&mautrix.ReqLogin{
		Type: "m.login.password",
		Identifier: mautrix.UserIdentifier{
			Type: "m.id.user",
			User: user,
		},
		Password:                 password,
		InitialDeviceDisplayName: "gomuks",

		StoreCredentials:   true,
		StoreHomeserverURL: true,
	})
	if err != nil {
		return err
	}
	c.finishLogin(resp)
	return nil
}

func (c *Container) finishLogin(resp *mautrix.RespLogin) {
	c.config.UserID = resp.UserID
	c.config.DeviceID = resp.DeviceID
	c.config.AccessToken = resp.AccessToken
	if resp.WellKnown != nil && len(resp.WellKnown.Homeserver.BaseURL) > 0 {
		c.config.HS = resp.WellKnown.Homeserver.BaseURL
	}
	c.config.Save()

	go c.Start()
}

func respondHTML(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>gomuks single-sign on</title>
  <meta charset="utf-8"/>
</head>
<body>
  <center>
    <h2>%s</h2>
  </center>
</body>
</html>`, message)))
}

func (c *Container) SingleSignOn() error {
	loginURL := c.client.BuildURLWithQuery(mautrix.ClientURLPath{"v3", "login", "sso", "redirect"}, map[string]string{
		"redirectUrl": "http://localhost:29325",
	})
	err := open.Open(loginURL)
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	server := &http.Server{Addr: ":29325"}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginToken := r.URL.Query().Get("loginToken")
		if len(loginToken) == 0 {
			respondHTML(w, http.StatusBadRequest, "Missing loginToken parameter")
			return
		}
		resp, err := c.client.Login(&mautrix.ReqLogin{
			Type:                     "m.login.token",
			Token:                    loginToken,
			InitialDeviceDisplayName: "gomuks",

			StoreCredentials:   true,
			StoreHomeserverURL: true,
		})
		if err != nil {
			respondHTML(w, http.StatusForbidden, err.Error())
			errChan <- err
			return
		}
		respondHTML(w, http.StatusOK, fmt.Sprintf("Successfully logged in as %s", resp.UserID))
		c.finishLogin(resp)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = server.Shutdown(ctx)
			if err != nil {
				debug.Printf("Failed to shut down SSO server: %v\n", err)
			}
			errChan <- err
		}()
	})
	err = server.ListenAndServe()
	if err != nil {
		return err
	}
	err = <-errChan
	return err
}

// Login sends a password login request with the given username and password.
func (c *Container) Login(user, password string) error {
	resp, err := c.client.GetLoginFlows()
	if err != nil {
		return err
	}
	ssoSkippedBecausePassword := false
	for _, flow := range resp.Flows {
		if flow.Type == "m.login.password" {
			return c.PasswordLogin(user, password)
		} else if flow.Type == "m.login.sso" {
			if len(password) == 0 {
				return c.SingleSignOn()
			} else {
				ssoSkippedBecausePassword = true
			}
		}
	}
	if ssoSkippedBecausePassword {
		return fmt.Errorf("password login is not supported\nleave the password field blank to use SSO")
	}
	return fmt.Errorf("no supported login flows")
}

// Logout revokes the access token, stops the syncer and calls the OnLogout() method of the UI.
func (c *Container) Logout() {
	c.client.Logout()
	c.Stop()
	c.config.DeleteSession()
	c.client = nil
	c.crypto = nil
	c.ui.OnLogout()
}

// Stop stops the Matrix syncer.
func (c *Container) Stop() {
	if c.running {
		debug.Print("Stopping Matrix container...")
		select {
		case c.stop <- true:
		default:
		}
		c.client.StopSync()
		debug.Print("Closing history manager...")
		err := c.history.Close()
		if err != nil {
			debug.Print("Error closing history manager:", err)
		}
		c.history = nil
		if c.crypto != nil {
			debug.Print("Flushing crypto store")
			err = c.crypto.FlushStore()
			if err != nil {
				debug.Print("Error flushing crypto store:", err)
			}
		}
	}
}

// UpdatePushRules fetches the push notification rules from the server and stores them in the current Session object.
func (c *Container) UpdatePushRules() {
	debug.Print("Updating push rules...")
	resp, err := c.client.GetPushRules()
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

var AccountDataGomuksPreferences = event.Type{
	Type:  "net.maunium.gomuks.preferences",
	Class: event.AccountDataEventType,
}

func init() {
	event.TypeMap[AccountDataGomuksPreferences] = reflect.TypeOf(config.UserPreferences{})
	gob.Register(&config.UserPreferences{})
}

type StubSyncingModal struct{}

func (s StubSyncingModal) SetIndeterminate()    {}
func (s StubSyncingModal) SetMessage(s2 string) {}
func (s StubSyncingModal) SetSteps(i int)       {}
func (s StubSyncingModal) Step()                {}
func (s StubSyncingModal) Close()               {}

// OnLogin initializes the syncer and updates the room list.
func (c *Container) OnLogin() {
	c.cryptoOnLogin()
	c.ui.OnLogin()

	c.client.Store = c.config

	debug.Print("Initializing syncer")
	c.syncer = NewGomuksSyncer(c.config.Rooms)
	if c.crypto != nil {
		c.syncer.OnSync(c.crypto.ProcessSyncResponse)
		c.syncer.OnEventType(event.StateMember, func(source mautrix.EventSource, evt *event.Event) {
			// Don't spam the crypto module with member events of an initial sync
			// TODO invalidate all group sessions when clearing cache?
			if c.config.AuthCache.InitialSyncDone {
				c.crypto.HandleMemberEvent(evt)
			}
		})
		c.syncer.OnEventType(event.EventEncrypted, c.HandleEncrypted)
	} else {
		c.syncer.OnEventType(event.EventEncrypted, c.HandleEncryptedUnsupported)
	}
	c.syncer.OnEventType(event.EventMessage, c.HandleMessage)
	c.syncer.OnEventType(event.EventSticker, c.HandleMessage)
	c.syncer.OnEventType(event.EventReaction, c.HandleMessage)
	c.syncer.OnEventType(event.EventRedaction, c.HandleRedaction)
	c.syncer.OnEventType(event.StateAliases, c.HandleMessage)
	c.syncer.OnEventType(event.StateCanonicalAlias, c.HandleMessage)
	c.syncer.OnEventType(event.StateTopic, c.HandleMessage)
	c.syncer.OnEventType(event.StateRoomName, c.HandleMessage)
	c.syncer.OnEventType(event.StateMember, c.HandleMembership)
	c.syncer.OnEventType(event.EphemeralEventReceipt, c.HandleReadReceipt)
	c.syncer.OnEventType(event.EphemeralEventTyping, c.HandleTyping)
	c.syncer.OnEventType(event.AccountDataDirectChats, c.HandleDirectChatInfo)
	c.syncer.OnEventType(event.AccountDataPushRules, c.HandlePushRules)
	c.syncer.OnEventType(event.AccountDataRoomTags, c.HandleTag)
	c.syncer.OnEventType(AccountDataGomuksPreferences, c.HandlePreferences)
	if len(c.config.AuthCache.NextBatch) == 0 {
		c.syncer.Progress = c.ui.MainView().OpenSyncingModal()
		c.syncer.Progress.SetMessage("Waiting for /sync response from server")
		c.syncer.Progress.SetIndeterminate()
		c.syncer.FirstDoneCallback = func() {
			c.syncer.Progress.Close()
			c.syncer.Progress = StubSyncingModal{}
			c.syncer.FirstDoneCallback = nil
		}
	}
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
	c.client.StreamSyncMinAge = 30 * time.Minute
	for {
		select {
		case <-c.stop:
			debug.Print("Stopping sync...")
			c.running = false
			return
		default:
			if err := c.client.Sync(); err != nil {
				if errors.Is(err, mautrix.MUnknownToken) {
					debug.Print("Sync() errored with ", err, " -> logging out")
					// TODO support soft logout
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

func (c *Container) HandlePreferences(source mautrix.EventSource, evt *event.Event) {
	if source&mautrix.EventSourceAccountData == 0 {
		return
	}
	orig := c.config.Preferences
	err := json.Unmarshal(evt.Content.VeryRaw, &c.config.Preferences)
	if err != nil {
		debug.Print("Failed to parse updated preferences:", err)
		return
	}
	debug.Printf("Updated preferences: %#v -> %#v", orig, c.config.Preferences)
	if c.config.AuthCache.InitialSyncDone {
		c.ui.HandleNewPreferences()
	}
}

func (c *Container) Preferences() *config.UserPreferences {
	return &c.config.Preferences
}

func (c *Container) SendPreferencesToMatrix() {
	defer debug.Recover()
	debug.Printf("Sending updated preferences: %#v", c.config.Preferences)
	err := c.client.SetAccountData(AccountDataGomuksPreferences.Type, &c.config.Preferences)
	if err != nil {
		debug.Print("Failed to update preferences:", err)
	}
}

func (c *Container) HandleRedaction(source mautrix.EventSource, evt *event.Event) {
	room := c.GetOrCreateRoom(evt.RoomID)
	var redactedEvt *muksevt.Event
	err := c.history.Update(room, evt.Redacts, func(redacted *muksevt.Event) error {
		redacted.Unsigned.RedactedBecause = evt
		redactedEvt = redacted
		return nil
	})
	if err != nil {
		debug.Print("Failed to mark", evt.Redacts, "as redacted:", err)
		return
	} else if !c.config.AuthCache.InitialSyncDone || !room.Loaded() {
		return
	}

	roomView := c.ui.MainView().GetRoom(evt.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle event %v: No room view found.", evt)
		return
	}

	roomView.AddRedaction(redactedEvt)
	if c.syncer.FirstSyncDone {
		c.ui.Render()
	}
}

var ErrCantEditOthersMessage = errors.New("can't edit message sent by someone else")

func (c *Container) HandleEdit(room *rooms.Room, editsID id.EventID, editEvent *muksevt.Event) {
	var origEvt *muksevt.Event
	err := c.history.Update(room, editsID, func(evt *muksevt.Event) error {
		if editEvent.Sender != evt.Sender {
			return ErrCantEditOthersMessage
		}
		evt.Gomuks.Edits = append(evt.Gomuks.Edits, editEvent)
		origEvt = evt
		return nil
	})
	if err == ErrCantEditOthersMessage {
		debug.Printf("Ignoring edit %s of %s by %s in %s: original event was sent by someone else", editEvent.ID, editsID, editEvent.Sender, editEvent.RoomID)
		return
	} else if err != nil {
		debug.Print("Failed to store edit in history db:", err)
		return
	} else if !c.config.AuthCache.InitialSyncDone || !room.Loaded() {
		return
	}

	roomView := c.ui.MainView().GetRoom(editEvent.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle edit event %v: No room view found.", editEvent)
		return
	}

	roomView.AddEdit(origEvt)
	if c.syncer.FirstSyncDone {
		c.ui.Render()
	}
}

func (c *Container) HandleReaction(room *rooms.Room, reactsTo id.EventID, reactEvent *muksevt.Event) {
	rel := reactEvent.Content.AsReaction().RelatesTo
	var origEvt *muksevt.Event
	err := c.history.Update(room, reactsTo, func(evt *muksevt.Event) error {
		if evt.Unsigned.Relations.Annotations.Map == nil {
			evt.Unsigned.Relations.Annotations.Map = make(map[string]int)
		}
		val, _ := evt.Unsigned.Relations.Annotations.Map[rel.Key]
		evt.Unsigned.Relations.Annotations.Map[rel.Key] = val + 1
		origEvt = evt
		return nil
	})
	if err != nil {
		debug.Print("Failed to store reaction in history db:", err)
		return
	} else if !c.config.AuthCache.InitialSyncDone || !room.Loaded() {
		return
	}

	roomView := c.ui.MainView().GetRoom(reactEvent.RoomID)
	if roomView == nil {
		debug.Printf("Failed to handle edit event %v: No room view found.", reactEvent)
		return
	}

	roomView.AddReaction(origEvt, rel.Key)
	if c.syncer.FirstSyncDone {
		c.ui.Render()
	}
}

func (c *Container) HandleEncryptedUnsupported(source mautrix.EventSource, mxEvent *event.Event) {
	mxEvent.Type = muksevt.EventEncryptionUnsupported
	origContent, _ := mxEvent.Content.Parsed.(*event.EncryptedEventContent)
	mxEvent.Content.Parsed = muksevt.EncryptionUnsupportedContent{Original: origContent}
	c.HandleMessage(source, mxEvent)
}

func (c *Container) HandleEncrypted(source mautrix.EventSource, mxEvent *event.Event) {
	evt, err := c.crypto.DecryptMegolmEvent(mxEvent)
	if err != nil {
		debug.Printf("Failed to decrypt event %s: %v", mxEvent.ID, err)
		mxEvent.Type = muksevt.EventBadEncrypted
		origContent, _ := mxEvent.Content.Parsed.(*event.EncryptedEventContent)
		mxEvent.Content.Parsed = &muksevt.BadEncryptedContent{
			Original: origContent,
			Reason:   err.Error(),
		}
		c.HandleMessage(source, mxEvent)
		return
	}
	if evt.Type.IsInRoomVerification() {
		err := c.crypto.ProcessInRoomVerification(evt)
		if err != nil {
			debug.Printf("[Crypto/Error] Failed to process in-room verification event %s of type %s: %v", evt.ID, evt.Type.String(), err)
		} else {
			debug.Printf("[Crypto/Debug] Processed in-room verification event %s of type %s", evt.ID, evt.Type.String())
		}
	} else {
		c.HandleMessage(source, evt)
	}
}

// HandleMessage is the event handler for the m.room.message timeline event.
func (c *Container) HandleMessage(source mautrix.EventSource, mxEvent *event.Event) {
	room := c.GetOrCreateRoom(mxEvent.RoomID)
	if source&mautrix.EventSourceLeave != 0 {
		room.HasLeft = true
		return
	} else if source&mautrix.EventSourceState != 0 {
		return
	}

	relatable, ok := mxEvent.Content.Parsed.(event.Relatable)
	if ok {
		rel := relatable.GetRelatesTo()
		if editID := rel.GetReplaceID(); len(editID) > 0 {
			c.HandleEdit(room, editID, muksevt.Wrap(mxEvent))
			return
		} else if reactionID := rel.GetAnnotationID(); mxEvent.Type == event.EventReaction && len(reactionID) > 0 {
			c.HandleReaction(room, reactionID, muksevt.Wrap(mxEvent))
			return
		}
	}

	events, err := c.history.Append(room, []*event.Event{mxEvent})
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
		if !pushRules.Notify {
			room.LastReceivedMessage = time.Unix(evt.Timestamp/1000, evt.Timestamp%1000*1000)
			room.AddUnread(evt.ID, pushRules.Notify, pushRules.Highlight)
			mainView.Bump(room)
			return
		}
	}

	message := roomView.AddEvent(evt)
	if message != nil {
		roomView.MxRoom().LastReceivedMessage = message.Time()
		if c.syncer.FirstSyncDone && evt.Sender != c.config.UserID {
			pushRules := c.PushRules().GetActions(roomView.MxRoom(), evt.Event).Should()
			mainView.NotifyMessage(roomView.MxRoom(), message, pushRules)
			c.ui.Render()
		}
	} else {
		debug.Printf("Parsing event %s type %s %v from %s in %s failed (ParseEvent() returned nil).", evt.ID, evt.Type.Repr(), evt.Content.Raw, evt.Sender, evt.RoomID)
	}
}

// HandleMembership is the event handler for the m.room.member state event.
func (c *Container) HandleMembership(source mautrix.EventSource, evt *event.Event) {
	isLeave := source&mautrix.EventSourceLeave != 0
	isTimeline := source&mautrix.EventSourceTimeline != 0
	if isLeave {
		c.GetOrCreateRoom(evt.RoomID).HasLeft = true
	}
	isNonTimelineLeave := isLeave && !isTimeline
	if !c.config.AuthCache.InitialSyncDone && isNonTimelineLeave {
		return
	} else if evt.StateKey != nil && id.UserID(*evt.StateKey) == c.config.UserID {
		c.processOwnMembershipChange(evt)
	} else if !isTimeline && (!c.config.AuthCache.InitialSyncDone || isLeave) {
		// We don't care about other users' membership events in the initial sync or chats we've left.
		return
	}

	c.HandleMessage(source, evt)
}

func (c *Container) processOwnMembershipChange(evt *event.Event) {
	membership := evt.Content.AsMember().Membership
	prevMembership := event.MembershipLeave
	if evt.Unsigned.PrevContent != nil {
		prevMembership = evt.Unsigned.PrevContent.AsMember().Membership
	}
	debug.Printf("Processing own membership change: %s->%s in %s", prevMembership, membership, evt.RoomID)
	if membership == prevMembership {
		return
	}
	room := c.GetRoom(evt.RoomID)
	switch membership {
	case "join":
		room.HasLeft = false
		if c.config.AuthCache.InitialSyncDone {
			c.ui.MainView().UpdateTags(room)
		}
		fallthrough
	case "invite":
		if c.config.AuthCache.InitialSyncDone {
			c.ui.MainView().AddRoom(room)
		}
	case "leave":
	case "ban":
		if c.config.AuthCache.InitialSyncDone {
			c.ui.MainView().RemoveRoom(room)
		}
		room.HasLeft = true
		room.Unload()
	default:
		return
	}
	c.ui.Render()
}

func (c *Container) parseReadReceipt(evt *event.Event) (largestTimestampEvent id.EventID) {
	var largestTimestamp int64

	for eventID, receipts := range *evt.Content.AsReceipt() {
		myInfo, ok := receipts.Read[c.config.UserID]
		if !ok {
			continue
		}

		if myInfo.Timestamp > largestTimestamp {
			largestTimestamp = myInfo.Timestamp
			largestTimestampEvent = eventID
		}
	}
	return
}

func (c *Container) HandleReadReceipt(source mautrix.EventSource, evt *event.Event) {
	if source&mautrix.EventSourceLeave != 0 {
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

func (c *Container) parseDirectChatInfo(evt *event.Event) map[*rooms.Room]id.UserID {
	directChats := make(map[*rooms.Room]id.UserID)
	for userID, roomIDList := range *evt.Content.AsDirectChats() {
		for _, roomID := range roomIDList {
			// TODO we shouldn't create direct chat rooms that we aren't in
			room := c.GetOrCreateRoom(roomID)
			if room != nil && !room.HasLeft {
				directChats[room] = userID
			}
		}
	}
	return directChats
}

func (c *Container) HandleDirectChatInfo(_ mautrix.EventSource, evt *event.Event) {
	directChats := c.parseDirectChatInfo(evt)
	for _, room := range c.config.Rooms.Map {
		userID, isDirect := directChats[room]
		if isDirect != room.IsDirect {
			room.IsDirect = isDirect
			room.OtherUser = userID
			if c.config.AuthCache.InitialSyncDone {
				c.ui.MainView().UpdateTags(room)
			}
		}
	}
}

// HandlePushRules is the event handler for the m.push_rules account data event.
func (c *Container) HandlePushRules(_ mautrix.EventSource, evt *event.Event) {
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
func (c *Container) HandleTag(_ mautrix.EventSource, evt *event.Event) {
	room := c.GetOrCreateRoom(evt.RoomID)

	tags := evt.Content.AsTag().Tags

	newTags := make([]rooms.RoomTag, len(tags))
	index := 0
	for tag, info := range tags {
		order := json.Number("0.5")
		if len(info.Order) > 0 {
			order = info.Order
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
func (c *Container) HandleTyping(_ mautrix.EventSource, evt *event.Event) {
	if !c.config.AuthCache.InitialSyncDone {
		return
	}
	c.ui.MainView().SetTyping(evt.RoomID, evt.Content.AsTyping().UserIDs)
}

func (c *Container) MarkRead(roomID id.RoomID, eventID id.EventID) {
	go func() {
		defer debug.Recover()
		err := c.client.MarkRead(roomID, eventID)
		if err != nil {
			debug.Printf("Failed to mark %s in %s as read: %v", eventID, roomID, err)
		}
	}()
}

func (c *Container) PrepareMediaMessage(room *rooms.Room, path string, rel *ifc.Relation) (*muksevt.Event, error) {
	resp, err := c.UploadMedia(path, room.Encrypted)
	if err != nil {
		return nil, err
	}
	content := event.MessageEventContent{
		MsgType: resp.MsgType,
		Body:    resp.Name,
		Info:    resp.Info,
	}
	if resp.EncryptionInfo != nil {
		content.File = &event.EncryptedFileInfo{
			EncryptedFile: *resp.EncryptionInfo,
			URL:           resp.ContentURI.CUString(),
		}
	} else {
		content.URL = resp.ContentURI.CUString()
	}

	return c.prepareEvent(room.ID, &content, rel), nil
}

func (c *Container) PrepareMarkdownMessage(roomID id.RoomID, msgtype event.MessageType, text, html string, rel *ifc.Relation) *muksevt.Event {
	var content event.MessageEventContent
	if html != "" {
		content = event.MessageEventContent{
			FormattedBody: html,
			Format:        event.FormatHTML,
			Body:          text,
			MsgType:       msgtype,
		}
	} else {
		content = format.RenderMarkdown(text, !c.config.Preferences.DisableMarkdown, !c.config.Preferences.DisableHTML)
		content.MsgType = msgtype
	}

	return c.prepareEvent(roomID, &content, rel)
}

func (c *Container) prepareEvent(roomID id.RoomID, content *event.MessageEventContent, rel *ifc.Relation) *muksevt.Event {
	if rel != nil && rel.Type == event.RelReplace {
		contentCopy := *content
		content.NewContent = &contentCopy
		content.Body = "* " + content.Body
		if len(content.FormattedBody) > 0 {
			content.FormattedBody = "* " + content.FormattedBody
		}
		content.RelatesTo = &event.RelatesTo{
			Type:    event.RelReplace,
			EventID: rel.Event.ID,
		}
	} else if rel != nil && rel.Type == event.RelReply {
		content.SetReply(rel.Event.Event)
	}

	txnID := c.client.TxnID()
	localEcho := muksevt.Wrap(&event.Event{
		ID:        id.EventID(txnID),
		Sender:    c.config.UserID,
		Type:      event.EventMessage,
		Timestamp: time.Now().UnixNano() / 1e6,
		RoomID:    roomID,
		Content:   event.Content{Parsed: content},
		Unsigned:  event.Unsigned{TransactionID: txnID},
	})
	localEcho.Gomuks.OutgoingState = muksevt.StateLocalEcho
	if rel != nil && rel.Type == event.RelReplace {
		localEcho.ID = rel.Event.ID
		localEcho.Gomuks.Edits = []*muksevt.Event{localEcho}
	}
	return localEcho
}

func (c *Container) Redact(roomID id.RoomID, eventID id.EventID, reason string) error {
	defer debug.Recover()
	_, err := c.client.RedactEvent(roomID, eventID, mautrix.ReqRedact{Reason: reason})
	return err
}

// SendMessage sends the given event.
func (c *Container) SendEvent(evt *muksevt.Event) (id.EventID, error) {
	defer debug.Recover()

	_, _ = c.client.UserTyping(evt.RoomID, false, 0)
	c.typing = 0
	room := c.GetRoom(evt.RoomID)
	if room != nil && room.Encrypted && c.crypto != nil && evt.Type != event.EventReaction {
		encrypted, err := c.crypto.EncryptMegolmEvent(evt.RoomID, evt.Type, &evt.Content)
		if err != nil {
			if isBadEncryptError(err) {
				return "", err
			}
			debug.Print("Got", err, "while trying to encrypt message, sharing group session and trying again...")
			err = c.crypto.ShareGroupSession(room.ID, room.GetMemberList())
			if err != nil {
				return "", err
			}
			encrypted, err = c.crypto.EncryptMegolmEvent(evt.RoomID, evt.Type, &evt.Content)
			if err != nil {
				return "", err
			}
		}
		evt.Type = event.EventEncrypted
		evt.Content = event.Content{Parsed: encrypted}
	}
	resp, err := c.client.SendMessageEvent(evt.RoomID, evt.Type, &evt.Content, mautrix.ReqSendEvent{TransactionID: evt.Unsigned.TransactionID})
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

func (c *Container) UploadMedia(path string, encrypt bool) (*ifc.UploadedMediaInfo, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	msgtype, info, err := getMediaInfo(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	uploadFileName := stat.Name()
	uploadMimeType := info.MimeType

	var content io.Reader
	var encryptionInfo *attachment.EncryptedFile
	if encrypt {
		uploadMimeType = "application/octet-stream"
		uploadFileName = ""
		encryptionInfo = attachment.NewEncryptedFile()
		content = encryptionInfo.EncryptStream(file)
	} else {
		content = file
	}

	resp, err := c.client.UploadMedia(mautrix.ReqUploadMedia{
		Content:       content,
		ContentLength: stat.Size(),
		ContentType:   uploadMimeType,
		FileName:      uploadFileName,
	})

	if err != nil {
		return nil, err
	}

	return &ifc.UploadedMediaInfo{
		RespMediaUpload: resp,
		EncryptionInfo:  encryptionInfo,
		Name:            stat.Name(),
		MsgType:         msgtype,
		Info:            &info,
	}, nil
}

func (c *Container) sendTypingAsync(roomID id.RoomID, typing bool, timeout int64) {
	defer debug.Recover()
	_, _ = c.client.UserTyping(roomID, typing, timeout)
}

// SendTyping sets whether or not the user is typing in the given room.
func (c *Container) SendTyping(roomID id.RoomID, typing bool) {
	ts := time.Now().Unix()
	if (c.typing > ts && typing) || (c.typing == 0 && !typing) {
		return
	}

	if typing {
		go c.sendTypingAsync(roomID, true, 20000)
		c.typing = ts + 15
	} else {
		go c.sendTypingAsync(roomID, false, 0)
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
func (c *Container) JoinRoom(roomID id.RoomID, server string) (*rooms.Room, error) {
	resp, err := c.client.JoinRoom(string(roomID), server, nil)
	if err != nil {
		return nil, err
	}

	room := c.GetOrCreateRoom(resp.RoomID)
	room.HasLeft = false
	return room, nil
}

// LeaveRoom makes the current user leave the given room.
func (c *Container) LeaveRoom(roomID id.RoomID) error {
	_, err := c.client.LeaveRoom(roomID)
	if err != nil {
		return err
	}

	node := c.GetOrCreateRoom(roomID)
	node.HasLeft = true
	node.Unload()
	return nil
}

func (c *Container) FetchMembers(room *rooms.Room) error {
	debug.Print("Fetching member list for", room.ID)
	members, err := c.client.Members(room.ID, mautrix.ReqMembers{At: room.LastPrevBatch})
	if err != nil {
		return err
	}
	debug.Printf("Fetched %d members for %s", len(members.Chunk), room.ID)
	for _, evt := range members.Chunk {
		err := evt.Content.ParseRaw(evt.Type)
		if err != nil {
			debug.Printf("Failed to parse member event of %s: %v", evt.GetStateKey(), err)
			continue
		}
		room.UpdateState(evt)
	}
	room.MembersFetched = true
	return nil
}

// GetHistory fetches room history.
func (c *Container) GetHistory(room *rooms.Room, limit int, dbPointer uint64) ([]*muksevt.Event, uint64, error) {
	events, newDBPointer, err := c.history.Load(room, limit, dbPointer)
	if err != nil {
		return nil, dbPointer, err
	}
	if len(events) > 0 {
		debug.Printf("Loaded %d events for %s from local cache", len(events), room.ID)
		return events, newDBPointer, nil
	}
	resp, err := c.client.Messages(room.ID, room.PrevBatch, "", 'b', nil, limit)
	if err != nil {
		return nil, dbPointer, err
	}
	debug.Printf("Loaded %d events for %s from server from %s to %s", len(resp.Chunk), room.ID, resp.Start, resp.End)
	for i, evt := range resp.Chunk {
		err := evt.Content.ParseRaw(evt.Type)
		if err != nil {
			debug.Printf("Failed to unmarshal content of event %s (type %s) by %s in %s: %v\n%s", evt.ID, evt.Type.Repr(), evt.Sender, evt.RoomID, err, string(evt.Content.VeryRaw))
		}

		if evt.Type == event.EventEncrypted {
			if c.crypto == nil {
				evt.Type = muksevt.EventEncryptionUnsupported
				origContent, _ := evt.Content.Parsed.(*event.EncryptedEventContent)
				evt.Content.Parsed = muksevt.EncryptionUnsupportedContent{Original: origContent}
			} else {
				decrypted, err := c.crypto.DecryptMegolmEvent(evt)
				if err != nil {
					debug.Printf("Failed to decrypt event %s: %v", evt.ID, err)
					evt.Type = muksevt.EventBadEncrypted
					origContent, _ := evt.Content.Parsed.(*event.EncryptedEventContent)
					evt.Content.Parsed = &muksevt.BadEncryptedContent{
						Original: origContent,
						Reason:   err.Error(),
					}
				} else {
					resp.Chunk[i] = decrypted
				}
			}
		}
	}
	for _, evt := range resp.State {
		room.UpdateState(evt)
	}
	room.PrevBatch = resp.End
	c.config.Rooms.Put(room)
	if len(resp.Chunk) == 0 {
		return []*muksevt.Event{}, dbPointer, nil
	}
	// TODO newDBPointer isn't accurate in this case yet, fix later
	events, newDBPointer, err = c.history.Prepend(room, resp.Chunk)
	if err != nil {
		return nil, dbPointer, err
	}
	return events, dbPointer, nil
}

func (c *Container) GetEvent(room *rooms.Room, eventID id.EventID) (*muksevt.Event, error) {
	evt, err := c.history.Get(room, eventID)
	if err != nil && err != EventNotFoundError {
		debug.Printf("Failed to get event %s from local cache: %v", eventID, err)
	} else if evt != nil {
		debug.Printf("Found event %s in local cache", eventID)
		return evt, err
	}
	mxEvent, err := c.client.GetEvent(room.ID, eventID)
	if err != nil {
		return nil, err
	}
	err = mxEvent.Content.ParseRaw(mxEvent.Type)
	if err != nil {
		return nil, err
	}
	debug.Printf("Loaded event %s from server", eventID)
	return muksevt.Wrap(mxEvent), nil
}

// GetOrCreateRoom gets the room instance stored in the session.
func (c *Container) GetOrCreateRoom(roomID id.RoomID) *rooms.Room {
	return c.config.Rooms.GetOrCreate(roomID)
}

// GetRoom gets the room instance stored in the session.
func (c *Container) GetRoom(roomID id.RoomID) *rooms.Room {
	return c.config.Rooms.Get(roomID)
}

func cp(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func (c *Container) DownloadToDisk(uri id.ContentURI, file *attachment.EncryptedFile, target string) (fullPath string, err error) {
	cachePath := c.GetCachePath(uri)
	if target == "" {
		fullPath = cachePath
	} else if !path.IsAbs(target) {
		fullPath = path.Join(c.config.DownloadDir, target)
	} else {
		fullPath = target
	}

	if _, statErr := os.Stat(cachePath); os.IsNotExist(statErr) {
		var body io.ReadCloser
		body, err = c.client.Download(uri)
		if err != nil {
			return
		}

		var data []byte
		data, err = ioutil.ReadAll(body)
		_ = body.Close()
		if err != nil {
			return
		}

		if file != nil {
			err = file.DecryptInPlace(data)
			if err != nil {
				return
			}
		}

		err = ioutil.WriteFile(cachePath, data, 0600)
		if err != nil {
			return
		}
	}

	if fullPath != cachePath {
		err = os.MkdirAll(path.Dir(fullPath), 0700)
		if err != nil {
			return
		}
		err = cp(cachePath, fullPath)
	}

	return
}

// Download fetches the given Matrix content (mxc) URL and returns the data, homeserver, file ID and potential errors.
//
// The file will be either read from the media cache (if found) or downloaded from the server.
func (c *Container) Download(uri id.ContentURI, file *attachment.EncryptedFile) (data []byte, err error) {
	cacheFile := c.GetCachePath(uri)
	var info os.FileInfo
	if info, err = os.Stat(cacheFile); err == nil && !info.IsDir() {
		data, err = ioutil.ReadFile(cacheFile)
		if err == nil {
			return
		}
	}

	data, err = c.download(uri, file, cacheFile)
	return
}

func (c *Container) GetDownloadURL(uri id.ContentURI) string {
	return c.client.GetDownloadURL(uri)
}

func (c *Container) download(uri id.ContentURI, file *attachment.EncryptedFile, cacheFile string) (data []byte, err error) {
	var body io.ReadCloser
	body, err = c.client.Download(uri)
	if err != nil {
		return
	}

	data, err = ioutil.ReadAll(body)
	_ = body.Close()
	if err != nil {
		return
	}

	if file != nil {
		err = file.DecryptInPlace(data)
		if err != nil {
			return
		}
	}

	err = ioutil.WriteFile(cacheFile, data, 0600)
	return
}

// GetCachePath gets the path to the cached version of the given homeserver:fileID combination.
// The file may or may not exist, use Download() to ensure it has been cached.
func (c *Container) GetCachePath(uri id.ContentURI) string {
	dir := filepath.Join(c.config.MediaDir, uri.Homeserver)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return ""
	}

	return filepath.Join(dir, uri.FileID)
}
