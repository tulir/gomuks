package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/hicli/database"
	"maunium.net/go/mautrix/id"
)

var ErrBadGateway = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.BAD_GATEWAY",
	StatusCode: http.StatusBadGateway,
}

func (gmx *Gomuks) downloadMediaFromCache(ctx context.Context, w http.ResponseWriter, entry *database.CachedMedia, force bool) bool {
	if !entry.UseCache() {
		if force {
			mautrix.MNotFound.WithMessage("Media not found in cache").Write(w)
			return true
		}
		return false
	}
	if entry.Error != nil {
		w.Header().Set("Mau-Cached-Error", "true")
		entry.Error.Write(w)
		return true
	}
	log := zerolog.Ctx(ctx)
	cacheFile, err := os.Open(gmx.cacheEntryToPath(entry))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !force {
			return false
		}
		log.Err(err).Msg("Failed to open cache file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to open cache file: %v", err)).Write(w)
		return true
	}
	defer func() {
		_ = cacheFile.Close()
	}()
	cacheEntryToHeaders(w, entry)
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, cacheFile)
	if err != nil {
		log.Err(err).Msg("Failed to copy cache file to response")
	}
	return true
}

func (gmx *Gomuks) cacheEntryToPath(entry *database.CachedMedia) string {
	hashPath := hex.EncodeToString(entry.Hash[:])
	return filepath.Join(gmx.CacheDir, "media", hashPath[0:2], hashPath[2:4], hashPath[4:])
}

func cacheEntryToHeaders(w http.ResponseWriter, entry *database.CachedMedia) {
	w.Header().Set("Content-Type", entry.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(entry.Size, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType(entry.ContentDisposition(), map[string]string{"filename": entry.FileName}))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none';")
}

func (gmx *Gomuks) DownloadMedia(w http.ResponseWriter, r *http.Request) {
	mxc := id.ContentURI{
		Homeserver: r.PathValue("server"),
		FileID:     r.PathValue("media_id"),
	}
	if !mxc.IsValid() {
		mautrix.MInvalidParam.WithMessage("Invalid mxc URI").Write(w)
		return
	}
	query := r.URL.Query()
	encrypted, _ := strconv.ParseBool(query.Get("encrypted"))

	logVal := zerolog.Ctx(r.Context()).With().
		Stringer("mxc_uri", mxc).
		Bool("encrypted", encrypted).
		Logger()
	log := &logVal
	ctx := log.WithContext(r.Context())
	cacheEntry, err := gmx.Client.DB.CachedMedia.Get(ctx, mxc)
	if err != nil {
		log.Err(err).Msg("Failed to get cached media entry")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to get cached media entry: %v", err)).Write(w)
		return
	} else if (cacheEntry == nil || cacheEntry.EncFile == nil) && encrypted {
		mautrix.MNotFound.WithMessage("Media encryption keys not found in cache").Write(w)
		return
	}

	if gmx.downloadMediaFromCache(ctx, w, cacheEntry, false) {
		return
	}

	tempFile, err := os.CreateTemp(gmx.TempDir, "download-*")
	if err != nil {
		log.Err(err).Msg("Failed to create temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create temp file: %v", err)).Write(w)
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	resp, err := gmx.Client.Client.Download(ctx, mxc)
	if err != nil {
		log.Err(err).Msg("Failed to download media")
		var httpErr mautrix.HTTPError
		if cacheEntry == nil {
			cacheEntry = &database.CachedMedia{
				MXC: mxc,
			}
		}
		if cacheEntry.Error == nil {
			cacheEntry.Error = &database.MediaError{
				ReceivedAt: jsontime.UnixMilliNow(),
			}
		} else {
			cacheEntry.Error.Attempts++
			cacheEntry.Error.ReceivedAt = jsontime.UnixMilliNow()
		}
		if errors.As(err, &httpErr) {
			if httpErr.WrappedError != nil {
				cacheEntry.Error.Matrix = ptr.Ptr(ErrBadGateway.WithMessage(httpErr.WrappedError.Error()))
				cacheEntry.Error.StatusCode = http.StatusBadGateway
			} else if httpErr.RespError != nil {
				cacheEntry.Error.Matrix = httpErr.RespError
				cacheEntry.Error.StatusCode = httpErr.Response.StatusCode
			} else {
				cacheEntry.Error.Matrix = ptr.Ptr(mautrix.MUnknown.WithMessage("Server returned non-JSON error with status %d", httpErr.Response.StatusCode))
				cacheEntry.Error.StatusCode = httpErr.Response.StatusCode
			}
		} else {
			cacheEntry.Error.Matrix = ptr.Ptr(ErrBadGateway.WithMessage(err.Error()))
			cacheEntry.Error.StatusCode = http.StatusBadGateway
		}
		err = gmx.Client.DB.CachedMedia.Put(ctx, cacheEntry)
		if err != nil {
			log.Err(err).Msg("Failed to save errored cache entry")
		}
		cacheEntry.Error.Write(w)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if cacheEntry == nil {
		cacheEntry = &database.CachedMedia{
			MXC:      mxc,
			MimeType: resp.Header.Get("Content-Type"),
			Size:     resp.ContentLength,
		}
	}

	reader := resp.Body
	if cacheEntry.EncFile != nil {
		err = cacheEntry.EncFile.PrepareForDecryption()
		if err != nil {
			log.Err(err).Msg("Failed to prepare media for decryption")
			mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to prepare media for decryption: %v", err)).Write(w)
			return
		}
		reader = cacheEntry.EncFile.DecryptStream(reader)
	}
	fileHasher := sha256.New()
	hashReader := io.TeeReader(reader, fileHasher)
	cacheEntry.Size, err = io.Copy(tempFile, hashReader)
	if err != nil {
		log.Err(err).Msg("Failed to copy media to temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to copy media to temp file: %v", err)).Write(w)
		return
	}
	err = reader.Close()
	if err != nil {
		log.Err(err).Msg("Failed to close media reader")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to finish reading media: %v", err)).Write(w)
		return
	}
	_ = tempFile.Close()
	if cacheEntry.FileName == "" {
		_, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		cacheEntry.FileName = params["filename"]
	}
	if cacheEntry.MimeType == "" {
		cacheEntry.MimeType = resp.Header.Get("Content-Type")
	}
	cacheEntry.Hash = (*[32]byte)(fileHasher.Sum(nil))
	cacheEntry.Error = nil
	err = gmx.Client.DB.CachedMedia.Put(ctx, cacheEntry)
	if err != nil {
		log.Err(err).Msg("Failed to save cache entry")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to save cache entry: %v", err)).Write(w)
		return
	}
	cachePath := gmx.cacheEntryToPath(cacheEntry)
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		log.Err(err).Msg("Failed to create cache directory")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create cache directory: %v", err)).Write(w)
		return
	}
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		log.Err(err).Msg("Failed to rename temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to rename temp file: %v", err)).Write(w)
		return
	}
	gmx.downloadMediaFromCache(ctx, w, cacheEntry, true)
}
