// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/crypto/ssss"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func (h *HiClient) checkIsCurrentDeviceVerified(ctx context.Context) (bool, error) {
	keys := h.Crypto.GetOwnCrossSigningPublicKeys(ctx)
	if keys == nil {
		return false, fmt.Errorf("own cross-signing keys not found")
	}
	isVerified, err := h.Crypto.CryptoStore.IsKeySignedBy(ctx, h.Account.UserID, h.Crypto.GetAccount().SigningKey(), h.Account.UserID, keys.SelfSigningKey)
	if err != nil {
		return false, fmt.Errorf("failed to check if current device is signed by own self-signing key: %w", err)
	}
	return isVerified, nil
}

func (h *HiClient) fetchKeyBackupKey(ctx context.Context, ssssKey *ssss.Key) error {
	latestVersion, err := h.Client.GetKeyBackupLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get key backup latest version: %w", err)
	}
	h.KeyBackupVersion = latestVersion.Version
	data, err := h.Crypto.SSSS.GetDecryptedAccountData(ctx, event.AccountDataMegolmBackupKey, ssssKey)
	if err != nil {
		return fmt.Errorf("failed to get megolm backup key from SSSS: %w", err)
	}
	key, err := backup.MegolmBackupKeyFromBytes(data)
	if err != nil {
		return fmt.Errorf("failed to parse megolm backup key: %w", err)
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretMegolmBackupV1, base64.StdEncoding.EncodeToString(key.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to store megolm backup key: %w", err)
	}
	h.KeyBackupKey = key
	return nil
}

func (h *HiClient) getAndDecodeSecret(ctx context.Context, secret id.Secret) ([]byte, error) {
	secretData, err := h.CryptoStore.GetSecret(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secret, err)
	} else if secretData == "" {
		return nil, fmt.Errorf("secret %s not found", secret)
	}
	data, err := base64.StdEncoding.DecodeString(secretData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret %s: %w", secret, err)
	}
	return data, nil
}

func (h *HiClient) loadPrivateKeys(ctx context.Context) error {
	zerolog.Ctx(ctx).Debug().Msg("Loading cross-signing private keys")
	masterKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSMaster)
	if err != nil {
		return fmt.Errorf("failed to get master key: %w", err)
	}
	selfSigningKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSSelfSigning)
	if err != nil {
		return fmt.Errorf("failed to get self-signing key: %w", err)
	}
	userSigningKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSUserSigning)
	if err != nil {
		return fmt.Errorf("failed to get user signing key: %w", err)
	}
	err = h.Crypto.ImportCrossSigningKeys(crypto.CrossSigningSeeds{
		MasterKey:      masterKeySeed,
		SelfSigningKey: selfSigningKeySeed,
		UserSigningKey: userSigningKeySeed,
	})
	if err != nil {
		return fmt.Errorf("failed to import cross-signing private keys: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("Loading key backup key")
	keyBackupKey, err := h.getAndDecodeSecret(ctx, id.SecretMegolmBackupV1)
	if err != nil {
		return fmt.Errorf("failed to get megolm backup key: %w", err)
	}
	h.KeyBackupKey, err = backup.MegolmBackupKeyFromBytes(keyBackupKey)
	if err != nil {
		return fmt.Errorf("failed to parse megolm backup key: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("Fetching key backup version")
	latestVersion, err := h.Client.GetKeyBackupLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get key backup latest version: %w", err)
	}
	h.KeyBackupVersion = latestVersion.Version
	zerolog.Ctx(ctx).Debug().Msg("Secrets loaded")
	return nil
}

func (h *HiClient) storeCrossSigningPrivateKeys(ctx context.Context) error {
	keys := h.Crypto.CrossSigningKeys
	err := h.CryptoStore.PutSecret(ctx, id.SecretXSMaster, base64.StdEncoding.EncodeToString(keys.MasterKey.Seed()))
	if err != nil {
		return err
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretXSSelfSigning, base64.StdEncoding.EncodeToString(keys.SelfSigningKey.Seed()))
	if err != nil {
		return err
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretXSUserSigning, base64.StdEncoding.EncodeToString(keys.UserSigningKey.Seed()))
	if err != nil {
		return err
	}
	return nil
}

func (h *HiClient) Verify(ctx context.Context, code string) error {
	defer h.dispatchCurrentState()
	keyID, keyData, err := h.Crypto.SSSS.GetDefaultKeyData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default SSSS key data: %w", err)
	}
	key, err := keyData.VerifyRecoveryKey(keyID, code)
	if errors.Is(err, ssss.ErrInvalidRecoveryKey) && keyData.Passphrase != nil {
		key, err = keyData.VerifyPassphrase(keyID, code)
	}
	if err != nil {
		return err
	}
	err = h.Crypto.FetchCrossSigningKeysFromSSSS(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to fetch cross-signing keys from SSSS: %w", err)
	}
	err = h.Crypto.SignOwnDevice(ctx, h.Crypto.OwnIdentity())
	if err != nil {
		return fmt.Errorf("failed to sign own device: %w", err)
	}
	err = h.Crypto.SignOwnMasterKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to sign own master key: %w", err)
	}
	err = h.storeCrossSigningPrivateKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to store cross-signing private keys: %w", err)
	}
	err = h.fetchKeyBackupKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to fetch key backup key: %w", err)
	}
	h.Verified = true
	if !h.IsSyncing() {
		go h.Sync()
	}
	return nil
}
