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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/exhttp"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli"
)

func (gmx *Gomuks) ExportKeys(w http.ResponseWriter, r *http.Request) {
	found, correct := gmx.doBasicAuth(r)
	if !found || !correct {
		hlog.FromRequest(r).Debug().Msg("Requesting credentials for key export request")
		w.Header().Set("WWW-Authenticate", `Basic realm="gomuks web" charset="UTF-8"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-store")
	err := r.ParseForm()
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to parse form")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Failed to parse form data\n"))
		return
	}
	roomID := id.RoomID(r.PathValue("room_id"))
	var sessions dbutil.RowIter[*crypto.InboundGroupSession]
	filename := "gomuks-keys.txt"
	if roomID == "" {
		sessions = gmx.Client.CryptoStore.GetAllGroupSessions(r.Context())
	} else {
		filename = fmt.Sprintf("gomuks-keys-%s.txt", roomID)
		sessions = gmx.Client.CryptoStore.GetGroupSessionsForRoom(r.Context(), roomID)
	}
	export, err := crypto.ExportKeysIter(r.FormValue("passphrase"), sessions)
	if errors.Is(err, crypto.ErrNoSessionsForExport) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("No keys found\n"))
		return
	} else if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to export keys")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Failed to export keys (see logs for more details)\n"))
		return
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", strconv.Itoa(len(export)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(export)
}

var badMultipartForm = mautrix.RespError{ErrCode: "FI.MAU.GOMUKS.BAD_FORM_DATA", Err: "Failed to parse form data", StatusCode: http.StatusBadRequest}

func (gmx *Gomuks) ImportKeys(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(5 * 1024 * 1024)
	if err != nil {
		badMultipartForm.Write(w)
		return
	}
	export, _, err := r.FormFile("export")
	if err != nil {
		badMultipartForm.WithMessage("Failed to get export file from form: %w", err).Write(w)
		return
	}
	exportData, err := io.ReadAll(export)
	if err != nil {
		badMultipartForm.WithMessage("Failed to read export file: %w", err).Write(w)
		return
	}
	importedCount, totalCount, err := gmx.Client.Crypto.ImportKeys(r.Context(), r.FormValue("passphrase"), exportData)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to import keys")
		mautrix.MUnknown.WithMessage("Failed to import keys: %w", err).Write(w)
		return
	}
	hlog.FromRequest(r).Info().
		Int("imported_count", importedCount).
		Int("total_count", totalCount).
		Msg("Successfully imported keys")
	exhttp.WriteJSONResponse(w, http.StatusOK, map[string]int{
		"imported": importedCount,
		"total":    totalCount,
	})
}

func (gmx *Gomuks) RestoreKeyBackup(w http.ResponseWriter, r *http.Request) {
	roomID := id.RoomID(r.PathValue("room_id"))
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	sendProgress := func(progress hicli.KeyBackupRestoreProgress) {
		progressJSON, err := json.Marshal(progress)
		if err != nil {
			zerolog.Ctx(r.Context()).Err(err).Msg("Failed to marshal progress notice")
			return
		}
		_, err = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", progressJSON)
		if err != nil {
			zerolog.Ctx(r.Context()).Err(err).Msg("Failed to write progress notice")
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	err := gmx.Client.RestoreKeyBackup(r.Context(), roomID, sendProgress)
	if err != nil {
		_, _ = fmt.Fprintf(w, "event: done\ndata: %s\n\n", err.Error())
	} else {
		_, _ = fmt.Fprint(w, "event: done\ndata: ok\n\n")
	}
}
