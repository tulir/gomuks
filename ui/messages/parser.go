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

package messages

import (
	"fmt"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
)

func ParseEvent(gmx ifc.Gomuks, room *rooms.Room, evt *gomatrix.Event) UIMessage {
	member := room.GetMember(evt.Sender)
	if member != nil {
		evt.Sender = member.DisplayName
	}
	switch evt.Type {
	case "m.room.message":
		return ParseMessage(gmx, evt)
	case "m.room.member":
		return ParseMembershipEvent(evt)
	}
	return nil
}

func unixToTime(unix int64) time.Time {
	timestamp := time.Now()
	if unix != 0 {
		timestamp = time.Unix(unix/1000, unix%1000*1000)
	}
	return timestamp
}

func ParseMessage(gmx ifc.Gomuks, evt *gomatrix.Event) UIMessage {
	msgtype, _ := evt.Content["msgtype"].(string)
	ts := unixToTime(evt.Timestamp)
	switch msgtype {
	case "m.text", "m.notice":
		text, _ := evt.Content["body"].(string)
		return NewTextMessage(evt.ID, evt.Sender, msgtype, text, ts)
	case "m.image":
		url, _ := evt.Content["url"].(string)
		data, hs, id, err := gmx.Matrix().Download(url)
		if err != nil {
			debug.Printf("Failed to download %s: %v", url, err)
		}
		return NewImageMessage(gmx, evt.ID, evt.Sender, msgtype, hs, id, data, ts)
	}
	return nil
}

func getMembershipEventContent(evt *gomatrix.Event) (sender string, text tstring.TString) {
	membership, _ := evt.Content["membership"].(string)
	displayname, _ := evt.Content["displayname"].(string)
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}
	prevMembership := "leave"
	prevDisplayname := ""
	if evt.Unsigned.PrevContent != nil {
		prevMembership, _ = evt.Unsigned.PrevContent["membership"].(string)
		prevDisplayname, _ = evt.Unsigned.PrevContent["displayname"].(string)
	}

	if membership != prevMembership {
		switch membership {
		case "invite":
			sender = "---"
			text = tstring.NewColorTString(fmt.Sprintf("%s invited %s.", evt.Sender, displayname), tcell.ColorYellow)
			text.Colorize(0, len(evt.Sender), widget.GetHashColor(evt.Sender))
			text.Colorize(len(evt.Sender)+len(" invited "), len(displayname), widget.GetHashColor(displayname))
		case "join":
			sender = "-->"
			text = tstring.NewColorTString(fmt.Sprintf("%s joined the room.", displayname), tcell.ColorGreen)
			text.Colorize(0, len(displayname), widget.GetHashColor(displayname))
		case "leave":
			sender = "<--"
			if evt.Sender != *evt.StateKey {
				reason, _ := evt.Content["reason"].(string)
				text = tstring.NewColorTString(fmt.Sprintf("%s kicked %s: %s", evt.Sender, displayname, reason), tcell.ColorRed)
				text.Colorize(0, len(evt.Sender), widget.GetHashColor(evt.Sender))
				text.Colorize(len(evt.Sender)+len(" kicked "), len(displayname), widget.GetHashColor(displayname))
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s left the room.", displayname), tcell.ColorRed)
				text.Colorize(0, len(displayname), widget.GetHashColor(displayname))
			}
		}
	} else if displayname != prevDisplayname {
		sender = "---"
		text = tstring.NewColorTString(fmt.Sprintf("%s changed their display name to %s.", prevDisplayname, displayname), tcell.ColorYellow)
		text.Colorize(0, len(prevDisplayname), widget.GetHashColor(prevDisplayname))
		text.Colorize(len(prevDisplayname)+len(" changed their display name to "), len(displayname), widget.GetHashColor(displayname))
	}
	return
}

func ParseMembershipEvent(evt *gomatrix.Event) UIMessage {
	sender, text := getMembershipEventContent(evt)
	ts := unixToTime(evt.Timestamp)
	return NewExpandedTextMessage(evt.ID, sender, "m.room.membership", text, ts)
}
