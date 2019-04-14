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

package ui

import (
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/ui/widget"
)

type MemberList struct {
	list roomMemberList
}

func NewMemberList() *MemberList {
	return &MemberList{}
}

type memberListItem struct {
	mautrix.Member
	PowerLevel int
	UserID     string
	Color      tcell.Color
}

type roomMemberList []*memberListItem

func (rml roomMemberList) Len() int {
	return len(rml)
}

func (rml roomMemberList) Less(i, j int) bool {
	if rml[i].PowerLevel != rml[j].PowerLevel {
		return rml[i].PowerLevel > rml[j].PowerLevel
	}
	return strings.Compare(strings.ToLower(rml[i].Displayname), strings.ToLower(rml[j].Displayname)) < 0
}

func (rml roomMemberList) Swap(i, j int) {
	rml[i], rml[j] = rml[j], rml[i]
}

func (ml *MemberList) Update(data map[string]*mautrix.Member, levels *mautrix.PowerLevels) *MemberList {
	ml.list = make(roomMemberList, len(data))
	i := 0
	for userID, member := range data {
		ml.list[i] = &memberListItem{
			Member:     *member,
			UserID:     userID,
			PowerLevel: levels.GetUserLevel(userID),
			Color:      widget.GetHashColor(userID),
		}
		i++
	}
	sort.Sort(ml.list)
	return ml
}

func (ml *MemberList) Draw(screen mauview.Screen) {
	width, _ := screen.Size()
	for y, member := range ml.list {
		if member.Membership == "invite" {
			widget.WriteLineSimpleColor(screen, member.Displayname, 1, y, member.Color)
			screen.SetCell(0, y, tcell.StyleDefault, '(')
			if sw := runewidth.StringWidth(member.Displayname); sw < width-1 {
				screen.SetCell(sw+1, y, tcell.StyleDefault, ')')
			} else {
				screen.SetCell(width-1, y, tcell.StyleDefault, ')')
			}
		} else {
			widget.WriteLineSimpleColor(screen, member.Displayname, 0, y, member.Color)
		}
	}
}
