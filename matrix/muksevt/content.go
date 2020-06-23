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

package muksevt

import (
	"encoding/gob"
	"reflect"

	"maunium.net/go/mautrix/event"
)

var EventBadEncrypted = event.Type{Type: "net.maunium.gomuks.bad_encrypted", Class: event.MessageEventType}
var EventEncryptionUnsupported = event.Type{Type: "net.maunium.gomuks.encryption_unsupported", Class: event.MessageEventType}

type BadEncryptedContent struct {
	Original *event.EncryptedEventContent `json:"-"`

	Reason string `json:"-"`
}

type EncryptionUnsupportedContent struct {
	Original *event.EncryptedEventContent `json:"-"`
}

func init() {
	gob.Register(&BadEncryptedContent{})
	gob.Register(&EncryptionUnsupportedContent{})
	event.TypeMap[EventBadEncrypted] = reflect.TypeOf(&BadEncryptedContent{})
	event.TypeMap[EventEncryptionUnsupported] = reflect.TypeOf(&EncryptionUnsupportedContent{})
}
