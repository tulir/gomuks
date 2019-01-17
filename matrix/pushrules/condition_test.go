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

package pushrules_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/mautrix"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
)

var (
	blankTestRoom       *rooms.Room
	displaynameTestRoom pushrules.Room

	countConditionTestEvent *mautrix.Event

	displaynamePushCondition *pushrules.PushCondition
)

func init() {
	blankTestRoom = rooms.NewRoom("!fakeroom:maunium.net", "@tulir:maunium.net")

	countConditionTestEvent = &mautrix.Event{
		Sender:    "@tulir:maunium.net",
		Type:      mautrix.EventMessage,
		Timestamp: 1523791120,
		ID:        "$123:maunium.net",
		RoomID:    "!fakeroom:maunium.net",
		Content: mautrix.Content{
			MsgType: mautrix.MsgText,
			Body: "test",
		},
	}

	displaynameTestRoom = newFakeRoom(4)
	displaynamePushCondition = &pushrules.PushCondition{
		Kind: pushrules.KindContainsDisplayName,
	}
}

func newFakeEvent(evtType mautrix.EventType, content mautrix.Content) *mautrix.Event {
	return &mautrix.Event{
		Sender:    "@tulir:maunium.net",
		Type:      evtType,
		Timestamp: 1523791120,
		ID:        "$123:maunium.net",
		RoomID:    "!fakeroom:maunium.net",
		Content:   content,
	}
}

func newCountPushCondition(condition string) *pushrules.PushCondition {
	return &pushrules.PushCondition{
		Kind:                 pushrules.KindRoomMemberCount,
		MemberCountCondition: condition,
	}
}

func newMatchPushCondition(key, pattern string) *pushrules.PushCondition {
	return &pushrules.PushCondition{
		Kind:    pushrules.KindEventMatch,
		Key:     key,
		Pattern: pattern,
	}
}

func TestPushCondition_Match_InvalidKind(t *testing.T) {
	condition := &pushrules.PushCondition{
		Kind: pushrules.PushCondKind("invalid"),
	}
	event := newFakeEvent(mautrix.EventType{Type: "m.room.foobar"}, mautrix.Content{})
	assert.False(t, condition.Match(blankTestRoom, event))
}

type FakeRoom struct {
	members map[string]*mautrix.Member
	owner   string
}

func newFakeRoom(memberCount int) *FakeRoom {
	room := &FakeRoom{
		owner:   "@tulir:maunium.net",
		members: make(map[string]*mautrix.Member),
	}

	if memberCount >= 1 {
		room.members["@tulir:maunium.net"] = &mautrix.Member{
			Membership:  mautrix.MembershipJoin,
			Displayname: "tulir",
		}
	}

	for i := 0; i < memberCount-1; i++ {
		mxid := fmt.Sprintf("@extrauser_%d:matrix.org", i)
		room.members[mxid] = &mautrix.Member{
			Membership:  mautrix.MembershipJoin,
			Displayname: fmt.Sprintf("Extra User %d", i),
		}
	}

	return room
}

func (fr *FakeRoom) GetMember(mxid string) *mautrix.Member {
	return fr.members[mxid]
}

func (fr *FakeRoom) GetSessionOwner() string {
	return fr.owner
}

func (fr *FakeRoom) GetMembers() map[string]*mautrix.Member {
	return fr.members
}
