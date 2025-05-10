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
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type PushNotification struct {
	Dismiss         []PushDismiss       `json:"dismiss,omitempty"`
	OrigMessages    []*PushNewMessage   `json:"-"`
	RawMessages     []json.RawMessage   `json:"messages,omitempty"`
	ImageAuth       string              `json:"image_auth,omitempty"`
	ImageAuthExpiry *jsontime.UnixMilli `json:"image_auth_expiry,omitempty"`
	HasImportant    bool                `json:"-"`
}

type PushDismiss struct {
	RoomID id.RoomID `json:"room_id"`
}

var pushClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	Timeout: 60 * time.Second,
}

func (gmx *Gomuks) SendPushNotifications(sync *jsoncmd.SyncComplete) {
	var ctx context.Context
	var push PushNotification
	for _, room := range sync.Rooms {
		if room.DismissNotifications && len(push.Dismiss) < 10 {
			push.Dismiss = append(push.Dismiss, PushDismiss{RoomID: room.Meta.ID})
		}
		for _, notif := range room.Notifications {
			if ctx == nil {
				ctx = gmx.Log.With().
					Str("action", "send push notification").
					Logger().WithContext(context.Background())
			}
			msg := gmx.formatPushNotificationMessage(ctx, notif)
			if msg == nil {
				continue
			}
			msgJSON, err := json.Marshal(msg)
			if err != nil {
				zerolog.Ctx(ctx).Err(err).
					Int64("event_rowid", int64(notif.RowID)).
					Stringer("event_id", notif.Event.ID).
					Msg("Failed to marshal push notification")
				continue
			} else if len(msgJSON) > 1500 {
				// This should not happen as long as formatPushNotificationMessage doesn't return too long messages
				zerolog.Ctx(ctx).Error().
					Int64("event_rowid", int64(notif.RowID)).
					Stringer("event_id", notif.Event.ID).
					Msg("Push notification too long")
				continue
			}
			push.RawMessages = append(push.RawMessages, msgJSON)
			push.OrigMessages = append(push.OrigMessages, msg)
		}
	}
	if len(push.Dismiss) == 0 && len(push.RawMessages) == 0 {
		return
	}
	if ctx == nil {
		ctx = gmx.Log.With().
			Str("action", "send push notification").
			Logger().WithContext(context.Background())
	}
	pushRegs, err := gmx.Client.DB.PushRegistration.GetAll(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get push registrations")
		return
	}
	if len(push.RawMessages) > 0 {
		exp := time.Now().Add(24 * time.Hour)
		push.ImageAuth = gmx.generateImageToken(24 * time.Hour)
		push.ImageAuthExpiry = ptr.Ptr(jsontime.UM(exp))
	}
	for notif := range push.Split {
		gmx.SendPushNotification(ctx, pushRegs, notif)
	}
}

func (pn *PushNotification) Split(yield func(*PushNotification) bool) {
	const maxSize = 2000
	currentSize := 0
	offset := 0
	hasSound := false
	for i, msg := range pn.RawMessages {
		if len(msg) >= maxSize {
			// This is already checked in SendPushNotifications, so this should never happen
			panic("push notification message too long")
		}
		if currentSize+len(msg) > maxSize {
			yield(&PushNotification{
				Dismiss:      pn.Dismiss,
				RawMessages:  pn.RawMessages[offset:i],
				ImageAuth:    pn.ImageAuth,
				HasImportant: hasSound,
			})
			offset = i
			currentSize = 0
			hasSound = false
		}
		currentSize += len(msg)
		hasSound = hasSound || pn.OrigMessages[i].Sound
	}
	yield(&PushNotification{
		Dismiss:      pn.Dismiss,
		RawMessages:  pn.RawMessages[offset:],
		ImageAuth:    pn.ImageAuth,
		HasImportant: hasSound,
	})
}

func (gmx *Gomuks) SendPushNotification(ctx context.Context, pushRegs []*database.PushRegistration, notif *PushNotification) {
	log := zerolog.Ctx(ctx).With().
		Bool("important", notif.HasImportant).
		Int("message_count", len(notif.RawMessages)).
		Int("dismiss_count", len(notif.Dismiss)).
		Logger()
	ctx = log.WithContext(ctx)
	rawPayload, err := json.Marshal(notif)
	if err != nil {
		log.Err(err).Msg("Failed to marshal push notification")
		return
	} else if base64.StdEncoding.EncodedLen(len(rawPayload)) >= 4000 {
		log.Error().Msg("Generated push payload too long")
		return
	}
	for _, reg := range pushRegs {
		devicePayload := rawPayload
		encrypted := false
		if reg.Encryption.Key != nil {
			var err error
			devicePayload, err = encryptPush(rawPayload, reg.Encryption.Key)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to encrypt push payload")
				continue
			}
			encrypted = true
		}
		switch reg.Type {
		case database.PushTypeFCM:
			if !encrypted {
				log.Warn().
					Str("device_id", reg.DeviceID).
					Msg("FCM push registration doesn't have encryption key")
				continue
			}
			var token string
			err = json.Unmarshal(reg.Data, &token)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to unmarshal FCM token")
				continue
			}
			gmx.SendFCMPush(ctx, token, devicePayload, notif.HasImportant)
		}
	}
}

func encryptPush(payload, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes long")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}
	iv := random.Bytes(12)
	encrypted := make([]byte, 12, 12+len(payload))
	copy(encrypted, iv)
	return gcm.Seal(encrypted, iv, payload, nil), nil
}

type PushRequest struct {
	Token        string `json:"token"`
	Payload      []byte `json:"payload"`
	HighPriority bool   `json:"high_priority"`
}

func (gmx *Gomuks) SendFCMPush(ctx context.Context, token string, payload []byte, highPriority bool) {
	wrappedPayload, _ := json.Marshal(&PushRequest{
		Token:        token,
		Payload:      payload,
		HighPriority: highPriority,
	})
	url := fmt.Sprintf("%s/_gomuks/push/fcm", gmx.Config.Push.FCMGateway)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(wrappedPayload))
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to create push request")
		return
	}
	resp, err := pushClient.Do(req)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("push_token", token).Msg("Failed to send push request")
	} else if resp.StatusCode != http.StatusOK {
		zerolog.Ctx(ctx).Error().
			Int("status", resp.StatusCode).
			Str("push_token", token).
			Msg("Non-200 status while sending push request")
	} else {
		zerolog.Ctx(ctx).Trace().
			Int("status", resp.StatusCode).
			Str("push_token", token).
			Msg("Sent push request")
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
}
