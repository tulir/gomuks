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

package room

import (
	"fmt"

	"maunium.net/go/gomatrix"
)

// Room represents a single Matrix room.
type Room struct {
	*gomatrix.Room

	PrevBatch        string
	Owner string
	memberCache      map[string]*Member
	firstMemberCache string
	nameCache        string
	topicCache       string
}

// UpdateState updates the room's current state with the given Event. This will clobber events based
// on the type/state_key combination.
func (room *Room) UpdateState(event *gomatrix.Event) {
	_, exists := room.State[event.Type]
	if !exists {
		room.State[event.Type] = make(map[string]*gomatrix.Event)
	}
	switch event.Type {
	case "m.room.member":
		room.memberCache = nil
		room.firstMemberCache = ""
		fallthrough
	case "m.room.name":
		fallthrough
	case "m.room.canonical_alias":
		fallthrough
	case "m.room.alias":
		room.nameCache = ""
	case "m.room.topic":
		room.topicCache = ""
	}
	room.State[event.Type][*event.StateKey] = event
}

// GetStateEvent returns the state event for the given type/state_key combo, or nil.
func (room *Room) GetStateEvent(eventType string, stateKey string) *gomatrix.Event {
	stateEventMap, _ := room.State[eventType]
	event, _ := stateEventMap[stateKey]
	return event
}

// GetStateEvents returns the state events for the given type.
func (room *Room) GetStateEvents(eventType string) map[string]*gomatrix.Event {
	stateEventMap, _ := room.State[eventType]
	return stateEventMap
}

// GetTopic returns the topic of the room.
func (room *Room) GetTopic() string {
	if len(room.topicCache) == 0 {
		topicEvt := room.GetStateEvent("m.room.topic", "")
		if topicEvt != nil {
			room.topicCache, _ = topicEvt.Content["topic"].(string)
		}
	}
	return room.topicCache
}

// updateNameFromNameEvent updates the room display name to be the name set in the name event.
func (room *Room) updateNameFromNameEvent() {
	nameEvt := room.GetStateEvent("m.room.name", "")
	if nameEvt != nil {
		room.nameCache, _ = nameEvt.Content["name"].(string)
	}
}

// updateNameFromCanonicalAlias updates the room display name to be the canonical alias of the room.
func (room *Room) updateNameFromCanonicalAlias() {
	canonicalAliasEvt := room.GetStateEvent("m.room.canonical_alias", "")
	if canonicalAliasEvt != nil {
		room.nameCache, _ = canonicalAliasEvt.Content["alias"].(string)
	}
}

// updateNameFromAliases updates the room display name to be the first room alias it finds.
//
// Deprecated: the Client-Server API recommends against using aliases as display name.
func (room *Room) updateNameFromAliases() {
	// TODO the spec says clients should not use m.room.aliases for room names.
	//      However, Riot also uses m.room.aliases, so this is here now.
	aliasEvents := room.GetStateEvents("m.room.aliases")
	for _, event := range aliasEvents {
		aliases, _ := event.Content["aliases"].([]interface{})
		if len(aliases) > 0 {
			room.nameCache, _ = aliases[0].(string)
			break
		}
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
	} else if len(members) == 2 {
		room.nameCache = members[room.firstMemberCache].DisplayName
	} else {
		firstMember := members[room.firstMemberCache].DisplayName
		room.nameCache = fmt.Sprintf("%s and %d others", firstMember, len(members)-2)
	}
}

// updateNameCache updates the room display name based on the room state in the order
// specified in section 11.2.2.5 of r0.3.0 of the Client-Server API specification.
func (room *Room) updateNameCache() {
	if len(room.nameCache) == 0 {
		room.updateNameFromNameEvent()
	}
	if len(room.nameCache) == 0 {
		room.updateNameFromCanonicalAlias()
	}
	if len(room.nameCache) == 0 {
		room.updateNameFromAliases()
	}
	if len(room.nameCache) == 0 {
		room.updateNameFromMembers()
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
func (room *Room) createMemberCache() map[string]*Member {
	cache := make(map[string]*Member)
	events := room.GetStateEvents("m.room.member")
	room.firstMemberCache = ""
	if events != nil {
		for userID, event := range events {
			if len(room.firstMemberCache) == 0 && userID != room.Owner {
				room.firstMemberCache = userID
			}
			member := eventToRoomMember(userID, event)
			if member.Membership != "leave" {
				cache[member.UserID] = member
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
func (room *Room) GetMembers() map[string]*Member {
	if len(room.memberCache) == 0 {
		room.createMemberCache()
	}
	return room.memberCache
}

// GetMember returns the member with the given MXID.
// If the member doesn't exist, nil is returned.
func (room *Room) GetMember(userID string) *Member {
	if len(room.memberCache) == 0 {
		room.createMemberCache()
	}
	member, _ := room.memberCache[userID]
	return member
}

// NewRoom creates a new Room with the given ID
func NewRoom(roomID, owner string) *Room {
	return &Room{
		Room: gomatrix.NewRoom(roomID),
		Owner: owner,
	}
}
