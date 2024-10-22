// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"time"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/id"
)

const (
	insertMediaQuery = `
		INSERT INTO media (mxc, enc_file, file_name, mime_type, size, hash, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (mxc) DO NOTHING
	`
	upsertMediaQuery = `
		INSERT INTO media (mxc, enc_file, file_name, mime_type, size, hash, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (mxc) DO UPDATE
			SET enc_file = COALESCE(excluded.enc_file, media.enc_file),
			    file_name = COALESCE(excluded.file_name, media.file_name),
			    mime_type = COALESCE(excluded.mime_type, media.mime_type),
			    size = COALESCE(excluded.size, media.size),
			    hash = COALESCE(excluded.hash, media.hash),
			    error = excluded.error
			WHERE excluded.error IS NULL OR media.hash IS NULL
	`
	getMediaQuery = `
		SELECT mxc, enc_file, file_name, mime_type, size, hash, error
		FROM media
		WHERE mxc = $1
	`
	addMediaReferenceQuery = `
		INSERT INTO media_reference (event_rowid, media_mxc)
		VALUES ($1, $2)
		ON CONFLICT (event_rowid, media_mxc) DO NOTHING
	`
)

type MediaQuery struct {
	*dbutil.QueryHelper[*Media]
}

func (mq *MediaQuery) Add(ctx context.Context, cm *Media) error {
	return mq.Exec(ctx, insertMediaQuery, cm.sqlVariables()...)
}

func (mq *MediaQuery) AddReference(ctx context.Context, evtRowID EventRowID, mxc id.ContentURI) error {
	return mq.Exec(ctx, addMediaReferenceQuery, evtRowID, &mxc)
}

func (mq *MediaQuery) Put(ctx context.Context, cm *Media) error {
	return mq.Exec(ctx, upsertMediaQuery, cm.sqlVariables()...)
}

func (mq *MediaQuery) Get(ctx context.Context, mxc id.ContentURI) (*Media, error) {
	return mq.QueryOne(ctx, getMediaQuery, &mxc)
}

type MediaError struct {
	Matrix     *mautrix.RespError `json:"data"`
	StatusCode int                `json:"status_code"`
	ReceivedAt jsontime.UnixMilli `json:"received_at"`
	Attempts   int                `json:"attempts"`
}

const MaxMediaBackoff = 7 * 24 * time.Hour

func (me *MediaError) backoff() time.Duration {
	return min(time.Duration(2<<me.Attempts)*time.Second, MaxMediaBackoff)
}

func (me *MediaError) UseCache() bool {
	return me != nil && time.Since(me.ReceivedAt.Time) < me.backoff()
}

func (me *MediaError) Write(w http.ResponseWriter) {
	if me.Matrix.ExtraData == nil {
		me.Matrix.ExtraData = make(map[string]any)
	}
	me.Matrix.ExtraData["fi.mau.hicli.error_ts"] = me.ReceivedAt.UnixMilli()
	me.Matrix.ExtraData["fi.mau.hicli.next_retry_ts"] = me.ReceivedAt.Add(me.backoff()).UnixMilli()
	w.Header().Set("Mau-Errored-At", me.ReceivedAt.Format(http.TimeFormat))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", max(int(time.Until(me.ReceivedAt.Add(me.backoff())).Seconds()), 0)))
	me.Matrix.WithStatus(me.StatusCode).Write(w)
}

type Media struct {
	MXC      id.ContentURI
	EncFile  *attachment.EncryptedFile
	FileName string
	MimeType string
	Size     int64
	Hash     *[32]byte
	Error    *MediaError
}

func (m *Media) ETag() string {
	if m.Hash == nil {
		return ""
	}
	return fmt.Sprintf(`"%x"`, m.Hash)
}

func (m *Media) UseCache() bool {
	return m != nil && (m.Hash != nil || m.Error.UseCache())
}

func (m *Media) sqlVariables() []any {
	var hash []byte
	if m.Hash != nil {
		hash = m.Hash[:]
	}
	return []any{
		&m.MXC, dbutil.JSONPtr(m.EncFile),
		dbutil.StrPtr(m.FileName), dbutil.StrPtr(m.MimeType), dbutil.NumPtr(m.Size),
		hash, dbutil.JSONPtr(m.Error),
	}
}

var safeMimes = []string{
	"text/css", "text/plain", "text/csv",
	"application/json", "application/ld+json",
	"image/jpeg", "image/gif", "image/png", "image/apng", "image/webp", "image/avif",
	"video/mp4", "video/webm", "video/ogg", "video/quicktime",
	"audio/mp4", "audio/webm", "audio/aac", "audio/mpeg", "audio/ogg", "audio/wave",
	"audio/wav", "audio/x-wav", "audio/x-pn-wav", "audio/flac", "audio/x-flac",
}

func (m *Media) Scan(row dbutil.Scannable) (*Media, error) {
	var mimeType, fileName sql.NullString
	var size sql.NullInt64
	var hash []byte
	err := row.Scan(&m.MXC, dbutil.JSON{Data: &m.EncFile}, &fileName, &mimeType, &size, &hash, dbutil.JSON{Data: &m.Error})
	if err != nil {
		return nil, err
	}
	m.MimeType = mimeType.String
	m.FileName = fileName.String
	m.Size = size.Int64
	if len(hash) == 32 {
		m.Hash = (*[32]byte)(hash)
	}
	return m, nil
}

func (m *Media) ContentDisposition() string {
	if slices.Contains(safeMimes, m.MimeType) {
		return "inline"
	}
	return "attachment"
}
