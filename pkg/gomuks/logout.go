// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
	"errors"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
)

func (gmx *Gomuks) Logout(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("Stopping client and logging out")
	gmx.Client.Stop()
	_, err := gmx.Client.Client.Logout(ctx)
	if err != nil && !errors.Is(err, mautrix.MUnknownToken) {
		log.Warn().Err(err).Msg("Failed to log out")
		return err
	}
	log.Info().Msg("Logout complete, removing data")
	err = os.RemoveAll(gmx.CacheDir)
	if err != nil {
		log.Err(err).Str("cache_dir", gmx.CacheDir).Msg("Failed to remove cache dir")
	}
	if gmx.DataDir == gmx.ConfigDir {
		err = os.Remove(filepath.Join(gmx.DataDir, "gomuks.db"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Err(err).Str("data_dir", gmx.DataDir).Msg("Failed to remove database")
		}
		_ = os.Remove(filepath.Join(gmx.DataDir, "gomuks.db-shm"))
		_ = os.Remove(filepath.Join(gmx.DataDir, "gomuks.db-wal"))
	} else {
		err = os.RemoveAll(gmx.DataDir)
		if err != nil {
			log.Err(err).Str("data_dir", gmx.DataDir).Msg("Failed to remove data dir")
		}
	}
	log.Info().Msg("Re-initializing directories")
	gmx.InitDirectories()
	log.Info().Msg("Restarting client")
	gmx.StartClient()
	gmx.Client.EventHandler(gmx.Client.State())
	gmx.Client.EventHandler(gmx.Client.SyncStatus.Load())
	log.Info().Msg("Client restarted")
	return nil
}
