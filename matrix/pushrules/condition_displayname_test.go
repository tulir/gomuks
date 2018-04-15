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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushCondition_Match_DisplayName(t *testing.T) {
	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.text",
		"body":    "tulir: test mention",
	})
	event.Sender = "@someone_else:matrix.org"
	assert.True(t, displaynamePushCondition.Match(displaynameTestRoom, event))
}

func TestPushCondition_Match_DisplayName_Fail(t *testing.T) {
	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.text",
		"body":    "not a mention",
	})
	event.Sender = "@someone_else:matrix.org"
	assert.False(t, displaynamePushCondition.Match(displaynameTestRoom, event))
}

func TestPushCondition_Match_DisplayName_CantHighlightSelf(t *testing.T) {
	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.text",
		"body":    "tulir: I can't highlight myself",
	})
	assert.False(t, displaynamePushCondition.Match(displaynameTestRoom, event))
}

func TestPushCondition_Match_DisplayName_FailsOnEmptyRoom(t *testing.T) {
	emptyRoom := newFakeRoom(0)
	event := newFakeEvent("m.room.message", map[string]interface{}{
		"msgtype": "m.text",
		"body":    "tulir: this room doesn't have the owner Member available, so it fails.",
	})
	event.Sender = "@someone_else:matrix.org"
	assert.False(t, displaynamePushCondition.Match(emptyRoom, event))
}
