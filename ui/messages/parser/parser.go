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

package parser

import (
	"fmt"
	"strings"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
)

func ParseEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *gomatrix.Event) messages.UIMessage {
	switch evt.Type {
	case "m.room.message":
		return ParseMessage(matrix, room, evt)
	case "m.room.member":
		return ParseMembershipEvent(room, evt)
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

func ParseMessage(matrix ifc.MatrixContainer, room *rooms.Room, evt *gomatrix.Event) messages.UIMessage {
	displayname := evt.Sender
	member := room.GetMember(evt.Sender)
	if member != nil {
		displayname = member.DisplayName
	}
	msgtype, _ := evt.Content["msgtype"].(string)
	ts := unixToTime(evt.Timestamp)
	switch msgtype {
	case "m.text", "m.notice", "m.emote":
		format, hasFormat := evt.Content["format"].(string)
		if hasFormat && format == "org.matrix.custom.html" {
			text := ParseHTMLMessage(room, evt, displayname)
			return messages.NewExpandedTextMessage(evt.ID, evt.Sender, displayname, msgtype, text, ts)
		}
		text, _ := evt.Content["body"].(string)
		text = strings.Replace(text, "\t", "    ", -1)
		return messages.NewTextMessage(evt.ID, evt.Sender, displayname, msgtype, text, ts)
	case "m.image":
		url, _ := evt.Content["url"].(string)
		data, hs, id, err := matrix.Download(url)
		if err != nil {
			debug.Printf("Failed to download %s: %v", url, err)
		}
		return messages.NewImageMessage(matrix, evt.ID, evt.Sender, displayname, msgtype, hs, id, data, ts)
	}
	return nil
}

func getMembershipChangeMessage(evt *gomatrix.Event, membership, prevMembership, senderDisplayname, displayname, prevDisplayname string) (sender string, text tstring.TString) {
	switch membership {
	case "invite":
		sender = "---"
		text = tstring.NewColorTString(fmt.Sprintf("%s invited %s.", senderDisplayname, displayname), tcell.ColorGreen)
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		text.Colorize(len(senderDisplayname)+len(" invited "), len(displayname), widget.GetHashColor(*evt.StateKey))
	case "join":
		sender = "-->"
		text = tstring.NewColorTString(fmt.Sprintf("%s joined the room.", displayname), tcell.ColorGreen)
		text.Colorize(0, len(displayname), widget.GetHashColor(*evt.StateKey))
	case "leave":
		sender = "<--"
		if evt.Sender != *evt.StateKey {
			if prevMembership == "ban" {
				text = tstring.NewColorTString(fmt.Sprintf("%s unbanned %s", senderDisplayname, displayname), tcell.ColorGreen)
				text.Colorize(len(senderDisplayname)+len(" unbanned "), len(displayname), widget.GetHashColor(*evt.StateKey))
			} else {
				reason, _ := evt.Content["reason"].(string)
				text = tstring.NewColorTString(fmt.Sprintf("%s kicked %s: %s", senderDisplayname, displayname, reason), tcell.ColorRed)
				text.Colorize(len(senderDisplayname)+len(" kicked "), len(displayname), widget.GetHashColor(*evt.StateKey))
			}
			text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		} else {
			if displayname == *evt.StateKey {
				displayname = prevDisplayname
			}
			text = tstring.NewColorTString(fmt.Sprintf("%s left the room.", displayname), tcell.ColorRed)
			text.Colorize(0, len(displayname), widget.GetHashColor(*evt.StateKey))
		}
	case "ban":
		reason, _ := evt.Content["reason"].(string)
		text = tstring.NewColorTString(fmt.Sprintf("%s banned %s: %s", senderDisplayname, displayname, reason), tcell.ColorRed)
		text.Colorize(len(senderDisplayname)+len(" banned "), len(displayname), widget.GetHashColor(*evt.StateKey))
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
	}
	return
}

func getMembershipEventContent(room *rooms.Room, evt *gomatrix.Event) (sender string, text tstring.TString) {
	member := room.GetMember(evt.Sender)
	senderDisplayname := evt.Sender
	if member != nil {
		senderDisplayname = member.DisplayName
	}

	membership, _ := evt.Content["membership"].(string)
	displayname, _ := evt.Content["displayname"].(string)
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}

	prevMembership := "leave"
	prevDisplayname := *evt.StateKey
	if evt.Unsigned.PrevContent != nil {
		prevMembership, _ = evt.Unsigned.PrevContent["membership"].(string)
		prevDisplayname, _ = evt.Unsigned.PrevContent["displayname"].(string)
		if len(prevDisplayname) == 0 {
			prevDisplayname = *evt.StateKey
		}
	}

	if membership != prevMembership {
		sender, text = getMembershipChangeMessage(evt, membership, prevMembership, senderDisplayname, displayname, prevDisplayname)
	} else if displayname != prevDisplayname {
		sender = "---"
		text = tstring.NewColorTString(fmt.Sprintf("%s changed their display name to %s.", prevDisplayname, displayname), tcell.ColorGreen)
		color := widget.GetHashColor(*evt.StateKey)
		text.Colorize(0, len(prevDisplayname), color)
		text.Colorize(len(prevDisplayname) + len(" changed their display name to "), len(displayname), color)
	}
	return
}

func ParseMembershipEvent(room *rooms.Room, evt *gomatrix.Event) messages.UIMessage {
	displayname, text := getMembershipEventContent(room, evt)
	if len(text) == 0 {
		return nil
	}

	ts := unixToTime(evt.Timestamp)
	return messages.NewExpandedTextMessage(evt.ID, evt.Sender, displayname, "m.room.member", text, ts)
}
