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

// +build cgo

package matrix

import (
	"path/filepath"

	"github.com/pkg/errors"

	"maunium.net/go/mautrix/crypto"

	"maunium.net/go/gomuks/debug"
)

type cryptoLogger struct{}

func (c cryptoLogger) Error(message string, args ...interface{}) {
	debug.Printf("[Crypto/Error] "+message, args...)
}

func (c cryptoLogger) Warn(message string, args ...interface{}) {
	debug.Printf("[Crypto/Warn] "+message, args...)
}

func (c cryptoLogger) Debug(message string, args ...interface{}) {
	debug.Printf("[Crypto/Debug] "+message, args...)
}

func (c cryptoLogger) Trace(message string, args ...interface{}) {
	debug.Printf("[Crypto/Trace] "+message, args...)
}

func isBadEncryptError(err error) bool {
	return err != crypto.SessionExpired && err != crypto.SessionNotShared && err != crypto.NoGroupSession
}

func (c *Container) initCrypto() error {
	cryptoStore, err := crypto.NewGobStore(filepath.Join(c.config.DataDir, "crypto.gob"))
	if err != nil {
		return errors.Wrap(err, "failed to open crypto store")
	}
	crypt := crypto.NewOlmMachine(c.client, cryptoLogger{}, cryptoStore, c.config.Rooms)
	crypt.AllowUnverifiedDevices = !c.config.SendToVerifiedOnly
	c.crypto = crypt
	err = c.crypto.Load()
	if err != nil {
		return errors.Wrap(err, "failed to create olm machine")
	}
	return nil
}
