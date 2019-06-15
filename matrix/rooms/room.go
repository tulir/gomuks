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

package rooms

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"time"

	sync "github.com/sasha-s/go-deadlock"

	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/debug"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(&Room{})
	gob.Register(0)
}

type RoomNameSource int

const (
	UnknownRoomName RoomNameSource = iota
	MemberRoomName
	AliasRoomName
	CanonicalAliasRoomName
	ExplicitRoomName
)

// RoomTag is a tag given to a specific room.
type RoomTag struct {
	// The name of the tag.
	Tag string
	// The order of the tag.
	Order string
}

type UnreadMessage struct {
	EventID   string
	Counted   bool
	Highlight bool
}

// Room represents a single Matrix room.
type Room struct {
	// The room ID.
	ID string

	// Whether or not the user has left the room.
	HasLeft bool

	// The first batch of events that has been fetched for this room.
	// Used for fetching additional history.
	PrevBatch string
	// The MXID of the user whose session this room was created for.
	SessionUserID string

	// The number of unread messages that were notified about.
	UnreadMessages   []UnreadMessage
	unreadCountCache *int
	highlightCache   *bool
	lastMarkedRead   string
	// Whether or not this room is marked as a direct chat.
	IsDirect bool

	// List of tags given to this room.
	RawTags []RoomTag
	// Timestamp of previously received actual message.
	LastReceivedMessage time.Time

	// Room state cache.
	state map[mautrix.EventType]map[string]*mautrix.Event
	// MXID -> Member cache calculated from membership events.
	memberCache map[string]*mautrix.Member
	// The first two non-SessionUserID members in the room. Calculated at
	// the same time as memberCache.
	firstMemberCache  *mautrix.Member
	secondMemberCache *mautrix.Member
	// The name of the room. Calculated from the state event name,
	// canonical_alias or alias or the member cache.
	NameCache string
	// The event type from which the name cache was calculated from.
	nameCacheSource RoomNameSource
	// The topic of the room. Directly fetched from the m.room.topic state event.
	topicCache string
	// The canonical alias of the room. Directly fetched from the m.room.canonical_alias state event.
	canonicalAliasCache string
	// The list of aliases. Directly fetched from the m.room.aliases state event.
	aliasesCache []string

	// Path for state store file.
	path string
	// Room cache object
	cache *RoomCache
	// Lock for state and other room stuff.
	lock sync.RWMutex
	// Function to call when room state is unloaded.
	onUnload func() bool
	// Function to call when room state is loaded.
	onLoad func() bool
	// Whether or not the room state has changed
	changed bool

	// Room state cache linked list.
	prev  *Room
	next  *Room
	touch int64
}

func debugPrintError(fn func() error, message string) {
	if err := fn(); err != nil {
		debug.Printf("%s: %v", message, err)
	}
}

func (room *Room) Loaded() bool {
	return room.state != nil
}

func (room *Room) Load() {
	room.cache.TouchNode(room)
	if room.Loaded() {
		return
	}
	if room.onLoad != nil && !room.onLoad() {
		return
	}
	room.lock.Lock()
	room.load()
	room.lock.Unlock()
}

func (room *Room) load() {
	if room.Loaded() {
		return
	}
	debug.Print("Loading state for room", room.ID, "from disk")
	room.state = make(map[mautrix.EventType]map[string]*mautrix.Event)
	file, err := os.OpenFile(room.path, os.O_RDONLY, 0600)
	if err != nil {
		if !os.IsNotExist(err) {
			debug.Print("Failed to open room state file for reading:", err)
		} else {
			debug.Print("Room state file for", room.ID, "does not exist")
		}
		return
	}
	defer debugPrintError(file.Close, "Failed to close room state file after reading")
	cmpReader, err := gzip.NewReader(file)
	if err != nil {
		debug.Print("Failed to open room state gzip reader:", err)
		return
	}
	defer debugPrintError(cmpReader.Close, "Failed to close room state gzip reader")
	dec := gob.NewDecoder(cmpReader)
	if err = dec.Decode(&room.state); err != nil {
		debug.Print("Failed to decode room state:", err)
	}
	room.changed = false
}

func (room *Room) Touch() {
	room.cache.TouchNode(room)
}

func (room *Room) Unload() bool {
	if room.onUnload != nil && !room.onUnload() {
		return false
	}
	debug.Print("Unloading", room.ID)
	room.Save()
	room.state = nil
	room.aliasesCache = nil
	room.topicCache = ""
	room.canonicalAliasCache = ""
	room.firstMemberCache = nil
	room.secondMemberCache = nil
	return true
}

func (room *Room) SetOnUnload(fn func() bool) {
	room.onUnload = fn
}

func (room *Room) SetOnLoad(fn func() bool) {
	room.onLoad = fn
}

func (room *Room) Save() {
	if !room.Loaded() {
		debug.Print("Failed to save room", room.ID, "state: room not loaded")
		return
	}
	if !room.changed {
		debug.Print("Not saving", room.ID, "as state hasn't changed")
		return
	}
	debug.Print("Saving state for room", room.ID, "to disk")
	file, err := os.OpenFile(room.path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		debug.Print("Failed to open room state file for writing:", err)
		return
	}
	defer debugPrintError(file.Close, "Failed to close room state file after writing")
	cmpWriter := gzip.NewWriter(file)
	defer debugPrintError(cmpWriter.Close, "Failed to close room state gzip writer")
	enc := gob.NewEncoder(cmpWriter)
	room.lock.RLock()
	defer room.lock.RUnlock()
	if err := enc.Encode(&room.state); err != nil {
		debug.Print("Failed to encode room state:", err)
	}
}

// MarkRead clears the new message statuses on this room.
func (room *Room) MarkRead(eventID string) bool {
	room.lock.Lock()
	defer room.lock.Unlock()
	if room.lastMarkedRead == eventID {
		return false
	}
	room.lastMarkedRead = eventID
	readToIndex := -1
	for index, unreadMessage := range room.UnreadMessages {
		if unreadMessage.EventID == eventID {
			readToIndex = index
		}
	}
	if readToIndex >= 0 {
		room.UnreadMessages = room.UnreadMessages[readToIndex+1:]
		room.highlightCache = nil
		room.unreadCountCache = nil
	}
	return true
}

func (room *Room) UnreadCount() int {
	room.lock.Lock()
	defer room.lock.Unlock()
	if room.unreadCountCache == nil {
		room.unreadCountCache = new(int)
		for _, unreadMessage := range room.UnreadMessages {
			if unreadMessage.Counted {
				*room.unreadCountCache++
			}
		}
	}
	return *room.unreadCountCache
}

func (room *Room) Highlighted() bool {
	room.lock.Lock()
	defer room.lock.Unlock()
	if room.highlightCache == nil {
		room.highlightCache = new(bool)
		for _, unreadMessage := range room.UnreadMessages {
			if unreadMessage.Highlight {
				*room.highlightCache = true
				break
			}
		}
	}
	return *room.highlightCache
}

func (room *Room) HasNewMessages() bool {
	return len(room.UnreadMessages) > 0
}

func (room *Room) AddUnread(eventID string, counted, highlight bool) {
	room.lock.Lock()
	defer room.lock.Unlock()
	room.UnreadMessages = append(room.UnreadMessages, UnreadMessage{
		EventID:   eventID,
		Counted:   counted,
		Highlight: highlight,
	})
	if counted {
		if room.unreadCountCache == nil {
			room.unreadCountCache = new(int)
		}
		*room.unreadCountCache++
	}
	if highlight {
		if room.highlightCache == nil {
			room.highlightCache = new(bool)
		}
		*room.highlightCache = true
	}
}

func (room *Room) Tags() []RoomTag {
	room.lock.RLock()
	defer room.lock.RUnlock()
	if len(room.RawTags) == 0 {
		if room.IsDirect {
			return []RoomTag{{"net.maunium.gomuks.fake.direct", "0.5"}}
		}
		return []RoomTag{{"", "0.5"}}
	}
	return room.RawTags
}

// UpdateState updates the room's current state with the given Event. This will clobber events based
// on the type/state_key combination.
func (room *Room) UpdateState(event *mautrix.Event) {
	room.Load()
	room.lock.Lock()
	defer room.lock.Unlock()
	room.changed = true
	_, exists := room.state[event.Type]
	if !exists {
		room.state[event.Type] = make(map[string]*mautrix.Event)
	}
	switch event.Type {
	case mautrix.StateRoomName:
		room.NameCache = event.Content.Name
		room.nameCacheSource = ExplicitRoomName
	case mautrix.StateCanonicalAlias:
		if room.nameCacheSource <= CanonicalAliasRoomName {
			room.NameCache = event.Content.Alias
			room.nameCacheSource = CanonicalAliasRoomName
		}
		room.canonicalAliasCache = event.Content.Alias
	case mautrix.StateAliases:
		if room.nameCacheSource <= AliasRoomName {
			room.NameCache = ""
		}
		room.aliasesCache = nil
	case mautrix.StateMember:
		if room.memberCache != nil {
			userID := event.GetStateKey()
			if event.Content.Membership == mautrix.MembershipLeave || event.Content.Membership == mautrix.MembershipBan {
				delete(room.memberCache, userID)
			} else if event.Content.Membership == mautrix.MembershipInvite || event.Content.Membership == mautrix.MembershipJoin {
				member := room.eventToMember(userID, &event.Content)
				existingMember, ok := room.memberCache[userID]
				if ok {
					*existingMember = *member
				} else {
					room.memberCache[userID] = member
					room.updateNthMemberCache(userID, member)
				}
			}
		}
		if room.nameCacheSource <= MemberRoomName {
			room.NameCache = ""
		}
	case mautrix.StateTopic:
		room.topicCache = event.Content.Topic
	}

	if event.Type != mautrix.StateMember {
		debug.Printf("Updating state %s#%s for %s", event.Type.String(), event.GetStateKey(), room.ID)
	}

	if event.StateKey == nil {
		room.state[event.Type][""] = event
	} else {
		room.state[event.Type][*event.StateKey] = event
	}
}

// GetStateEvent returns the state event for the given type/state_key combo, or nil.
func (room *Room) GetStateEvent(eventType mautrix.EventType, stateKey string) *mautrix.Event {
	room.Load()
	room.lock.RLock()
	defer room.lock.RUnlock()
	stateEventMap, _ := room.state[eventType]
	event, _ := stateEventMap[stateKey]
	return event
}

// getStateEvents returns the state events for the given type.
func (room *Room) getStateEvents(eventType mautrix.EventType) map[string]*mautrix.Event {
	stateEventMap, _ := room.state[eventType]
	return stateEventMap
}

// GetTopic returns the topic of the room.
func (room *Room) GetTopic() string {
	if len(room.topicCache) == 0 {
		topicEvt := room.GetStateEvent(mautrix.StateTopic, "")
		if topicEvt != nil {
			room.topicCache = topicEvt.Content.Topic
		}
	}
	return room.topicCache
}

func (room *Room) GetCanonicalAlias() string {
	if len(room.canonicalAliasCache) == 0 {
		canonicalAliasEvt := room.GetStateEvent(mautrix.StateCanonicalAlias, "")
		if canonicalAliasEvt != nil {
			room.canonicalAliasCache = canonicalAliasEvt.Content.Alias
		} else {
			room.canonicalAliasCache = "-"
		}
	}
	if room.canonicalAliasCache == "-" {
		return ""
	}
	return room.canonicalAliasCache
}

// GetAliases returns the list of aliases that point to this room.
func (room *Room) GetAliases() []string {
	if room.aliasesCache == nil {
		room.lock.RLock()
		aliasEvents := room.getStateEvents(mautrix.StateAliases)
		room.aliasesCache = []string{}
		for _, event := range aliasEvents {
			room.aliasesCache = append(room.aliasesCache, event.Content.Aliases...)
		}
		room.lock.RUnlock()
	}
	return room.aliasesCache
}

// updateNameFromNameEvent updates the room display name to be the name set in the name event.
func (room *Room) updateNameFromNameEvent() {
	nameEvt := room.GetStateEvent(mautrix.StateRoomName, "")
	if nameEvt != nil {
		room.NameCache = nameEvt.Content.Name
	}
}

// updateNameFromAliases updates the room display name to be the first room alias it finds.
//
// Deprecated: the Client-Server API recommends against using non-canonical aliases as display name.
func (room *Room) updateNameFromAliases() {
	// TODO the spec says clients should not use m.room.aliases for room names.
	//      However, Riot also uses m.room.aliases, so this is here now.
	aliases := room.GetAliases()
	if len(aliases) > 0 {
		sort.Sort(sort.StringSlice(aliases))
		room.NameCache = aliases[0]
	}
}

// updateNameFromMembers updates the room display name based on the members in this room.
//
// The room name depends on the number of users:
//  Less than two users -> "Empty room"
//  Exactly two users   -> The display name of the other user.
//  More than two users -> The display name of one of the other users, followed
//                         by "and X others", where X is the number of users
//                         excluding the local user and the named user.
func (room *Room) updateNameFromMembers() {
	members := room.GetMembers()
	if len(members) <= 1 {
		room.NameCache = "Empty room"
	} else if room.firstMemberCache == nil {
		room.NameCache = "Room"
	} else if len(members) == 2 {
		room.NameCache = room.firstMemberCache.Displayname
	} else if len(members) == 3 && room.secondMemberCache != nil {
		room.NameCache = fmt.Sprintf("%s and %s", room.firstMemberCache.Displayname, room.secondMemberCache.Displayname)
	} else {
		members := room.firstMemberCache.Displayname
		count := len(members) - 2
		if room.secondMemberCache != nil {
			members += ", " + room.secondMemberCache.Displayname
			count--
		}
		room.NameCache = fmt.Sprintf("%s and %d others", members, count)
	}
}

// updateNameCache updates the room display name based on the room state in the order
// specified in spec section 11.2.2.5.
func (room *Room) updateNameCache() {
	if len(room.NameCache) == 0 {
		room.updateNameFromNameEvent()
		room.nameCacheSource = ExplicitRoomName
	}
	if len(room.NameCache) == 0 {
		room.NameCache = room.GetCanonicalAlias()
		room.nameCacheSource = CanonicalAliasRoomName
	}
	if len(room.NameCache) == 0 {
		room.updateNameFromAliases()
		room.nameCacheSource = AliasRoomName
	}
	if len(room.NameCache) == 0 {
		room.updateNameFromMembers()
		room.nameCacheSource = MemberRoomName
	}
}

// GetTitle returns the display name of the room.
//
// The display name is returned from the cache.
// If the cache is empty, it is updated first.
func (room *Room) GetTitle() string {
	room.updateNameCache()
	return room.NameCache
}

func (room *Room) eventToMember(userID string, content *mautrix.Content) *mautrix.Member {
	member := &content.Member
	member.Membership = content.Membership
	if len(member.Displayname) == 0 {
		member.Displayname = userID
	}
	return member
}

func (room *Room) updateNthMemberCache(userID string, member *mautrix.Member) {
	if userID != room.SessionUserID {
		if room.firstMemberCache == nil {
			room.firstMemberCache = member
		} else if room.secondMemberCache == nil {
			room.secondMemberCache = member
		}
	}
}

// createMemberCache caches all member events into a easily processable MXID -> *Member map.
func (room *Room) createMemberCache() map[string]*mautrix.Member {
	if len(room.memberCache) > 0 {
		return room.memberCache
	}
	cache := make(map[string]*mautrix.Member)
	room.lock.RLock()
	events := room.getStateEvents(mautrix.StateMember)
	room.firstMemberCache = nil
	room.secondMemberCache = nil
	if events != nil {
		for userID, event := range events {
			member := room.eventToMember(userID, &event.Content)
			if member.Membership == mautrix.MembershipJoin || member.Membership == mautrix.MembershipInvite {
				cache[userID] = member
				room.updateNthMemberCache(userID, member)
			}
		}
	}
	room.lock.RUnlock()
	room.lock.Lock()
	room.memberCache = cache
	room.lock.Unlock()
	return cache
}

// GetMembers returns the members in this room.
//
// The members are returned from the cache.
// If the cache is empty, it is updated first.
func (room *Room) GetMembers() map[string]*mautrix.Member {
	room.Load()
	room.createMemberCache()
	return room.memberCache
}

// GetMember returns the member with the given MXID.
// If the member doesn't exist, nil is returned.
func (room *Room) GetMember(userID string) *mautrix.Member {
	room.Load()
	room.createMemberCache()
	room.lock.RLock()
	member, _ := room.memberCache[userID]
	room.lock.RUnlock()
	return member
}

// GetSessionOwner returns the ID of the user whose session this room was created for.
func (room *Room) GetSessionOwner() string {
	return room.SessionUserID
}

// NewRoom creates a new Room with the given ID
func NewRoom(roomID string, cache *RoomCache) *Room {
	return &Room{
		ID:    roomID,
		state: make(map[mautrix.EventType]map[string]*mautrix.Event),
		path:  cache.roomPath(roomID),
		cache: cache,

		SessionUserID: cache.getOwner(),
	}
}
