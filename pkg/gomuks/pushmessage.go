// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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

package gomuks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"unicode/utf8"

	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type PushNewMessage struct {
	Timestamp  jsontime.UnixMilli  `json:"timestamp"`
	EventID    id.EventID          `json:"event_id"`
	EventRowID database.EventRowID `json:"event_rowid"`

	RoomID     id.RoomID        `json:"room_id"`
	RoomName   string           `json:"room_name"`
	RoomAvatar string           `json:"room_avatar,omitempty"`
	Sender     NotificationUser `json:"sender"`
	Self       NotificationUser `json:"self"`

	Text    string `json:"text"`
	Image   string `json:"image,omitempty"`
	Mention bool   `json:"mention,omitempty"`
	Reply   bool   `json:"reply,omitempty"`
	Sound   bool   `json:"sound,omitempty"`
}

type NotificationUser struct {
	ID     id.UserID `json:"id"`
	Name   string    `json:"name"`
	Avatar string    `json:"avatar,omitempty"`
}

func getAvatarLinkForNotification(name, ident string, uri id.ContentURIString) string {
	parsed := uri.ParseOrIgnore()
	if !parsed.IsValid() {
		return ""
	}
	var fallbackChar rune
	if name == "" {
		fallbackChar, _ = utf8.DecodeRuneInString(ident[1:])
	} else {
		fallbackChar, _ = utf8.DecodeRuneInString(name)
	}
	return fmt.Sprintf("_gomuks/media/%s/%s?encrypted=false&fallback=%s", parsed.Homeserver, parsed.FileID, url.QueryEscape(string(fallbackChar)))
}

func (gmx *Gomuks) getNotificationUser(ctx context.Context, roomID id.RoomID, userID id.UserID) (user NotificationUser) {
	user = NotificationUser{ID: userID, Name: userID.Localpart()}
	memberEvt, err := gmx.Client.DB.CurrentState.Get(ctx, roomID, event.StateMember, userID.String())
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Stringer("of_user_id", userID).Msg("Failed to get member event")
		return
	} else if memberEvt == nil {
		return
	}
	var memberContent event.MemberEventContent
	_ = json.Unmarshal(memberEvt.Content, &memberContent)
	if memberContent.Displayname != "" {
		user.Name = memberContent.Displayname
	}
	if len(user.Name) > 50 {
		user.Name = user.Name[:50] + "…"
	}
	if memberContent.AvatarURL != "" {
		user.Avatar = getAvatarLinkForNotification(memberContent.Displayname, userID.String(), memberContent.AvatarURL)
	}
	return
}

func (gmx *Gomuks) formatPushNotificationMessage(ctx context.Context, notif jsoncmd.SyncNotification) *PushNewMessage {
	evtType := notif.Event.Type
	rawContent := notif.Event.Content
	if evtType == event.EventEncrypted.Type {
		evtType = notif.Event.DecryptedType
		rawContent = notif.Event.Decrypted
	}
	if evtType != event.EventMessage.Type && evtType != event.EventSticker.Type {
		return nil
	}
	var content event.MessageEventContent
	err := json.Unmarshal(rawContent, &content)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).
			Stringer("event_id", notif.Event.ID).
			Msg("Failed to unmarshal message content to format push notification")
		return nil
	}
	var roomAvatar, image string
	if notif.Room.Avatar != nil {
		avatarIdent := notif.Room.ID.String()
		if ptr.Val(notif.Room.DMUserID) != "" {
			avatarIdent = notif.Room.DMUserID.String()
		}
		roomAvatar = getAvatarLinkForNotification(ptr.Val(notif.Room.Name), avatarIdent, notif.Room.Avatar.CUString())
	}
	roomName := ptr.Val(notif.Room.Name)
	if roomName == "" {
		roomName = "Unnamed room"
	}
	if len(roomName) > 50 {
		roomName = roomName[:50] + "…"
	}
	text := content.Body
	if len(text) > 400 {
		text = text[:350] + "[…]"
	}
	if content.MsgType == event.MsgImage || evtType == event.EventSticker.Type {
		if content.File != nil && content.File.URL != "" {
			parsed := content.File.URL.ParseOrIgnore()
			if len(content.File.URL) < 255 && parsed.IsValid() {
				image = fmt.Sprintf("_gomuks/media/%s/%s?encrypted=true", parsed.Homeserver, parsed.FileID)
			}
		} else if content.URL != "" {
			parsed := content.URL.ParseOrIgnore()
			if len(content.URL) < 255 && parsed.IsValid() {
				image = fmt.Sprintf("_gomuks/media/%s/%s?encrypted=false", parsed.Homeserver, parsed.FileID)
			}
		}
		if content.FileName == "" || content.FileName == content.Body {
			text = "Sent a photo"
		}
	} else if content.MsgType.IsMedia() {
		if content.FileName == "" || content.FileName == content.Body {
			text = "Sent a file: " + text
		}
	}
	return &PushNewMessage{
		Timestamp:  notif.Event.Timestamp,
		EventID:    notif.Event.ID,
		EventRowID: notif.Event.RowID,

		RoomID:     notif.Room.ID,
		RoomName:   roomName,
		RoomAvatar: roomAvatar,
		Sender:     gmx.getNotificationUser(ctx, notif.Room.ID, notif.Event.Sender),
		Self:       gmx.getNotificationUser(ctx, notif.Room.ID, gmx.Client.Account.UserID),

		Text:    text,
		Image:   image,
		Mention: content.Mentions.Has(gmx.Client.Account.UserID),
		Reply:   content.RelatesTo.GetNonFallbackReplyTo() != "",
		Sound:   notif.Sound,
	}
}
