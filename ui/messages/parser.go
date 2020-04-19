// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/html"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
)

func getCachedEvent(mainView ifc.MainView, roomID id.RoomID, eventID id.EventID) *UIMessage {
	if roomView := mainView.GetRoom(roomID); roomView != nil {
		if replyToIfcMsg := roomView.GetEvent(eventID); replyToIfcMsg != nil {
			if replyToMsg, ok := replyToIfcMsg.(*UIMessage); ok && replyToMsg != nil {
				return replyToMsg
			}
		}
	}
	return nil
}

func ParseEvent(matrix ifc.MatrixContainer, mainView ifc.MainView, room *rooms.Room, evt *muksevt.Event) *UIMessage {
	msg := directParseEvent(matrix, room, evt)
	if msg == nil {
		return nil
	}
	if content, ok := evt.Content.Parsed.(*event.MessageEventContent); ok && len(content.GetReplyTo()) > 0 {
		if replyToMsg := getCachedEvent(mainView, room.ID, content.GetReplyTo()); replyToMsg != nil {
			msg.ReplyTo = replyToMsg.Clone()
		} else if replyToEvt, _ := matrix.GetEvent(room, content.GetReplyTo()); replyToEvt != nil {
			if replyToMsg := directParseEvent(matrix, room, replyToEvt); replyToMsg != nil {
				msg.ReplyTo = replyToMsg
				msg.ReplyTo.Reactions = nil
			} else {
				// TODO add unrenderable reply header
			}
		} else {
			// TODO add unknown reply header
		}
	}
	return msg
}

func directParseEvent(matrix ifc.MatrixContainer, room *rooms.Room, evt *muksevt.Event) *UIMessage {
	displayname := string(evt.Sender)
	member := room.GetMember(evt.Sender)
	if member != nil {
		displayname = member.Displayname
	}
	if evt.Unsigned.RedactedBecause != nil || evt.Type == event.EventRedaction {
		return NewRedactedMessage(evt, displayname)
	}
	switch content := evt.Content.Parsed.(type) {
	case *event.MessageEventContent:
		if evt.Type == event.EventSticker {
			content.MsgType = event.MsgImage
		}
		return ParseMessage(matrix, room, evt, displayname)
	case *event.EncryptedEventContent:
		return NewExpandedTextMessage(evt, displayname, tstring.NewStyleTString("Encrypted messages are not yet supported", tcell.StyleDefault.Italic(true)))
	case *event.TopicEventContent, *event.RoomNameEventContent, *event.CanonicalAliasEventContent:
		return ParseStateEvent(evt, displayname)
	case *event.MemberEventContent:
		return ParseMembershipEvent(room, evt)
	default:
		debug.Printf("Unknown event content type %T in directParseEvent", content)
		return nil
	}
}

func ParseStateEvent(evt *muksevt.Event, displayname string) *UIMessage {
	text := tstring.NewColorTString(displayname, widget.GetHashColor(evt.Sender))
	switch content := evt.Content.Parsed.(type) {
	case *event.TopicEventContent:
		if len(content.Topic) == 0 {
			text = text.AppendColor(" removed the topic.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the topic to ", tcell.ColorGreen).
				AppendStyle(content.Topic, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.RoomNameEventContent:
		if len(content.Name) == 0 {
			text = text.AppendColor(" removed the room name.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the room name to ", tcell.ColorGreen).
				AppendStyle(content.Name, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.CanonicalAliasEventContent:
		if len(content.Alias) == 0 {
			text = text.AppendColor(" removed the main address of the room.", tcell.ColorGreen)
		} else {
			text = text.AppendColor(" changed the main address of the room to ", tcell.ColorGreen).
				AppendStyle(string(content.Alias), tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	//case event.StateAliases:
	//	text = ParseAliasEvent(evt, displayname)
	}
	return NewExpandedTextMessage(evt, displayname, text)
}

func ParseMessage(matrix ifc.MatrixContainer, room *rooms.Room, evt *muksevt.Event, displayname string) *UIMessage {
	content := evt.Content.AsMessage()
	if len(content.GetReplyTo()) > 0 {
		content.RemoveReplyFallback()
	}
	if len(evt.Gomuks.Edits) > 0 {
		content = evt.Gomuks.Edits[len(evt.Gomuks.Edits)-1].Content.AsMessage().NewContent
	}
	switch content.MsgType {
	case event.MsgText, event.MsgNotice, event.MsgEmote:
		if content.Format == event.FormatHTML {
			return NewHTMLMessage(evt, displayname, html.Parse(room, content, evt.Sender, displayname))
		}
		content.Body = strings.Replace(content.Body, "\t", "    ", -1)
		return NewTextMessage(evt, displayname, content.Body)
	case event.MsgImage, event.MsgVideo, event.MsgAudio, event.MsgFile:
		msg := NewFileMessage(matrix, evt, displayname)
		if !matrix.Preferences().DisableDownloads {
			renderer := msg.Renderer.(*FileMessage)
			renderer.DownloadPreview()
		}
		return msg
	}
	return nil
}

func getMembershipChangeMessage(evt *muksevt.Event, content *event.MemberEventContent, prevMembership event.Membership, senderDisplayname, displayname, prevDisplayname string) (sender string, text tstring.TString) {
	switch content.Membership {
	case "invite":
		sender = "---"
		text = tstring.NewColorTString(fmt.Sprintf("%s invited %s.", senderDisplayname, displayname), tcell.ColorGreen)
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		text.Colorize(len(senderDisplayname)+len(" invited "), len(displayname), widget.GetHashColor(evt.StateKey))
	case "join":
		sender = "-->"
		if prevMembership == event.MembershipInvite {
			text = tstring.NewColorTString(fmt.Sprintf("%s accepted the invite.", displayname), tcell.ColorGreen)
		} else {
			text = tstring.NewColorTString(fmt.Sprintf("%s joined the room.", displayname), tcell.ColorGreen)
		}
		text.Colorize(0, len(displayname), widget.GetHashColor(evt.StateKey))
	case "leave":
		sender = "<--"
		if evt.Sender != id.UserID(*evt.StateKey) {
			if prevMembership == event.MembershipBan {
				text = tstring.NewColorTString(fmt.Sprintf("%s unbanned %s", senderDisplayname, displayname), tcell.ColorGreen)
				text.Colorize(len(senderDisplayname)+len(" unbanned "), len(displayname), widget.GetHashColor(evt.StateKey))
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s kicked %s: %s", senderDisplayname, displayname, content.Reason), tcell.ColorRed)
				text.Colorize(len(senderDisplayname)+len(" kicked "), len(displayname), widget.GetHashColor(evt.StateKey))
			}
			text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		} else {
			if displayname == *evt.StateKey {
				displayname = prevDisplayname
			}
			if prevMembership == event.MembershipInvite {
				text = tstring.NewColorTString(fmt.Sprintf("%s rejected the invite.", displayname), tcell.ColorRed)
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s left the room.", displayname), tcell.ColorRed)
			}
			text.Colorize(0, len(displayname), widget.GetHashColor(evt.StateKey))
		}
	case "ban":
		text = tstring.NewColorTString(fmt.Sprintf("%s banned %s: %s", senderDisplayname, displayname, content.Reason), tcell.ColorRed)
		text.Colorize(len(senderDisplayname)+len(" banned "), len(displayname), widget.GetHashColor(evt.StateKey))
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
	}
	return
}

func getMembershipEventContent(room *rooms.Room, evt *muksevt.Event) (sender string, text tstring.TString) {
	member := room.GetMember(evt.Sender)
	senderDisplayname := string(evt.Sender)
	if member != nil {
		senderDisplayname = member.Displayname
	}

	content := evt.Content.AsMember()
	displayname := content.Displayname
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}

	prevMembership := event.MembershipLeave
	prevDisplayname := *evt.StateKey
	if evt.Unsigned.PrevContent != nil {
		prevContent := evt.Unsigned.PrevContent.AsMember()
		prevMembership = prevContent.Membership
		prevDisplayname = prevContent.Displayname
		if len(prevDisplayname) == 0 {
			prevDisplayname = *evt.StateKey
		}
	}

	if content.Membership != prevMembership {
		sender, text = getMembershipChangeMessage(evt, content, prevMembership, senderDisplayname, displayname, prevDisplayname)
	} else if displayname != prevDisplayname {
		sender = "---"
		color := widget.GetHashColor(evt.StateKey)
		text = tstring.NewBlankTString().
			AppendColor(prevDisplayname, color).
			AppendColor(" changed their display name to ", tcell.ColorGreen).
			AppendColor(displayname, color).
			AppendColor(".", tcell.ColorGreen)
	}
	return
}

func ParseMembershipEvent(room *rooms.Room, evt *muksevt.Event) *UIMessage {
	displayname, text := getMembershipEventContent(room, evt)
	if len(text) == 0 {
		return nil
	}

	return NewExpandedTextMessage(evt, displayname, text)
}

//func ParseAliasEvent(evt *muksevt.Event, displayname string) tstring.TString {
//	var prevAliases []string
//	if evt.Unsigned.PrevContent != nil {
//		prevAliases = evt.Unsigned.PrevContent.Aliases
//	}
//	aliases := evt.Content.Aliases
//	var added, removed []tstring.TString
//Outer1:
//	for _, oldAlias := range prevAliases {
//		for _, newAlias := range aliases {
//			if oldAlias == newAlias {
//				continue Outer1
//			}
//		}
//		removed = append(removed, tstring.NewStyleTString(oldAlias, tcell.StyleDefault.Foreground(widget.GetHashColor(oldAlias)).Underline(true)))
//	}
//Outer2:
//	for _, newAlias := range aliases {
//		for _, oldAlias := range prevAliases {
//			if oldAlias == newAlias {
//				continue Outer2
//			}
//		}
//		added = append(added, tstring.NewStyleTString(newAlias, tcell.StyleDefault.Foreground(widget.GetHashColor(newAlias)).Underline(true)))
//	}
//	var addedStr, removedStr tstring.TString
//	if len(added) == 1 {
//		addedStr = added[0]
//	} else if len(added) > 1 {
//		addedStr = tstring.
//			Join(added[:len(added)-1], ", ").
//			Append(" and ").
//			AppendTString(added[len(added)-1])
//	}
//	if len(removed) == 1 {
//		removedStr = removed[0]
//	} else if len(removed) > 1 {
//		removedStr = tstring.
//			Join(removed[:len(removed)-1], ", ").
//			Append(" and ").
//			AppendTString(removed[len(removed)-1])
//	}
//	text := tstring.NewBlankTString()
//	if len(addedStr) > 0 && len(removedStr) > 0 {
//		text = text.AppendColor(fmt.Sprintf("%s added ", displayname), tcell.ColorGreen).
//			AppendTString(addedStr).
//			AppendColor(" and removed ", tcell.ColorGreen).
//			AppendTString(removedStr).
//			AppendColor(" as addresses for this room.", tcell.ColorGreen)
//	} else if len(addedStr) > 0 {
//		text = text.AppendColor(fmt.Sprintf("%s added ", displayname), tcell.ColorGreen).
//			AppendTString(addedStr).
//			AppendColor(" as addresses for this room.", tcell.ColorGreen)
//	} else if len(removedStr) > 0 {
//		text = text.AppendColor(fmt.Sprintf("%s removed ", displayname), tcell.ColorGreen).
//			AppendTString(removedStr).
//			AppendColor(" as addresses for this room.", tcell.ColorGreen)
//	} else {
//		return nil
//	}
//	return text
//}
