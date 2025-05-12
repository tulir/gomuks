// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type ProfileDevice struct {
	DeviceID    id.DeviceID   `json:"device_id"`
	Name        string        `json:"name"`
	IdentityKey id.Curve25519 `json:"identity_key"`
	SigningKey  id.Ed25519    `json:"signing_key"`
	Fingerprint string        `json:"fingerprint"`
	Trust       id.TrustState `json:"trust_state"`
}

type ProfileEncryptionInfo struct {
	DevicesTracked bool             `json:"devices_tracked"`
	Devices        []*ProfileDevice `json:"devices"`
	MasterKey      string           `json:"master_key"`
	FirstMasterKey string           `json:"first_master_key"`
	UserTrusted    bool             `json:"user_trusted"`
	Errors         []string         `json:"errors"`
}

type PaginationResponse struct {
	Events        []*database.Event                  `json:"events"`
	Receipts      map[id.EventID][]*database.Receipt `json:"receipts"`
	RelatedEvents []*database.Event                  `json:"related_events"`
	HasMore       bool                               `json:"has_more"`
	FromServer    bool                               `json:"from_server"`
}
