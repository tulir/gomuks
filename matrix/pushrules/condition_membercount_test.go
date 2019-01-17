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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushCondition_Match_KindMemberCount_OneToOne_ImplicitPrefix(t *testing.T) {
	condition := newCountPushCondition("2")
	room := newFakeRoom(2)
	assert.True(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_OneToOne_ExplicitPrefix(t *testing.T) {
	condition := newCountPushCondition("==2")
	room := newFakeRoom(2)
	assert.True(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_BigRoom(t *testing.T) {
	condition := newCountPushCondition(">200")
	room := newFakeRoom(201)
	assert.True(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_BigRoom_Fail(t *testing.T) {
	condition := newCountPushCondition(">=200")
	room := newFakeRoom(199)
	assert.False(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_SmallRoom(t *testing.T) {
	condition := newCountPushCondition("<10")
	room := newFakeRoom(9)
	assert.True(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_SmallRoom_Fail(t *testing.T) {
	condition := newCountPushCondition("<=10")
	room := newFakeRoom(11)
	assert.False(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_InvalidPrefix(t *testing.T) {
	condition := newCountPushCondition("??10")
	room := newFakeRoom(11)
	assert.False(t, condition.Match(room, countConditionTestEvent))
}

func TestPushCondition_Match_KindMemberCount_InvalidCondition(t *testing.T) {
	condition := newCountPushCondition("foobar")
	room := newFakeRoom(1)
	assert.False(t, condition.Match(room, countConditionTestEvent))
}
