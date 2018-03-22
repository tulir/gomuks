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

package rooms

import (
	"maunium.net/go/gomatrix"
)

type Membership string

// The allowed membership states as specified in spec section 10.5.5.
const (
	MembershipJoin   Membership = "join"
	MembershipLeave  Membership = "leave"
	MembershipInvite Membership = "invite"
	MembershipKnock  Membership = "knock"
)

// Member represents a member in a room.
type Member struct {
	// The MXID of the member.
	UserID string `json:"-"`
	// The membership status. Defaults to leave.
	Membership Membership `json:"membership"`
	// The display name of the user. Defaults to the user ID.
	DisplayName string `json:"displayname"`
	// The avatar URL of the user. Defaults to an empty string.
	AvatarURL string `json:"avatar_url"`
}

// eventToRoomMember converts a m.room.member state event into a Member object.
func eventToRoomMember(userID string, event *gomatrix.Event) *Member {
	if event == nil {
		return &Member{
			UserID:     userID,
			Membership: MembershipLeave,
		}
	}
	membership, _ := event.Content["membership"].(string)
	avatarURL, _ := event.Content["avatar_url"].(string)

	displayName, _ := event.Content["displayname"].(string)
	if len(displayName) == 0 {
		displayName = userID
	}

	return &Member{
		UserID:      userID,
		Membership:  Membership(membership),
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
}
