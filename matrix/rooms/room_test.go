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

package rooms_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/rooms"
)

func TestNewRoom_DefaultValues(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	assert.Equal(t, "!test:maunium.net", room.ID)
	assert.Equal(t, "@tulir:maunium.net", room.SessionUserID)
	assert.Empty(t, room.GetMembers())
	assert.Equal(t, "Empty room", room.GetTitle())
	assert.Empty(t, room.GetAliases())
	assert.Empty(t, room.GetCanonicalAlias())
	assert.Empty(t, room.GetTopic())
	assert.Nil(t, room.GetSessionOwner())
}

func TestRoom_GetCanonicalAlias(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	room.UpdateState(&gomatrix.Event{
		Type: "m.room.canonical_alias",
		Content: map[string]interface{}{
			"alias": "#foo:maunium.net",
		},
	})
	assert.Equal(t, "#foo:maunium.net", room.GetCanonicalAlias())
}

func TestRoom_GetTopic(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	room.UpdateState(&gomatrix.Event{
		Type: "m.room.topic",
		Content: map[string]interface{}{
			"topic": "test topic",
		},
	})
	assert.Equal(t, "test topic", room.GetTopic())
}

func TestRoom_Tags_Empty(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	assert.Empty(t, room.RawTags)
	tags := room.Tags()
	assert.Len(t, tags, 1)
	assert.Equal(t, "", tags[0].Tag)
	assert.Equal(t, "0.5", tags[0].Order)
}

func TestRoom_Tags_NotEmpty(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	room.RawTags = []rooms.RoomTag{{Tag: "foo", Order: "1"}, {Tag: "bar", Order: "1"}}
	tags := room.Tags()
	assert.Equal(t, room.RawTags, tags)
}

func TestRoom_GetAliases(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addAliases(room)

	aliases := room.GetAliases()
	assert.Contains(t, aliases, "#bar:maunium.net")
	assert.Contains(t, aliases, "#test:maunium.net")
	assert.Contains(t, aliases, "#foo:matrix.org")
	assert.Contains(t, aliases, "#test:matrix.org")
}

func addName(room *rooms.Room) {
	room.UpdateState(&gomatrix.Event{
		Type: "m.room.name",
		Content: map[string]interface{}{
			"name": "Test room",
		},
	})
}

func addCanonicalAlias(room *rooms.Room) {
	room.UpdateState(&gomatrix.Event{
		Type: "m.room.canonical_alias",
		Content: map[string]interface{}{
			"alias": "#foo:maunium.net",
		},
	})
}

func addAliases(room *rooms.Room) {
	server1 := "maunium.net"
	room.UpdateState(&gomatrix.Event{
		Type:     "m.room.aliases",
		StateKey: &server1,
		Content: map[string]interface{}{
			"aliases": []interface{}{"#bar:maunium.net", "#test:maunium.net", "#foo:maunium.net"},
		},
	})

	server2 := "matrix.org"
	room.UpdateState(&gomatrix.Event{
		Type:     "m.room.aliases",
		StateKey: &server2,
		Content: map[string]interface{}{
			"aliases": []interface{}{"#foo:matrix.org", "#test:matrix.org"},
		},
	})
}

func addMembers(room *rooms.Room, count int) {
	user1 := "@tulir:maunium.net"
	room.UpdateState(&gomatrix.Event{
		Type:     "m.room.member",
		StateKey: &user1,
		Content: map[string]interface{}{
			"displayname": "tulir",
			"membership":  "join",
		},
	})

	for i := 1; i < count; i++ {
		userN := fmt.Sprintf("@user_%d:matrix.org", i+1)
		content := map[string]interface{}{
			"membership": "join",
		}
		if i%2 == 1 {
			content["displayname"] = fmt.Sprintf("User #%d", i+1)
		}
		if i%5 == 0 {
			content["membership"] = "invite"
		}
		room.UpdateState(&gomatrix.Event{
			Type:     "m.room.member",
			StateKey: &userN,
			Content:  content,
		})
	}
}

func TestRoom_GetMembers(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 6)

	members := room.GetMembers()
	assert.Len(t, members, 6)
}

func TestRoom_GetMember(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 6)

	assert.NotNil(t, room.GetMember("@user_2:matrix.org"))
	assert.NotNil(t, room.GetMember("@tulir:maunium.net"))
	assert.Equal(t, room.GetMember("@tulir:maunium.net"), room.GetSessionOwner())
}

func TestRoom_GetTitle_ExplicitName(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 4)
	addName(room)
	addCanonicalAlias(room)
	addAliases(room)
	assert.Equal(t, "Test room", room.GetTitle())
}

func TestRoom_GetTitle_CanonicalAlias(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 4)
	addCanonicalAlias(room)
	addAliases(room)
	assert.Equal(t, "#foo:maunium.net", room.GetTitle())
}

func TestRoom_GetTitle_FirstAlias(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 2)
	addAliases(room)
	assert.Equal(t, "#bar:maunium.net", room.GetTitle())
}

func TestRoom_GetTitle_Members_Empty(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 1)
	assert.Equal(t, "Empty room", room.GetTitle())
}

func TestRoom_GetTitle_Members_OneToOne(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 2)
	assert.Equal(t, "User #2", room.GetTitle())
}

func TestRoom_GetTitle_Members_GroupChat(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")
	addMembers(room, 76)
	assert.Contains(t, room.GetTitle(), " and 74 others")
}

func TestRoom_MarkRead(t *testing.T) {
	room := rooms.NewRoom("!test:maunium.net", "@tulir:maunium.net")

	room.AddUnread("foo", true, false)
	assert.Equal(t, 1, room.UnreadCount())
	assert.False(t, room.Highlighted())

	room.AddUnread("bar", true, false)
	assert.Equal(t, 2, room.UnreadCount())
	assert.False(t, room.Highlighted())

	room.AddUnread("asd", false, true)
	assert.Equal(t, 2, room.UnreadCount())
	assert.True(t, room.Highlighted())

	room.MarkRead("")
	assert.Empty(t, room.UnreadMessages)
}
