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
	"html"
	"strings"
	"time"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix"
	"maunium.net/go/tcell"
)

func ParseEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) messages.UIMessage {
	switch evt.Type {
	case mautrix.EventSticker:
		evt.Content.MsgType = mautrix.MsgImage
		fallthrough
	case mautrix.EventMessage:
		return ParseMessage(matrix, room, evt)
	case mautrix.StateTopic, mautrix.StateRoomName, mautrix.StateAliases, mautrix.StateCanonicalAlias:
		return ParseStateEvent(matrix, room, evt)
	case mautrix.StateMember:
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

func ParseStateEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) messages.UIMessage {
	displayname := evt.Sender
	member := room.GetMember(evt.Sender)
	if member != nil {
		displayname = member.Displayname
	}
	text := tstring.NewColorTString(displayname, widget.GetHashColor(evt.Sender))
	switch evt.Type {
	case mautrix.StateTopic:
		if len(evt.Content.Topic) == 0 {
			text = text.AppendColor(" removed the topic.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the topic to ", tcell.ColorGreen).
				AppendStyle(evt.Content.Topic, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case mautrix.StateRoomName:
		if len(evt.Content.Name) == 0 {
			text = text.AppendColor(" removed the room name.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the room name to ", tcell.ColorGreen).
				AppendStyle(evt.Content.Name, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case mautrix.StateCanonicalAlias:
		if len(evt.Content.Alias) == 0 {
			text = text.AppendColor(" removed the main address of the room.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the main address of the room to ", tcell.ColorGreen).
				AppendStyle(evt.Content.Alias, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case mautrix.StateAliases:
		text = ParseAliasEvent(evt, displayname)
	}
	ts := unixToTime(evt.Timestamp)
	return messages.NewExpandedTextMessage(evt.ID, evt.Sender, displayname, mautrix.MessageType(evt.Type.Type), text, ts)
}

func ParseMessage(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) messages.UIMessage {
	displayname := evt.Sender
	member := room.GetMember(evt.Sender)
	if member != nil {
		displayname = member.Displayname
	}
	if len(evt.Content.GetReplyTo()) > 0 {
		evt.Content.RemoveReplyFallback()
		replyToEvt, _ := matrix.Client().GetEvent(room.ID, evt.Content.GetReplyTo())
		replyToEvt.Content.RemoveReplyFallback()
		if len(replyToEvt.Content.FormattedBody) == 0 {
			replyToEvt.Content.FormattedBody = html.EscapeString(replyToEvt.Content.Body)
		}
		evt.Content.FormattedBody = fmt.Sprintf(
			"In reply to <a href='https://matrix.to/#/%[1]s'>%[1]s</a><blockquote>%[2]s</blockquote><br/>%[3]s",
			replyToEvt.Sender, replyToEvt.Content.FormattedBody, evt.Content.FormattedBody)
	}
	ts := unixToTime(evt.Timestamp)
	switch evt.Content.MsgType {
	case "m.text", "m.notice", "m.emote":
		if evt.Content.Format == mautrix.FormatHTML {
			text := ParseHTMLMessage(room, evt, displayname)
			return messages.NewExpandedTextMessage(evt.ID, evt.Sender, displayname, evt.Content.MsgType, text, ts)
		}
		evt.Content.Body = strings.Replace(evt.Content.Body, "\t", "    ", -1)
		return messages.NewTextMessage(evt.ID, evt.Sender, displayname, evt.Content.MsgType, evt.Content.Body, ts)
	case "m.image":
		data, hs, id, err := matrix.Download(evt.Content.URL)
		if err != nil {
			debug.Printf("Failed to download %s: %v", evt.Content.URL, err)
		}
		return messages.NewImageMessage(matrix, evt.ID, evt.Sender, displayname, evt.Content.MsgType, evt.Content.Body, hs, id, data, ts)
	}
	return nil
}

func getMembershipChangeMessage(evt *mautrix.Event, membership, prevMembership mautrix.Membership, senderDisplayname, displayname, prevDisplayname string) (sender string, text tstring.TString) {
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
			if prevMembership == mautrix.MembershipBan {
				text = tstring.NewColorTString(fmt.Sprintf("%s unbanned %s", senderDisplayname, displayname), tcell.ColorGreen)
				text.Colorize(len(senderDisplayname)+len(" unbanned "), len(displayname), widget.GetHashColor(*evt.StateKey))
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s kicked %s: %s", senderDisplayname, displayname, evt.Content.Reason), tcell.ColorRed)
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
		text = tstring.NewColorTString(fmt.Sprintf("%s banned %s: %s", senderDisplayname, displayname, evt.Content.Reason), tcell.ColorRed)
		text.Colorize(len(senderDisplayname)+len(" banned "), len(displayname), widget.GetHashColor(*evt.StateKey))
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
	}
	return
}

func getMembershipEventContent(room *rooms.Room, evt *mautrix.Event) (sender string, text tstring.TString) {
	member := room.GetMember(evt.Sender)
	senderDisplayname := evt.Sender
	if member != nil {
		senderDisplayname = member.Displayname
	}

	membership := evt.Content.Membership
	displayname := evt.Content.Displayname
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}

	prevMembership := mautrix.MembershipLeave
	prevDisplayname := *evt.StateKey
	if evt.Unsigned.PrevContent != nil {
		prevMembership = evt.Unsigned.PrevContent.Membership
		prevDisplayname = evt.Unsigned.PrevContent.Displayname
		if len(prevDisplayname) == 0 {
			prevDisplayname = *evt.StateKey
		}
	}

	if membership != prevMembership {
		sender, text = getMembershipChangeMessage(evt, membership, prevMembership, senderDisplayname, displayname, prevDisplayname)
	} else if displayname != prevDisplayname {
		sender = "---"
		color := widget.GetHashColor(*evt.StateKey)
		text = tstring.NewBlankTString().
			AppendColor(prevDisplayname, color).
			AppendColor(" changed their display name to ", tcell.ColorGreen).
			AppendColor(displayname, color).
			AppendColor(".", tcell.ColorGreen)
	}
	return
}

func ParseMembershipEvent(room *rooms.Room, evt *mautrix.Event) messages.UIMessage {
	displayname, text := getMembershipEventContent(room, evt)
	if len(text) == 0 {
		return nil
	}

	ts := unixToTime(evt.Timestamp)
	return messages.NewExpandedTextMessage(evt.ID, evt.Sender, displayname, "m.room.member", text, ts)
}

func ParseAliasEvent(evt *mautrix.Event, displayname string) tstring.TString {
	var prevAliases []string
	if evt.Unsigned.PrevContent != nil {
		prevAliases = evt.Unsigned.PrevContent.Aliases
	}
	aliases := evt.Content.Aliases
	var added, removed []tstring.TString
Outer1:
	for _, oldAlias := range prevAliases {
		for _, newAlias := range aliases {
			if oldAlias == newAlias {
				continue Outer1
			}
		}
		removed = append(removed, tstring.NewStyleTString(oldAlias, tcell.StyleDefault.Foreground(widget.GetHashColor(oldAlias)).Underline(true)))
	}
Outer2:
	for _, newAlias := range aliases {
		for _, oldAlias := range prevAliases {
			if oldAlias == newAlias {
				continue Outer2
			}
		}
		added = append(added, tstring.NewStyleTString(newAlias, tcell.StyleDefault.Foreground(widget.GetHashColor(newAlias)).Underline(true)))
	}
	var addedStr, removedStr tstring.TString
	if len(added) == 1 {
		addedStr = added[0]
	} else if len(added) > 1 {
		addedStr = tstring.
			Join(added[:len(added)-1], ", ").
			Append(" and ").
			AppendTString(added[len(added)-1])
	}
	if len(removed) == 1 {
		removedStr = removed[0]
	} else if len(removed) > 1 {
		removedStr = tstring.
			Join(removed[:len(removed)-1], ", ").
			Append(" and ").
			AppendTString(removed[len(removed)-1])
	}
	text := tstring.NewBlankTString()
	if len(addedStr) > 0 && len(removedStr) > 0 {
		text = text.AppendColor(fmt.Sprintf("%s added ", displayname), tcell.ColorGreen).
			AppendTString(addedStr).
			AppendColor(" and removed ", tcell.ColorGreen).
			AppendTString(removedStr).
			AppendColor(" as addresses for this room.", tcell.ColorGreen)
	} else if len(addedStr) > 0 {
		text = text.AppendColor(fmt.Sprintf("%s added ", displayname), tcell.ColorGreen).
			AppendTString(addedStr).
			AppendColor(" as addresses for this room.", tcell.ColorGreen)
	} else if len(removedStr) > 0 {
		text = text.AppendColor(fmt.Sprintf("%s removed ", displayname), tcell.ColorGreen).
			AppendTString(removedStr).
			AppendColor(" as addresses for this room.", tcell.ColorGreen)
	} else {
		return nil
	}
	return text
}
