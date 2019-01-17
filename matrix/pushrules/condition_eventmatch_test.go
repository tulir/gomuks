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
	"maunium.net/go/mautrix"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushCondition_Match_KindEvent_MsgType(t *testing.T) {
	condition := newMatchPushCondition("content.msgtype", "m.emote")
	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		Raw: map[string]interface{}{
			"msgtype": "m.emote",
			"body": "tests gomuks pushconditions",
		},
	})
	assert.True(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_MsgType_Fail(t *testing.T) {
	condition := newMatchPushCondition("content.msgtype", "m.emote")

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		Raw: map[string]interface{}{
			"msgtype": "m.text",
			"body": "I'm testing gomuks pushconditions",
		},
	})
	assert.False(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_EventType(t *testing.T) {
	condition := newMatchPushCondition("type", "m.room.foo")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	assert.True(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_EventType_IllegalGlob(t *testing.T) {
	condition := newMatchPushCondition("type", "m.room.invalid_glo[b")
	event := newFakeEvent(mautrix.NewEventType("m.room.invalid_glob"), mautrix.Content{})
	assert.False(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_Sender_Fail(t *testing.T) {
	condition := newMatchPushCondition("sender", "@foo:maunium.net")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	assert.False(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_RoomID(t *testing.T) {
	condition := newMatchPushCondition("room_id", "!fakeroom:maunium.net")
	event := newFakeEvent(mautrix.NewEventType(""), mautrix.Content{})
	assert.True(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_BlankStateKey(t *testing.T) {
	condition := newMatchPushCondition("state_key", "")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	assert.True(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_BlankStateKey_Fail(t *testing.T) {
	condition := newMatchPushCondition("state_key", "not blank")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	assert.False(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_NonBlankStateKey(t *testing.T) {
	condition := newMatchPushCondition("state_key", "*:maunium.net")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	event.StateKey = &event.Sender
	assert.True(t, condition.Match(blankTestRoom, event))
}

func TestPushCondition_Match_KindEvent_UnknownKey(t *testing.T) {
	condition := newMatchPushCondition("non-existent key", "doesn't affect anything")
	event := newFakeEvent(mautrix.NewEventType("m.room.foo"), mautrix.Content{})
	assert.False(t, condition.Match(blankTestRoom, event))
}
