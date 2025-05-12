// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package hicli contains a highly opinionated high-level framework for developing instant messaging clients on Matrix.
package hicli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	_ "go.mau.fi/util/dbutil/litestream"
	"go.mau.fi/util/exerrors"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type HiClient struct {
	DB          *database.Database
	CryptoDB    *dbutil.Database
	Account     *database.Account
	Client      *mautrix.Client
	Crypto      *crypto.OlmMachine
	CryptoStore *crypto.SQLCryptoStore
	ClientStore *database.ClientStateStore
	Log         zerolog.Logger

	Verified bool

	KeyBackupVersion id.KeyBackupVersion
	KeyBackupKey     *backup.MegolmBackupKey

	PushRules  atomic.Pointer[pushrules.PushRuleset]
	SyncStatus atomic.Pointer[jsoncmd.SyncStatus]
	syncErrors int
	lastSync   time.Time

	ToDeviceInSync atomic.Bool

	EventHandler func(evt any)
	LogoutFunc   func(context.Context) error

	firstSyncReceived bool
	syncingID         int
	syncLock          sync.Mutex
	stopSync          atomic.Pointer[context.CancelFunc]
	encryptLock       sync.Mutex
	loginLock         sync.Mutex

	requestQueueWakeup chan struct{}

	jsonRequestsLock sync.Mutex
	jsonRequests     map[int64]context.CancelCauseFunc

	paginationInterrupterLock sync.Mutex
	paginationInterrupter     map[id.RoomID]context.CancelCauseFunc
}

var (
	_ mautrix.StateStore        = (*database.ClientStateStore)(nil)
	_ mautrix.StateStoreUpdater = (*database.ClientStateStore)(nil)
	_ crypto.StateStore         = (*database.ClientStateStore)(nil)
)

var ErrTimelineReset = errors.New("got limited timeline sync response")

func New(rawDB, cryptoDB *dbutil.Database, log zerolog.Logger, pickleKey []byte, evtHandler func(any)) *HiClient {
	if cryptoDB == nil {
		cryptoDB = rawDB
	}
	if rawDB.Owner == "" {
		rawDB.Owner = "hicli"
		rawDB.IgnoreForeignTables = true
	}
	if rawDB.Log == nil {
		rawDB.Log = dbutil.ZeroLogger(log.With().Str("db_section", "hicli").Logger())
	}
	db := database.New(rawDB)
	c := &HiClient{
		DB:  db,
		Log: log,

		requestQueueWakeup:    make(chan struct{}, 1),
		jsonRequests:          make(map[int64]context.CancelCauseFunc),
		paginationInterrupter: make(map[id.RoomID]context.CancelCauseFunc),

		EventHandler: evtHandler,
	}
	if cryptoDB != rawDB {
		c.CryptoDB = cryptoDB
	}
	c.SyncStatus.Store(syncWaiting)
	c.ClientStore = &database.ClientStateStore{Database: db}
	c.Client = &mautrix.Client{
		UserAgent: mautrix.DefaultUserAgent,
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				// This needs to be relatively high to allow initial syncs,
				// it's lowered after the first sync in postProcessSyncResponse
				ResponseHeaderTimeout: 300 * time.Second,
				// Default settings from http.DefaultTransport
				Proxy:                 http.ProxyFromEnvironment,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          5,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 300 * time.Second,
		},
		Syncer:     (*hiSyncer)(c),
		Store:      (*hiStore)(c),
		StateStore: c.ClientStore,
		Log:        log.With().Str("component", "mautrix client").Logger(),

		DefaultHTTPBackoff: 1 * time.Second,
		DefaultHTTPRetries: 6,
	}
	c.CryptoStore = crypto.NewSQLCryptoStore(cryptoDB, dbutil.ZeroLogger(log.With().Str("db_section", "crypto").Logger()), "", "", pickleKey)
	cryptoLog := log.With().Str("component", "crypto").Logger()
	c.Crypto = crypto.NewOlmMachine(c.Client, &cryptoLog, c.CryptoStore, c.ClientStore)
	c.Crypto.SessionReceived = c.handleReceivedMegolmSession
	c.Crypto.DisableRatchetTracking = true
	c.Crypto.DisableDecryptKeyFetching = true
	c.Crypto.IgnorePostDecryptionParseErrors = true
	c.Client.Crypto = (*hiCryptoHelper)(c)
	return c
}

func (h *HiClient) tempClient(homeserverURL string) (*mautrix.Client, error) {
	parsedURL, err := url.Parse(homeserverURL)
	if err != nil {
		return nil, err
	}
	return &mautrix.Client{
		HomeserverURL: parsedURL,
		UserAgent:     h.Client.UserAgent,
		Client:        h.Client.Client,
		Log:           h.Log.With().Str("component", "temp mautrix client").Logger(),
	}, nil
}

func (h *HiClient) IsLoggedIn() bool {
	return h.Account != nil
}

func (h *HiClient) Start(ctx context.Context, userID id.UserID, expectedAccount *database.Account) error {
	if expectedAccount != nil && userID != expectedAccount.UserID {
		panic(fmt.Errorf("invalid parameters: different user ID in expected account and user ID"))
	}
	err := h.DB.Upgrade(ctx)
	if err != nil {
		return fmt.Errorf("failed to upgrade hicli db: %w", err)
	}
	err = h.CryptoStore.DB.Upgrade(ctx)
	if err != nil {
		return fmt.Errorf("failed to upgrade crypto db: %w", err)
	}
	account, err := h.DB.Account.Get(ctx, userID)
	if err != nil {
		return err
	} else if account == nil && expectedAccount != nil {
		err = h.DB.Account.Put(ctx, expectedAccount)
		if err != nil {
			return err
		}
		account = expectedAccount
	} else if expectedAccount != nil && expectedAccount.DeviceID != account.DeviceID {
		return fmt.Errorf("device ID mismatch: expected %s, got %s", expectedAccount.DeviceID, account.DeviceID)
	}
	if account != nil {
		zerolog.Ctx(ctx).Debug().Stringer("user_id", account.UserID).Msg("Preparing client with existing credentials")
		h.Account = account
		h.CryptoStore.AccountID = account.UserID.String()
		h.CryptoStore.DeviceID = account.DeviceID
		h.Client.UserID = account.UserID
		h.Client.DeviceID = account.DeviceID
		h.Client.AccessToken = account.AccessToken
		h.Client.HomeserverURL, err = url.Parse(account.HomeserverURL)
		if err != nil {
			return err
		}
		err = h.CheckServerVersions(ctx)
		if err != nil {
			return err
		}
		err = h.Crypto.Load(ctx)
		if err != nil {
			return fmt.Errorf("failed to load olm machine: %w", err)
		}

		h.Verified, err = h.checkIsCurrentDeviceVerified(ctx)
		if err != nil {
			return err
		}
		zerolog.Ctx(ctx).Debug().Bool("verified", h.Verified).Msg("Checked current device verification status")
		if h.Verified {
			err = h.loadPrivateKeys(ctx)
			if err != nil {
				return err
			}
			go h.Sync()
		}
	}
	return nil
}

var ErrFailedToCheckServerVersions = errors.New("failed to check server versions")
var ErrOutdatedServer = errors.New("homeserver is outdated")
var MinimumSpecVersion = mautrix.SpecV11

func (h *HiClient) CheckServerVersions(ctx context.Context) error {
	return h.checkServerVersions(ctx, h.Client)
}

func (h *HiClient) checkServerVersions(ctx context.Context, cli *mautrix.Client) error {
	versions, err := cli.Versions(ctx)
	if err != nil {
		return exerrors.NewDualError(ErrFailedToCheckServerVersions, err)
	} else if !versions.Contains(MinimumSpecVersion) {
		return fmt.Errorf("%w (minimum: %s, highest supported: %s)", ErrOutdatedServer, MinimumSpecVersion, versions.GetLatest())
	}
	return nil
}

func (h *HiClient) IsSyncing() bool {
	return h.stopSync.Load() != nil
}

func (h *HiClient) Sync() {
	h.Client.StopSync()
	if fn := h.stopSync.Load(); fn != nil {
		(*fn)()
	}
	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	h.syncingID++
	syncingID := h.syncingID
	log := h.Log.With().
		Str("action", "sync").
		Int("sync_id", syncingID).
		Logger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.stopSync.Store(&cancel)
	go h.RunRequestQueue(h.Log.WithContext(ctx))
	go h.LoadPushRules(h.Log.WithContext(ctx))
	ctx = log.WithContext(ctx)
	log.Info().Msg("Starting syncing")
	err := h.Client.SyncWithContext(ctx)
	if err != nil && ctx.Err() == nil {
		h.markSyncErrored(err, true)
		log.Err(err).Msg("Fatal error in syncer")
	} else {
		h.SyncStatus.Store(syncWaiting)
		log.Info().Msg("Syncing stopped")
	}
}

func (h *HiClient) Stop() {
	h.Client.StopSync()
	if fn := h.stopSync.Swap(nil); fn != nil {
		(*fn)()
	}
	h.syncLock.Lock()
	//lint:ignore SA2001 just acquire the lock to make sure Sync is done
	h.syncLock.Unlock()
	err := h.DB.Close()
	if err != nil {
		h.Log.Err(err).Msg("Failed to close database cleanly")
	}
	if h.CryptoDB != nil {
		err = h.CryptoDB.Close()
		if err != nil {
			h.Log.Err(err).Msg("Failed to close crypto database cleanly")
		}
	}
}
