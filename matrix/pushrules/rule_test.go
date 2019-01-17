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
	"github.com/stretchr/testify/assert"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/mautrix"
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

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		Raw: map[string]interface{}{
			"msgtype": "m.emote",
			"body": "is testing pushrules",
		},
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
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

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		Raw: map[string]interface{}{
			"msgtype": "m.emote",
			"body": "is testing pushrules",
		},
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
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

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		Raw: map[string]interface{}{
			"msgtype": "m.text",
			"body": "I'm testing pushrules",
		},
		MsgType: mautrix.MsgText,
		Body:    "I'm testing pushrules",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "is testing*",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "is testing*",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is not testing pushrules",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_ImplicitGlob(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "testing",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is not testing pushrules",
	})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Content_IllegalGlob(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.ContentRule,
		Enabled: true,
		Pattern: "this is not a valid glo[b",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "this is not a valid glob",
	})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Room(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!fakeroom:maunium.net",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Room_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!otherroom:maunium.net",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Sender(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@tulir:maunium.net",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{})
	assert.True(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_Sender_Fail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "@someone:matrix.org",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{})
	assert.False(t, rule.Match(blankTestRoom, event))
}

func TestPushRule_Match_UnknownTypeAlwaysFail(t *testing.T) {
	rule := &pushrules.PushRule{
		Type:    pushrules.PushRuleType("foobar"),
		Enabled: true,
		RuleID:  "@someone:matrix.org",
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{})
	assert.False(t, rule.Match(blankTestRoom, event))
}
