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
	"encoding/json"

	"maunium.net/go/mautrix"
)

type PushRuleset struct {
	Override  PushRuleArray
	Content   PushRuleArray
	Room      PushRuleMap
	Sender    PushRuleMap
	Underride PushRuleArray
}

type rawPushRuleset struct {
	Override  PushRuleArray `json:"override"`
	Content   PushRuleArray `json:"content"`
	Room      PushRuleArray `json:"room"`
	Sender    PushRuleArray `json:"sender"`
	Underride PushRuleArray `json:"underride"`
}

// UnmarshalJSON parses JSON into this PushRuleset.
//
// For override, sender and underride push rule arrays, the type is added
// to each PushRule and the array is used as-is.
//
// For room and sender push rule arrays, the type is added to each PushRule
// and the array is converted to a map with the rule ID as the key and the
// PushRule as the value.
func (rs *PushRuleset) UnmarshalJSON(raw []byte) (err error) {
	data := rawPushRuleset{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return
	}

	rs.Override = data.Override.SetType(OverrideRule)
	rs.Content = data.Content.SetType(ContentRule)
	rs.Room = data.Room.SetTypeAndMap(RoomRule)
	rs.Sender = data.Sender.SetTypeAndMap(SenderRule)
	rs.Underride = data.Underride.SetType(UnderrideRule)
	return
}

// MarshalJSON is the reverse of UnmarshalJSON()
func (rs *PushRuleset) MarshalJSON() ([]byte, error) {
	data := rawPushRuleset{
		Override:  rs.Override,
		Content:   rs.Content,
		Room:      rs.Room.Unmap(),
		Sender:    rs.Sender.Unmap(),
		Underride: rs.Underride,
	}
	return json.Marshal(&data)
}

// DefaultPushActions is the value returned if none of the rule
// collections in a Ruleset match the event given to GetActions()
var DefaultPushActions = make(PushActionArray, 0)

// GetActions matches the given event against all of the push rule
// collections in this push ruleset in the order of priority as
// specified in spec section 11.12.1.4.
func (rs *PushRuleset) GetActions(room Room, event *mautrix.Event) (match PushActionArray) {
	// Add push rule collections to array in priority order
	arrays := []PushRuleCollection{rs.Override, rs.Content, rs.Room, rs.Sender, rs.Underride}
	// Loop until one of the push rule collections matches the room/event combo.
	for _, pra := range arrays {
		if match = pra.GetActions(room, event); match != nil {
			// Match found, return it.
			return
		}
	}
	// No match found, return default actions.
	return DefaultPushActions
}
