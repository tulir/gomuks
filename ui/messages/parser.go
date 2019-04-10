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

package messages

import (
	"fmt"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/html"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
)

func getCachedEvent(mainView ifc.MainView, roomID, eventID string) UIMessage {
	if roomView := mainView.GetRoom(roomID); roomView != nil {
		if replyToIfcMsg := roomView.GetEvent(eventID); replyToIfcMsg != nil {
			if replyToMsg, ok := replyToIfcMsg.(UIMessage); ok && replyToMsg != nil {
				return replyToMsg
			}
		}
	}
	return nil
}

func ParseEvent(matrix ifc.MatrixContainer, mainView ifc.MainView, room *rooms.Room, evt *mautrix.Event) UIMessage {
	msg := directParseEvent(matrix, room, evt)
	if msg == nil {
		return nil
	}
	if len(evt.Content.GetReplyTo()) > 0 {
		replyToRoom := room
		if len(evt.Content.RelatesTo.InReplyTo.RoomID) > 0 {
			replyToRoom = matrix.GetRoom(evt.Content.RelatesTo.InReplyTo.RoomID)
		}

		if replyToMsg := getCachedEvent(mainView, replyToRoom.ID, evt.Content.GetReplyTo()); replyToMsg != nil {
			replyToMsg = replyToMsg.Clone()
			replyToMsg.SetReplyTo(nil)
			msg.SetReplyTo(replyToMsg)
		} else if replyToEvt, _ := matrix.GetEvent(replyToRoom, evt.Content.GetReplyTo()); replyToEvt != nil {
			if replyToMsg := directParseEvent(matrix, replyToRoom, replyToEvt); replyToMsg != nil {
				msg.SetReplyTo(replyToMsg)
			} else {
				// TODO add unrenderable reply header
			}
		} else {
			// TODO add unknown reply header
		}
	}
	return msg
}

func directParseEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) UIMessage {
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

func ParseStateEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) UIMessage {
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
	return NewExpandedTextMessage(evt, displayname, text)
}

func ParseMessage(matrix ifc.MatrixContainer, room *rooms.Room, evt *mautrix.Event) UIMessage {
	displayname := evt.Sender
	member := room.GetMember(evt.Sender)
	if member != nil {
		displayname = member.Displayname
	}
	if len(evt.Content.GetReplyTo()) > 0 {
		evt.Content.RemoveReplyFallback()
	}
	switch evt.Content.MsgType {
	case "m.text", "m.notice", "m.emote":
		if evt.Content.Format == mautrix.FormatHTML {
			return NewHTMLMessage(evt, displayname, html.Parse(room, evt, displayname))
		}
		evt.Content.Body = strings.Replace(evt.Content.Body, "\t", "    ", -1)
		return NewTextMessage(evt, displayname, evt.Content.Body)
	case "m.image":
		data, hs, id, err := matrix.Download(evt.Content.URL)
		if err != nil {
			debug.Printf("Failed to download %s: %v", evt.Content.URL, err)
		}
		return NewImageMessage(matrix, evt, displayname, evt.Content.Body, hs, id, data)
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

func ParseMembershipEvent(room *rooms.Room, evt *mautrix.Event) UIMessage {
	displayname, text := getMembershipEventContent(room, evt)
	if len(text) == 0 {
		return nil
	}

	return NewExpandedTextMessage(evt, displayname, text)
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
