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

import "encoding/json"

// PushActionType is the type of a PushAction
type PushActionType string

// The allowed push action types as specified in spec section 11.12.1.4.1.
const (
	ActionNotify     PushActionType = "notify"
	ActionDontNotify PushActionType = "dont_notify"
	ActionCoalesce   PushActionType = "coalesce"
	ActionSetTweak   PushActionType = "set_tweak"
)

// PushActionTweak is the type of the tweak in SetTweak push actions.
type PushActionTweak string

// The allowed tweak types as specified in spec section 11.12.1.4.1.1.
const (
	TweakSound     PushActionTweak = "sound"
	TweakHighlight PushActionTweak = "highlight"
)

// PushActionArray is an array of PushActions.
type PushActionArray []*PushAction

// PushActionArrayShould contains the important information parsed from a PushActionArray.
type PushActionArrayShould struct {
	// Whether or not the array contained a Notify, DontNotify or Coalesce action type.
	NotifySpecified bool
	// Whether or not the event in question should trigger a notification.
	Notify bool
	// Whether or not the event in question should be highlighted.
	Highlight bool

	// Whether or not the event in question should trigger a sound alert.
	PlaySound bool
	// The name of the sound to play if PlaySound is true.
	SoundName string
}

// Should parses this push action array and returns the relevant details wrapped in a PushActionArrayShould struct.
func (actions PushActionArray) Should() (should PushActionArrayShould) {
	for _, action := range actions {
		switch action.Action {
		case ActionNotify, ActionCoalesce:
			should.Notify = true
			should.NotifySpecified = true
		case ActionDontNotify:
			should.Notify = false
			should.NotifySpecified = true
		case ActionSetTweak:
			switch action.Tweak {
			case TweakHighlight:
				var ok bool
				should.Highlight, ok = action.Value.(bool)
				if !ok {
					// Highlight value not specified, so assume true since the tweak is set.
					should.Highlight = true
				}
			case TweakSound:
				should.SoundName = action.Value.(string)
				should.PlaySound = len(should.SoundName) > 0
			}
		}
	}
	return
}

// PushAction is a single action that should be triggered when receiving a message.
type PushAction struct {
	Action PushActionType
	Tweak  PushActionTweak
	Value  interface{}
}

// UnmarshalJSON parses JSON into this PushAction.
//
//  * If the JSON is a single string, the value is stored in the Action field.
//  * If the JSON is an object with the set_tweak field, Action will be set to
//    "set_tweak", Tweak will be set to the value of the set_tweak field and
//    and Value will be set to the value of the value field.
//  * In any other case, the function does nothing.
func (action *PushAction) UnmarshalJSON(raw []byte) error {
	var data interface{}

	err := json.Unmarshal(raw, &data)
	if err != nil {
		return err
	}

	switch val := data.(type) {
	case string:
		action.Action = PushActionType(val)
	case map[string]interface{}:
		tweak, ok := val["set_tweak"].(string)
		if ok {
			action.Action = ActionSetTweak
			action.Tweak = PushActionTweak(tweak)
			action.Value, _ = val["value"]
		}
	}
	return nil
}

// MarshalJSON is the reverse of UnmarshalJSON()
func (action *PushAction) MarshalJSON() (raw []byte, err error) {
	if action.Action == ActionSetTweak {
		data := map[string]interface{}{
			"set_tweak": action.Tweak,
			"value":     action.Value,
		}
		return json.Marshal(&data)
	} else {
		data := string(action.Action)
		return json.Marshal(&data)
	}
}
