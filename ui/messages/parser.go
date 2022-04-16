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

	"go.mau.fi/tcell"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/matrix/muksevt"
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
			if replyToMsg = directParseEvent(matrix, room, replyToEvt); replyToMsg != nil {
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
	case *muksevt.BadEncryptedContent:
		return NewExpandedTextMessage(evt, displayname, tstring.NewStyleTString(content.Reason, tcell.StyleDefault.Italic(true)))
	case *muksevt.EncryptionUnsupportedContent:
		return NewExpandedTextMessage(evt, displayname, tstring.NewStyleTString("gomuks not built with encryption support", tcell.StyleDefault.Italic(true)))
	case *event.TopicEventContent, *event.RoomNameEventContent, *event.CanonicalAliasEventContent:
		return ParseStateEvent(evt, displayname)
	case *event.MemberEventContent:
		return ParseMembershipEvent(room, evt)
	default:
		debug.Printf("Unknown event content type %T in directParseEvent", content)
		return nil
	}
}

func findAltAliasDifference(newList, oldList []id.RoomAlias) (addedStr, removedStr tstring.TString) {
	var addedList, removedList []tstring.TString
OldLoop:
	for _, oldAlias := range oldList {
		for _, newAlias := range newList {
			if oldAlias == newAlias {
				continue OldLoop
			}
		}
		removedList = append(removedList, tstring.NewStyleTString(string(oldAlias), tcell.StyleDefault.Foreground(widget.GetHashColor(oldAlias)).Underline(true)))
	}
NewLoop:
	for _, newAlias := range newList {
		for _, oldAlias := range oldList {
			if newAlias == oldAlias {
				continue NewLoop
			}
		}
		addedList = append(addedList, tstring.NewStyleTString(string(newAlias), tcell.StyleDefault.Foreground(widget.GetHashColor(newAlias)).Underline(true)))
	}
	if len(addedList) == 1 {
		addedStr = tstring.NewColorTString("added alternative address ", tcell.ColorGreen).AppendTString(addedList[0])
	} else if len(addedList) != 0 {
		addedStr = tstring.
			Join(addedList[:len(addedList)-1], ", ").
			PrependColor("added alternative addresses ", tcell.ColorGreen).
			AppendColor(" and ", tcell.ColorGreen).
			AppendTString(addedList[len(addedList)-1])
	}
	if len(removedList) == 1 {
		removedStr = tstring.NewColorTString("removed alternative address ", tcell.ColorGreen).AppendTString(removedList[0])
	} else if len(removedList) != 0 {
		removedStr = tstring.
			Join(removedList[:len(removedList)-1], ", ").
			PrependColor("removed alternative addresses ", tcell.ColorGreen).
			AppendColor(" and ", tcell.ColorGreen).
			AppendTString(removedList[len(removedList)-1])
	}
	return
}

func ParseStateEvent(evt *muksevt.Event, displayname string) *UIMessage {
	text := tstring.NewColorTString(displayname, widget.GetHashColor(evt.Sender)).Append(" ")
	switch content := evt.Content.Parsed.(type) {
	case *event.TopicEventContent:
		if len(content.Topic) == 0 {
			text = text.AppendColor("removed the topic.", tcell.ColorGreen)
		} else {
			text = text.AppendColor("changed the topic to ", tcell.ColorGreen).
				AppendStyle(content.Topic, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.RoomNameEventContent:
		if len(content.Name) == 0 {
			text = text.AppendColor("removed the room name.", tcell.ColorGreen)
		} else {
			text = text.AppendColor("changed the room name to ", tcell.ColorGreen).
				AppendStyle(content.Name, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.CanonicalAliasEventContent:
		prevContent := &event.CanonicalAliasEventContent{}
		if evt.Unsigned.PrevContent != nil {
			_ = evt.Unsigned.PrevContent.ParseRaw(evt.Type)
			prevContent = evt.Unsigned.PrevContent.AsCanonicalAlias()
		}
		debug.Printf("%+v -> %+v", prevContent, content)
		if len(content.Alias) == 0 && len(prevContent.Alias) != 0 {
			text = text.AppendColor("removed the main address of the room", tcell.ColorGreen)
		} else if content.Alias != prevContent.Alias {
			text = text.
				AppendColor("changed the main address of the room to ", tcell.ColorGreen).
				AppendStyle(string(content.Alias), tcell.StyleDefault.Underline(true))
		} else {
			added, removed := findAltAliasDifference(content.AltAliases, prevContent.AltAliases)
			if len(added) > 0 {
				if len(removed) > 0 {
					text = text.
						AppendTString(added).
						AppendColor(" and ", tcell.ColorGreen).
						AppendTString(removed)
				} else {
					text = text.AppendTString(added)
				}
			} else if len(removed) > 0 {
				text = text.AppendTString(removed)
			} else {
				text = text.AppendColor("changed nothing", tcell.ColorGreen)
			}
			text = text.AppendColor(" for this room", tcell.ColorGreen)
		}
	}
	return NewExpandedTextMessage(evt, displayname, text)
}

func ParseMessage(matrix ifc.MatrixContainer, room *rooms.Room, evt *muksevt.Event, displayname string) *UIMessage {
	content := evt.Content.AsMessage()
	if len(content.GetReplyTo()) > 0 {
		content.RemoveReplyFallback()
	}
	if len(evt.Gomuks.Edits) > 0 {
		newContent := evt.Gomuks.Edits[len(evt.Gomuks.Edits)-1].Content.AsMessage().NewContent
		if newContent != nil {
			content = newContent
		}
	}
	switch content.MsgType {
	case event.MsgText, event.MsgNotice, event.MsgEmote:
		if content.Format == event.FormatHTML {
			return NewHTMLMessage(evt, displayname, html.Parse(matrix.Preferences(), room, content, evt, displayname))
		}
		content.Body = strings.Replace(content.Body, "\t", "    ", -1)
		return NewHTMLMessage(evt, displayname, html.TextToEntity(content.Body, evt.ID, matrix.Preferences().EnableInlineURLs()))
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
		_ = evt.Unsigned.PrevContent.ParseRaw(evt.Type)
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
