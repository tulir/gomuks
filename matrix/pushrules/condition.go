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

package pushrules

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/lib/glob"
)

// Room is an interface with the functions that are needed for processing room-specific push conditions
type Room interface {
	GetMember(mxid string) *rooms.Member
	GetMembers() map[string]*rooms.Member
	GetSessionOwner() string
}

// PushCondKind is the type of a push condition.
type PushCondKind string

// The allowed push condition kinds as specified in section 11.12.1.4.3 of r0.3.0 of the Client-Server API.
const (
	KindEventMatch          PushCondKind = "event_match"
	KindContainsDisplayName PushCondKind = "contains_display_name"
	KindRoomMemberCount     PushCondKind = "room_member_count"
)

// PushCondition wraps a condition that is required for a specific PushRule to be used.
type PushCondition struct {
	// The type of the condition.
	Kind PushCondKind `json:"kind"`
	// The dot-separated field of the event to match. Only applicable if kind is EventMatch.
	Key string `json:"key,omitempty"`
	// The glob-style pattern to match the field against. Only applicable if kind is EventMatch.
	Pattern string `json:"pattern,omitempty"`
	// The condition that needs to be fulfilled for RoomMemberCount-type conditions.
	// A decimal integer optionally prefixed by ==, <, >, >= or <=. Prefix "==" is assumed if no prefix found.
	MemberCountCondition string `json:"is,omitempty"`
}

// MemberCountFilterRegex is the regular expression to parse the MemberCountCondition of PushConditions.
var MemberCountFilterRegex = regexp.MustCompile("^(==|[<>]=?)?([0-9]+)$")

// Match checks if this condition is fulfilled for the given event in the given room.
func (cond *PushCondition) Match(room Room, event *mautrix.Event) bool {
	switch cond.Kind {
	case KindEventMatch:
		return cond.matchValue(room, event)
	case KindContainsDisplayName:
		return cond.matchDisplayName(room, event)
	case KindRoomMemberCount:
		return cond.matchMemberCount(room, event)
	default:
		return false
	}
}

func (cond *PushCondition) matchValue(room Room, event *mautrix.Event) bool {
	index := strings.IndexRune(cond.Key, '.')
	key := cond.Key
	subkey := ""
	if index > 0 {
		subkey = key[index+1:]
		key = key[0:index]
	}

	pattern, err := glob.Compile(cond.Pattern)
	if err != nil {
		return false
	}

	switch key {
	case "type":
		return pattern.MatchString(event.Type.String())
	case "sender":
		return pattern.MatchString(event.Sender)
	case "room_id":
		return pattern.MatchString(event.RoomID)
	case "state_key":
		if event.StateKey == nil {
			return cond.Pattern == ""
		}
		return pattern.MatchString(*event.StateKey)
	case "content":
		val, _ := event.Content.Raw[subkey].(string)
		return pattern.MatchString(val)
	default:
		return false
	}
}

func (cond *PushCondition) matchDisplayName(room Room, event *mautrix.Event) bool {
	ownerID := room.GetSessionOwner()
	if ownerID == event.Sender {
		return false
	}
	member := room.GetMember(ownerID)
	if member == nil {
		return false
	}

	msg := event.Content.Body
	isAcceptable := func(r uint8) bool {
		return unicode.IsSpace(rune(r)) || unicode.IsPunct(rune(r))
	}
	length := len(member.Displayname)
	for index := strings.Index(msg, member.Displayname); index != -1; index = strings.Index(msg, member.Displayname) {
		if (index <= 0 || isAcceptable(msg[index-1])) && (index + length >= len(msg) || isAcceptable(msg[index+length])) {
			return true
		}
		msg = msg[index+len(member.Displayname):]
	}
	return false
}

func (cond *PushCondition) matchMemberCount(room Room, event *mautrix.Event) bool {
	group := MemberCountFilterRegex.FindStringSubmatch(cond.MemberCountCondition)
	if len(group) != 3 {
		return false
	}

	operator := group[1]
	wantedMemberCount, _ := strconv.Atoi(group[2])

	memberCount := len(room.GetMembers())

	switch operator {
	case "==", "":
		return memberCount == wantedMemberCount
	case ">":
		return memberCount > wantedMemberCount
	case ">=":
		return memberCount >= wantedMemberCount
	case "<":
		return memberCount < wantedMemberCount
	case "<=":
		return memberCount <= wantedMemberCount
	default:
		// Should be impossible due to regex.
		return false
	}
}
