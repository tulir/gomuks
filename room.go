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
	"maunium.net/go/gomatrix"
)

// Room represents a single Matrix room.
type Room struct {
	*gomatrix.Room

	PrevBatch   string
	memberCache map[string]*RoomMember
	nameCache   string
	topicCache  string
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
	case "m.room.name":
	case "m.room.canonical_alias":
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

// GetTitle returns the display title of the room.
func (room *Room) GetTitle() string {
	if len(room.nameCache) == 0 {
		nameEvt := room.GetStateEvent("m.room.name", "")
		if nameEvt != nil {
			room.nameCache, _ = nameEvt.Content["name"].(string)
		}
	}
	if len(room.nameCache) == 0 {
		canonicalAliasEvt := room.GetStateEvent("m.room.canonical_alias", "")
		if canonicalAliasEvt != nil {
			room.nameCache, _ = canonicalAliasEvt.Content["alias"].(string)
		}
	}
	if len(room.nameCache) == 0 {
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
	if len(room.nameCache) == 0 {
		// TODO follow other title rules in spec
		room.nameCache = room.ID
	}
	return room.nameCache
}

type RoomMember struct {
	UserID      string `json:"-"`
	Membership  string `json:"membership"`
	DisplayName string `json:"displayname"`
	AvatarURL   string `json:"avatar_url"`
}

func eventToRoomMember(userID string, event *gomatrix.Event) *RoomMember {
	if event == nil {
		return &RoomMember{
			UserID:     userID,
			Membership: "leave",
		}
	}
	membership, _ := event.Content["membership"].(string)
	avatarURL, _ := event.Content["avatar_url"].(string)

	displayName, _ := event.Content["displayname"].(string)
	if len(displayName) == 0 {
		displayName = userID
	}

	return &RoomMember{
		UserID:      userID,
		Membership:  membership,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}

func (room *Room) createMemberCache() map[string]*RoomMember {
	cache := make(map[string]*RoomMember)
	events := room.GetStateEvents("m.room.member")
	if events != nil {
		for userID, event := range events {
			member := eventToRoomMember(userID, event)
			if member.Membership != "leave" {
				cache[member.UserID] = member
			}
		}
	}
	room.memberCache = cache
	return cache
}

func (room *Room) GetMembers() map[string]*RoomMember {
	if len(room.memberCache) == 0 {
		room.createMemberCache()
	}
	return room.memberCache
}

func (room *Room) GetMember(userID string) *RoomMember {
	if len(room.memberCache) == 0 {
		room.createMemberCache()
	}
	member, _ := room.memberCache[userID]
	return member
}

// NewRoom creates a new Room with the given ID
func NewRoom(roomID string) *Room {
	return &Room{
		Room: gomatrix.NewRoom(roomID),
	}
}
