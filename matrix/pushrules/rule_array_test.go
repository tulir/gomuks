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
	"maunium.net/go/mautrix"
	"testing"
)

func TestPushRuleArray_GetActions_FirstMatchReturns(t *testing.T) {
	cond1 := newMatchPushCondition("content.msgtype", "m.emote")
	cond2 := newMatchPushCondition("content.body", "no match")
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "ping"},
	}
	rule1 := &pushrules.PushRule{
		Type:       pushrules.OverrideRule,
		Enabled:    true,
		Conditions: []*pushrules.PushCondition{cond1, cond2},
		Actions:    actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule2 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!fakeroom:maunium.net",
		Actions: actions2,
	}

	actions3 := pushrules.PushActionArray{
		{Action: pushrules.ActionCoalesce},
	}
	rule3 := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@tulir:maunium.net",
		Actions: actions3,
	}

	rules := pushrules.PushRuleArray{rule1, rule2, rule3}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.Equal(t, rules.GetActions(blankTestRoom, event), actions2)
}

func TestPushRuleArray_GetActions_NoMatchesIsNil(t *testing.T) {
	cond1 := newMatchPushCondition("content.msgtype", "m.emote")
	cond2 := newMatchPushCondition("content.body", "no match")
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "ping"},
	}
	rule1 := &pushrules.PushRule{
		Type:       pushrules.OverrideRule,
		Enabled:    true,
		Conditions: []*pushrules.PushCondition{cond1, cond2},
		Actions:    actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule2 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!realroom:maunium.net",
		Actions: actions2,
	}

	actions3 := pushrules.PushActionArray{
		{Action: pushrules.ActionCoalesce},
	}
	rule3 := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@otheruser:maunium.net",
		Actions: actions3,
	}

	rules := pushrules.PushRuleArray{rule1, rule2, rule3}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.Nil(t, rules.GetActions(blankTestRoom, event))
}

func TestPushRuleMap_GetActions_RoomRuleExists(t *testing.T) {
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule1 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!realroom:maunium.net",
		Actions: actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
	}
	rule2 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!thirdroom:maunium.net",
		Actions: actions2,
	}

	actions3 := pushrules.PushActionArray{
		{Action: pushrules.ActionCoalesce},
	}
	rule3 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!fakeroom:maunium.net",
		Actions: actions3,
	}

	rules := pushrules.PushRuleMap{
		Map: map[string]*pushrules.PushRule{
			rule1.RuleID: rule1,
			rule2.RuleID: rule2,
			rule3.RuleID: rule3,
		},
		Type: pushrules.RoomRule,
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.Equal(t, rules.GetActions(blankTestRoom, event), actions3)
}

func TestPushRuleMap_GetActions_RoomRuleDoesntExist(t *testing.T) {
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule1 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!realroom:maunium.net",
		Actions: actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
	}
	rule2 := &pushrules.PushRule{
		Type:    pushrules.RoomRule,
		Enabled: true,
		RuleID:  "!thirdroom:maunium.net",
		Actions: actions2,
	}

	rules := pushrules.PushRuleMap{
		Map: map[string]*pushrules.PushRule{
			rule1.RuleID: rule1,
			rule2.RuleID: rule2,
		},
		Type: pushrules.RoomRule,
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.Nil(t, rules.GetActions(blankTestRoom, event))
}

func TestPushRuleMap_GetActions_SenderRuleExists(t *testing.T) {
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule1 := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@tulir:maunium.net",
		Actions: actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
	}
	rule2 := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@someone:maunium.net",
		Actions: actions2,
	}

	actions3 := pushrules.PushActionArray{
		{Action: pushrules.ActionCoalesce},
	}
	rule3 := &pushrules.PushRule{
		Type:    pushrules.SenderRule,
		Enabled: true,
		RuleID:  "@otheruser:matrix.org",
		Actions: actions3,
	}

	rules := pushrules.PushRuleMap{
		Map: map[string]*pushrules.PushRule{
			rule1.RuleID: rule1,
			rule2.RuleID: rule2,
			rule3.RuleID: rule3,
		},
		Type: pushrules.SenderRule,
	}

	event := newFakeEvent(mautrix.EventMessage, mautrix.Content{
		MsgType: mautrix.MsgEmote,
		Body:    "is testing pushrules",
	})
	assert.Equal(t, rules.GetActions(blankTestRoom, event), actions1)
}

func TestPushRuleArray_SetTypeAndMap(t *testing.T) {
	actions1 := pushrules.PushActionArray{
		{Action: pushrules.ActionDontNotify},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakHighlight, Value: false},
		{Action: pushrules.ActionSetTweak, Tweak: pushrules.TweakSound, Value: "pong"},
	}
	rule1 := &pushrules.PushRule{
		Enabled: true,
		RuleID:  "@tulir:maunium.net",
		Actions: actions1,
	}

	actions2 := pushrules.PushActionArray{
		{Action: pushrules.ActionNotify},
	}
	rule2 := &pushrules.PushRule{
		Enabled: true,
		RuleID:  "@someone:maunium.net",
		Actions: actions2,
	}

	actions3 := pushrules.PushActionArray{
		{Action: pushrules.ActionCoalesce},
	}
	rule3 := &pushrules.PushRule{
		Enabled: true,
		RuleID:  "@otheruser:matrix.org",
		Actions: actions3,
	}

	ruleArray := pushrules.PushRuleArray{rule1, rule2, rule3}
	ruleMap := ruleArray.SetTypeAndMap(pushrules.SenderRule)
	assert.Equal(t, pushrules.SenderRule, ruleMap.Type)
	for _, rule := range ruleArray {
		assert.Equal(t, rule, ruleMap.Map[rule.RuleID])
	}
	newRuleArray := ruleMap.Unmap()
	for _, rule := range ruleArray {
		assert.Contains(t, newRuleArray, rule)
	}
}
