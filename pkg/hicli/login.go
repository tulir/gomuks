// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"fmt"
	"net/url"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

var InitialDeviceDisplayName = "mautrix hiclient"

func (h *HiClient) LoginPassword(ctx context.Context, homeserverURL, username, password string) error {
	var err error
	h.Client.HomeserverURL, err = url.Parse(homeserverURL)
	if err != nil {
		return err
	}
	return h.Login(ctx, &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password: password,
	})
}

func (h *HiClient) Login(ctx context.Context, req *mautrix.ReqLogin) error {
	h.loginLock.Lock()
	defer h.loginLock.Unlock()
	if h.IsLoggedIn() {
		return fmt.Errorf("already logged in")
	}

	err := h.CheckServerVersions(ctx)
	if err != nil {
		return err
	}
	req.InitialDeviceDisplayName = InitialDeviceDisplayName
	req.StoreCredentials = true
	req.StoreHomeserverURL = true
	resp, err := h.Client.Login(ctx, req)
	if err != nil {
		return err
	}
	defer h.dispatchCurrentState()
	h.Account = &database.Account{
		UserID:        resp.UserID,
		DeviceID:      resp.DeviceID,
		AccessToken:   resp.AccessToken,
		HomeserverURL: h.Client.HomeserverURL.String(),
	}
	h.CryptoStore.AccountID = resp.UserID.String()
	h.CryptoStore.DeviceID = resp.DeviceID
	err = h.DB.Account.Put(ctx, h.Account)
	if err != nil {
		return err
	}
	err = h.Crypto.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load olm machine: %w", err)
	}
	err = h.Crypto.ShareKeys(ctx, 0)
	if err != nil {
		return err
	}
	_, err = h.Crypto.FetchKeys(ctx, []id.UserID{h.Account.UserID}, true)
	if err != nil {
		return fmt.Errorf("failed to fetch own devices: %w", err)
	}
	return nil
}

func (h *HiClient) LoginAndVerify(ctx context.Context, homeserverURL, username, password, recoveryKey string) error {
	err := h.LoginPassword(ctx, homeserverURL, username, password)
	if err != nil {
		return err
	}
	err = h.Verify(ctx, recoveryKey)
	if err != nil {
		return err
	}
	return nil
}
