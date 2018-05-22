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

package pushrules_test

import (
	"github.com/stretchr/testify/assert"
	"maunium.net/go/gomuks/matrix/pushrules"
	"testing"
)

func TestPushRule_Match_Conditions(t *testing.T) {
	cond1 := newMatchPushCondition("content.msgtype", "m.emote")
	cond2 := newMatchPushCondition("content.body", "*pushrules")
	rule := &pushrules.PushRule{
		Type:       pushrules.OverrideRule,
		Enabled:    true,
		Conditions: []*pushrules.PushCondition{cond1, cond2},
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "is testing pushrules",
	})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Conditions_Disabled(t *testing.T) {
	cond1 := newMatchPushCondition("content.msgtype", "m.emote")
	cond2 := newMatchPushCondition("content.body", "*pushrules")
	rule := &pushrules.PushRule{
		Type:       pushrules.OverrideRule,
		Enabled:    false,
		Conditions: []*pushrules.PushCondition{cond1, cond2},
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "is testing pushrules",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Conditions_FailIfOneFails(t *testing.T) {
	cond1 := newMatchPushCondition("content.msgtype", "m.emote")
	cond2 := newMatchPushCondition("content.body", "*pushrules")
	rule := &pushrules.PushRule{
		Type:       pushrules.OverrideRule,
		Enabled:    true,
		Conditions: []*pushrules.PushCondition{cond1, cond2},
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.text",
		"body":    "I'm testing pushrules",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "is testing*",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "is testing pushrules",
	})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "is testing*",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "is not testing pushrules",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_ImplicitGlob(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "testing",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "is not testing pushrules",
	})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_IllegalGlob(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "this is not a valid glo[b",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.emote",
		"body":    "this is not a valid glob",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Room(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!fakeroom:maunium.net",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Room_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!otherroom:maunium.net",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Sender(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@tulir:maunium.net",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Sender_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "@someone:matrix.org",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_UnknownTypeAlwaysFail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.PushRuleType("foobar"),
		Enabled: true,
		RuleID:  "@someone:matrix.org",
	}

	event := newFakeEvent("m.room.message", map[string]interface{}{})
	assert.False(t, rule.Match(blankTestRoom, event))
}
