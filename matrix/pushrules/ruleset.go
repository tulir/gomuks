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

package pushrules

import (
	"encoding/json"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/rooms"
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

func (rs *PushRuleset) UnmarshalJSON(raw []byte) (err error) {
	data := rawPushRuleset{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return
	}

	rs.Override = data.Override.setType(OverrideRule)
	rs.Content = data.Content.setType(ContentRule)
	rs.Room = data.Room.setTypeAndMap(RoomRule)
	rs.Sender = data.Sender.setTypeAndMap(SenderRule)
	rs.Underride = data.Underride.setType(UnderrideRule)
	return
}

func (rs *PushRuleset) MarshalJSON() ([]byte, error) {
	data := rawPushRuleset{
		Override:  rs.Override,
		Content:   rs.Content,
		Room:      rs.Room.unmap(),
		Sender:    rs.Sender.unmap(),
		Underride: rs.Underride,
	}
	return json.Marshal(&data)
}

var DefaultPushActions = make(PushActionArray, 0)

func (rs *PushRuleset) GetActions(room *rooms.Room, event *gomatrix.Event) (match PushActionArray) {
	if match = rs.Override.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Content.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Room.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Sender.GetActions(room, event); match != nil {
		return
	}
	if match = rs.Underride.GetActions(room, event); match != nil {
		return
	}
	return DefaultPushActions
}
