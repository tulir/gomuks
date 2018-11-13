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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/mautrix"
	"maunium.net/go/gomuks/matrix/pushrules"
)

var mapExamplePushRules map[string]interface{}

func init() {
	mapExamplePushRules = make(map[string]interface{})
	json.Unmarshal([]byte(JSONExamplePushRules), &mapExamplePushRules)
}

func TestEventToPushRules(t *testing.T) {
	event := &mautrix.Event{
		Type:      "m.push_rules",
		Timestamp: 1523380910,
		Content:   mapExamplePushRules,
	}
	pushRuleset, err := pushrules.EventToPushRules(event)
	assert.Nil(t, err)
	assert.NotNil(t, pushRuleset)

	assert.IsType(t, pushRuleset.Override, pushrules.PushRuleArray{})
	assert.IsType(t, pushRuleset.Content, pushrules.PushRuleArray{})
	assert.IsType(t, pushRuleset.Room, pushrules.PushRuleMap{})
	assert.IsType(t, pushRuleset.Sender, pushrules.PushRuleMap{})
	assert.IsType(t, pushRuleset.Underride, pushrules.PushRuleArray{})
	assert.Len(t, pushRuleset.Override, 2)
	assert.Len(t, pushRuleset.Content, 1)
	assert.Empty(t, pushRuleset.Room.Map)
	assert.Empty(t, pushRuleset.Sender.Map)
	assert.Len(t, pushRuleset.Underride, 6)

	assert.Len(t, pushRuleset.Content[0].Actions, 3)
	assert.True(t, pushRuleset.Content[0].Default)
	assert.True(t, pushRuleset.Content[0].Enabled)
	assert.Empty(t, pushRuleset.Content[0].Conditions)
	assert.Equal(t, "alice", pushRuleset.Content[0].Pattern)
	assert.Equal(t, ".m.rule.contains_user_name", pushRuleset.Content[0].RuleID)

	assert.False(t, pushRuleset.Override[0].Actions.Should().Notify)
	assert.True(t, pushRuleset.Override[0].Actions.Should().NotifySpecified)
}

const JSONExamplePushRules = `{
  "global": {
    "content": [
      {
        "actions": [
          "notify",
          {
            "set_tweak": "sound",
            "value": "default"
          },
          {
            "set_tweak": "highlight"
          }
        ],
        "default": true,
        "enabled": true,
        "pattern": "alice",
        "rule_id": ".m.rule.contains_user_name"
      }
    ],
    "override": [
      {
        "actions": [
          "dont_notify"
        ],
        "conditions": [],
        "default": true,
        "enabled": false,
        "rule_id": ".m.rule.master"
      },
      {
        "actions": [
          "dont_notify"
        ],
        "conditions": [
          {
            "key": "content.msgtype",
            "kind": "event_match",
            "pattern": "m.notice"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.suppress_notices"
      }
    ],
    "room": [],
    "sender": [],
    "underride": [
      {
        "actions": [
          "notify",
          {
            "set_tweak": "sound",
            "value": "ring"
          },
          {
            "set_tweak": "highlight",
            "value": false
          }
        ],
        "conditions": [
          {
            "key": "type",
            "kind": "event_match",
            "pattern": "m.call.invite"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.call"
      },
      {
        "actions": [
          "notify",
          {
            "set_tweak": "sound",
            "value": "default"
          },
          {
            "set_tweak": "highlight"
          }
        ],
        "conditions": [
          {
            "kind": "contains_display_name"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.contains_display_name"
      },
      {
        "actions": [
          "notify",
          {
            "set_tweak": "sound",
            "value": "default"
          },
          {
            "set_tweak": "highlight",
            "value": false
          }
        ],
        "conditions": [
          {
            "is": "2",
            "kind": "room_member_count"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.room_one_to_one"
      },
      {
        "actions": [
          "notify",
          {
            "set_tweak": "sound",
            "value": "default"
          },
          {
            "set_tweak": "highlight",
            "value": false
          }
        ],
        "conditions": [
          {
            "key": "type",
            "kind": "event_match",
            "pattern": "m.room.member"
          },
          {
            "key": "content.membership",
            "kind": "event_match",
            "pattern": "invite"
          },
          {
            "key": "state_key",
            "kind": "event_match",
            "pattern": "@alice:example.com"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.invite_for_me"
      },
      {
        "actions": [
          "notify",
          {
            "set_tweak": "highlight",
            "value": false
          }
        ],
        "conditions": [
          {
            "key": "type",
            "kind": "event_match",
            "pattern": "m.room.member"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.member_event"
      },
      {
        "actions": [
          "notify",
          {
            "set_tweak": "highlight",
            "value": false
          }
        ],
        "conditions": [
          {
            "key": "type",
            "kind": "event_match",
            "pattern": "m.room.message"
          }
        ],
        "default": true,
        "enabled": true,
        "rule_id": ".m.rule.message"
      }
    ]
  }
}`
