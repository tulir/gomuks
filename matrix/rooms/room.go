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

package rooms

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"encoding/gob"
	"maunium.net/go/mautrix"
	"maunium.net/go/gomuks/debug"
	"os"
)

func init() {
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

type RoomNameSource int

const (
	ExplicitRoomName RoomNameSource = iota
	CanonicalAliasRoomName
	AliasRoomName
	MemberRoomName
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
	*mautrix.Room

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
	// Whether or not this room is marked as a direct chat.
	IsDirect bool

	// List of tags given to this room
	RawTags []RoomTag
	// Timestamp of previously received actual message.
	LastReceivedMessage time.Time

	// MXID -> Member cache calculated from membership events.
	memberCache map[string]*mautrix.Member
	// The first non-SessionUserID member in the room. Calculated at
	// the same time as memberCache.
	firstMemberCache *mautrix.Member
	// The name of the room. Calculated from the state event name,
	// canonical_alias or alias or the member cache.
	nameCache string
	// The event type from which the name cache was calculated from.
	nameCacheSource RoomNameSource
	// The topic of the room. Directly fetched from the m.room.topic state event.
	topicCache string
	// The canonical alias of the room. Directly fetched from the m.room.canonical_alias state event.
	canonicalAliasCache string
	// The list of aliases. Directly fetched from the m.room.aliases state event.
	aliasesCache []string

	// fetchHistoryLock is used to make sure multiple goroutines don't fetch
	// history for this room at the same time.
	fetchHistoryLock *sync.Mutex
}

// LockHistory locks the history fetching mutex.
// If the mutex is nil, it will be created.
func (room *Room) LockHistory() {
	if room.fetchHistoryLock == nil {
		room.fetchHistoryLock = &sync.Mutex{}
	}
	room.fetchHistoryLock.Lock()
}

// UnlockHistory unlocks the history fetching mutex.
// If the mutex is nil, this does nothing.
func (room *Room) UnlockHistory() {
	if room.fetchHistoryLock != nil {
		room.fetchHistoryLock.Unlock()
	}
}

func (room *Room) Load(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	dec := gob.NewDecoder(file)
	return dec.Decode(room)
}

func (room *Room) Save(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)
	return enc.Encode(room)
}

// MarkRead clears the new message statuses on this room.
func (room *Room) MarkRead(eventID string) {
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
}

func (room *Room) UnreadCount() int {
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
	_, exists := room.State[event.Type]
	if !exists {
		room.State[event.Type] = make(map[string]*mautrix.Event)
	}
	switch event.Type {
	case mautrix.StateRoomName:
		room.nameCache = ""
	case mautrix.StateCanonicalAlias:
		if room.nameCacheSource >= CanonicalAliasRoomName {
			room.nameCache = ""
		}
		room.canonicalAliasCache = ""
	case mautrix.StateAliases:
		if room.nameCacheSource >= AliasRoomName {
			room.nameCache = ""
		}
		room.aliasesCache = nil
	case mautrix.StateMember:
		room.memberCache = nil
		room.firstMemberCache = nil
		if room.nameCacheSource >= MemberRoomName {
			room.nameCache = ""
		}
	case mautrix.StateTopic:
		room.topicCache = ""
	}

	stateKey := ""
	if event.StateKey != nil {
		stateKey = *event.StateKey
	}
	if event.Type != mautrix.StateMember {
		debug.Printf("Updating state %s#%s for %s", event.Type, stateKey, room.ID)
	}

	if event.StateKey == nil {
		room.State[event.Type][""] = event
	} else {
		room.State[event.Type][*event.StateKey] = event
	}
}

// GetStateEvent returns the state event for the given type/state_key combo, or nil.
func (room *Room) GetStateEvent(eventType mautrix.EventType, stateKey string) *mautrix.Event {
	stateEventMap, _ := room.State[eventType]
	event, _ := stateEventMap[stateKey]
	return event
}

// GetStateEvents returns the state events for the given type.
func (room *Room) GetStateEvents(eventType mautrix.EventType) map[string]*mautrix.Event {
	stateEventMap, _ := room.State[eventType]
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
		aliasEvents := room.GetStateEvents(mautrix.StateAliases)
		room.aliasesCache = []string{}
		for _, event := range aliasEvents {
			room.aliasesCache = append(room.aliasesCache, event.Content.Aliases...)
		}
	}
	return room.aliasesCache
}

// updateNameFromNameEvent updates the room display name to be the name set in the name event.
func (room *Room) updateNameFromNameEvent() {
	nameEvt := room.GetStateEvent(mautrix.StateRoomName, "")
	if nameEvt != nil {
		room.nameCache = nameEvt.Content.Name
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
		room.nameCache = aliases[0]
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
		room.nameCache = "Empty room"
	} else if room.firstMemberCache == nil {
		room.nameCache = "Room"
	} else if len(members) == 2 {
		room.nameCache = room.firstMemberCache.Displayname
	} else {
		firstMember := room.firstMemberCache.Displayname
		room.nameCache = fmt.Sprintf("%s and %d others", firstMember, len(members)-2)
	}
}

// updateNameCache updates the room display name based on the room state in the order
// specified in spec section 11.2.2.5.
func (room *Room) updateNameCache() {
	if len(room.nameCache) == 0 {
		room.updateNameFromNameEvent()
		room.nameCacheSource = ExplicitRoomName
	}
	if len(room.nameCache) == 0 {
		room.nameCache = room.GetCanonicalAlias()
		room.nameCacheSource = CanonicalAliasRoomName
	}
	if len(room.nameCache) == 0 {
		room.updateNameFromAliases()
		room.nameCacheSource = AliasRoomName
	}
	if len(room.nameCache) == 0 {
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
	return room.nameCache
}

// createMemberCache caches all member events into a easily processable MXID -> *Member map.
func (room *Room) createMemberCache() map[string]*mautrix.Member {
	cache := make(map[string]*mautrix.Member)
	events := room.GetStateEvents(mautrix.StateMember)
	room.firstMemberCache = nil
	if events != nil {
		for userID, event := range events {
			member := &event.Content.Member
			if room.firstMemberCache == nil && userID != room.SessionUserID {
				room.firstMemberCache = member
			}
			if member.Membership != "leave" {
				cache[userID] = member
			}
		}
	}
	room.memberCache = cache
	return cache
}

// GetMembers returns the members in this room.
//
// The members are returned from the cache.
// If the cache is empty, it is updated first.
func (room *Room) GetMembers() map[string]*mautrix.Member {
	if len(room.memberCache) == 0 || room.firstMemberCache == nil {
		room.createMemberCache()
	}
	return room.memberCache
}

// GetMember returns the member with the given MXID.
// If the member doesn't exist, nil is returned.
func (room *Room) GetMember(userID string) *mautrix.Member {
	if len(room.memberCache) == 0 {
		room.createMemberCache()
	}
	member, _ := room.memberCache[userID]
	return member
}

// GetSessionOwner returns the Member instance of the user whose session this room was created for.
func (room *Room) GetSessionOwner() string {
	return room.SessionUserID
}

// NewRoom creates a new Room with the given ID
func NewRoom(roomID, owner string) *Room {
	return &Room{
		Room:             mautrix.NewRoom(roomID),
		fetchHistoryLock: &sync.Mutex{},
		SessionUserID:    owner,
	}
}
