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

//go:build cgo

package matrix

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/dbutil"

	"maunium.net/go/gomuks/debug"
)

type cryptoLogger struct {
	prefix string
}

func (c cryptoLogger) Error(message string, args ...interface{}) {
	debug.Printf(fmt.Sprintf("[%s/Error] %s", c.prefix, message), args...)
}

func (c cryptoLogger) Warn(message string, args ...interface{}) {
	debug.Printf(fmt.Sprintf("[%s/Warn] %s", c.prefix, message), args...)
}

func (c cryptoLogger) Debug(message string, args ...interface{}) {
	debug.Printf(fmt.Sprintf("[%s/Debug] %s", c.prefix, message), args...)
}

func (c cryptoLogger) Trace(message string, args ...interface{}) {
	debug.Printf(fmt.Sprintf("[%s/Trace] %s", c.prefix, message), args...)
}

func (c cryptoLogger) WarnUnsupportedVersion(current, latest int) {
	c.Warn("Unsupported database schema version: currently on v%d, latest known: v%d - continuing anyway", current, latest)
}

func (c cryptoLogger) PrepareUpgrade(current, latest int) {
	c.Debug("Database currently on v%d, latest: v%d", current, latest)
}

func (c cryptoLogger) DoUpgrade(from, to int, message string) {
	c.Debug("Upgrading database from v%d to v%d: %s", from, to, message)
}

func (c cryptoLogger) QueryTiming(_ context.Context, method, query string, _ []interface{}, duration time.Duration) {
	if duration > 1*time.Second {
		c.Warn("%s(%s) took %.3f seconds", method, query, duration.Seconds())
	}
}

func isBadEncryptError(err error) bool {
	return err != crypto.SessionExpired && err != crypto.SessionNotShared && err != crypto.NoGroupSession
}

func (c *Container) initCrypto() error {
	var cryptoStore crypto.Store
	var err error
	legacyStorePath := filepath.Join(c.config.DataDir, "crypto.gob")
	if _, err = os.Stat(legacyStorePath); err == nil {
		debug.Printf("Using legacy crypto store as %s exists", legacyStorePath)
		cryptoStore, err = crypto.NewGobStore(legacyStorePath)
		if err != nil {
			return fmt.Errorf("file open: %w", err)
		}
	} else {
		debug.Printf("Using SQLite crypto store")
		newStorePath := filepath.Join(c.config.DataDir, "crypto.db")
		db, err := sql.Open("sqlite3", newStorePath)
		if err != nil {
			return fmt.Errorf("sql open: %w", err)
		}
		mauDb, err := dbutil.NewWithDB(db, "sqlite3")
		if err != nil {
			return fmt.Errorf("mautrix dbutils instanciation: %w", err)
		}
		accID := fmt.Sprintf("%s/%s", c.config.UserID.String(), c.config.DeviceID)
		sqlStore := crypto.NewSQLCryptoStore(mauDb, cryptoLogger{"Crypto/DB"}, accID, c.config.DeviceID, []byte("fi.mau.gomuks"))
		err = sqlStore.Upgrade()
		if err != nil {
			return fmt.Errorf("create table: %w", err)
		}
		cryptoStore = sqlStore
	}
	crypt := crypto.NewOlmMachine(c.client, cryptoLogger{"Crypto"}, cryptoStore, c.config.Rooms)
	if c.config.SendToVerifiedOnly {
		crypt.SendKeysMinTrust = id.TrustStateCrossSignedVerified
	}
	c.crypto = crypt
	err = c.crypto.Load()
	if err != nil {
		return fmt.Errorf("failed to create olm machine: %w", err)
	}
	return nil
}

func (c *Container) cryptoOnLogin() {
	sqlStore, ok := c.crypto.(*crypto.OlmMachine).CryptoStore.(*crypto.SQLCryptoStore)
	if !ok {
		return
	}
	sqlStore.DeviceID = c.config.DeviceID
	sqlStore.AccountID = fmt.Sprintf("%s/%s", c.config.UserID.String(), c.config.DeviceID)
}
