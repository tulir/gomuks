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

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	_ "golang.org/x/image/webp"

	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

var ErrBadGateway = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.BAD_GATEWAY",
	StatusCode: http.StatusBadGateway,
}

func (gmx *Gomuks) downloadMediaFromCache(ctx context.Context, w http.ResponseWriter, entry *database.Media, force bool) bool {
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
	cacheFile, err := os.Open(gmx.cacheEntryToPath(entry.Hash[:]))
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

func (gmx *Gomuks) cacheEntryToPath(hash []byte) string {
	hashPath := hex.EncodeToString(hash[:])
	return filepath.Join(gmx.CacheDir, "media", hashPath[0:2], hashPath[2:4], hashPath[4:])
}

func cacheEntryToHeaders(w http.ResponseWriter, entry *database.Media) {
	w.Header().Set("Content-Type", entry.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(entry.Size, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType(entry.ContentDisposition(), map[string]string{"filename": entry.FileName}))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none';")
}

type noErrorWriter struct {
	io.Writer
}

func (new *noErrorWriter) Write(p []byte) (n int, err error) {
	n, _ = new.Writer.Write(p)
	return
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
	cacheEntry, err := gmx.Client.DB.Media.Get(ctx, mxc)
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
			cacheEntry = &database.Media{
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
		err = gmx.Client.DB.Media.Put(ctx, cacheEntry)
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
		cacheEntry = &database.Media{
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
	if cacheEntry.FileName == "" {
		_, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		cacheEntry.FileName = params["filename"]
	}
	if cacheEntry.MimeType == "" {
		cacheEntry.MimeType = resp.Header.Get("Content-Type")
	}
	cacheEntry.Size = resp.ContentLength
	fileHasher := sha256.New()
	wrappedReader := io.TeeReader(reader, fileHasher)
	if cacheEntry.Size > 0 && cacheEntry.EncFile == nil {
		cacheEntryToHeaders(w, cacheEntry)
		w.WriteHeader(http.StatusOK)
		wrappedReader = io.TeeReader(wrappedReader, &noErrorWriter{w})
		w = nil
	}
	cacheEntry.Size, err = io.Copy(tempFile, wrappedReader)
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
	cacheEntry.Hash = (*[32]byte)(fileHasher.Sum(nil))
	cacheEntry.Error = nil
	err = gmx.Client.DB.Media.Put(ctx, cacheEntry)
	if err != nil {
		log.Err(err).Msg("Failed to save cache entry")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to save cache entry: %v", err)).Write(w)
		return
	}
	cachePath := gmx.cacheEntryToPath(cacheEntry.Hash[:])
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
	if w != nil {
		gmx.downloadMediaFromCache(ctx, w, cacheEntry, true)
	}
}

func (gmx *Gomuks) UploadMedia(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)
	tempFile, err := os.CreateTemp(gmx.TempDir, "upload-*")
	if err != nil {
		log.Err(err).Msg("Failed to create temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create temp file: %v", err)).Write(w)
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	hasher := sha256.New()
	_, err = io.Copy(tempFile, io.TeeReader(r.Body, hasher))
	if err != nil {
		log.Err(err).Msg("Failed to copy upload media to temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to copy media to temp file: %v", err)).Write(w)
		return
	}
	_ = tempFile.Close()

	checksum := hasher.Sum(nil)
	cachePath := gmx.cacheEntryToPath(checksum)
	if _, err = os.Stat(cachePath); err == nil {
		log.Debug().Str("path", cachePath).Msg("Media already exists in cache, removing temp file")
	} else {
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
	}

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		log.Err(err).Msg("Failed to open cache file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to open cache file: %v", err)).Write(w)
		return
	}

	msgType, info, defaultFileName, err := gmx.generateFileInfo(r.Context(), cacheFile)
	if err != nil {
		log.Err(err).Msg("Failed to generate file info")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to generate file info: %v", err)).Write(w)
		return
	}
	encrypt, _ := strconv.ParseBool(r.URL.Query().Get("encrypt"))
	if msgType == event.MsgVideo {
		err = gmx.generateVideoThumbnail(r.Context(), cacheFile.Name(), encrypt, info)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate video thumbnail")
		}
	}
	fileName := r.URL.Query().Get("filename")
	if fileName == "" {
		fileName = defaultFileName
	}
	content := &event.MessageEventContent{
		MsgType:  msgType,
		Body:     fileName,
		Info:     info,
		FileName: fileName,
	}
	content.File, content.URL, err = gmx.uploadFile(r.Context(), checksum, cacheFile, encrypt, int64(info.Size), info.MimeType, fileName)
	if err != nil {
		log.Err(err).Msg("Failed to upload media")
		writeMaybeRespError(err, w)
		return
	}
	exhttp.WriteJSONResponse(w, http.StatusOK, content)
}

func (gmx *Gomuks) uploadFile(ctx context.Context, checksum []byte, cacheFile *os.File, encrypt bool, fileSize int64, mimeType, fileName string) (*event.EncryptedFileInfo, id.ContentURIString, error) {
	cm := &database.Media{
		FileName: fileName,
		MimeType: mimeType,
		Size:     fileSize,
		Hash:     (*[32]byte)(checksum),
	}
	var cacheReader io.ReadSeekCloser = cacheFile
	if encrypt {
		cm.EncFile = attachment.NewEncryptedFile()
		cacheReader = cm.EncFile.EncryptStream(cacheReader)
		mimeType = "application/octet-stream"
		fileName = ""
	}
	resp, err := gmx.Client.Client.UploadMedia(ctx, mautrix.ReqUploadMedia{
		Content:       cacheReader,
		ContentLength: fileSize,
		ContentType:   mimeType,
		FileName:      fileName,
	})
	err2 := cacheReader.Close()
	if err != nil {
		return nil, "", err
	} else if err2 != nil {
		return nil, "", fmt.Errorf("failed to close cache reader: %w", err)
	}
	cm.MXC = resp.ContentURI
	err = gmx.Client.DB.Media.Put(ctx, cm)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).
			Stringer("mxc", cm.MXC).
			Hex("checksum", checksum).
			Msg("Failed to save cache entry")
	}
	if cm.EncFile != nil {
		return &event.EncryptedFileInfo{
			EncryptedFile: *cm.EncFile,
			URL:           resp.ContentURI.CUString(),
		}, "", nil
	} else {
		return nil, resp.ContentURI.CUString(), nil
	}
}

func (gmx *Gomuks) generateFileInfo(ctx context.Context, file *os.File) (event.MessageType, *event.FileInfo, string, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to stat cache file: %w", err)
	}
	mimeType, err := mimetype.DetectReader(file)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to detect mime type: %w", err)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
	}
	info := &event.FileInfo{
		MimeType: mimeType.String(),
		Size:     int(fileInfo.Size()),
	}
	var msgType event.MessageType
	var defaultFileName string
	switch strings.Split(mimeType.String(), "/")[0] {
	case "image":
		msgType = event.MsgImage
		defaultFileName = "image" + mimeType.Extension()
		cfg, _, err := image.DecodeConfig(file)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode image config")
		}
		info.Width = cfg.Width
		info.Height = cfg.Height
	case "video":
		msgType = event.MsgVideo
		defaultFileName = "video" + mimeType.Extension()
	case "audio":
		msgType = event.MsgAudio
		defaultFileName = "audio" + mimeType.Extension()
	default:
		msgType = event.MsgFile
		defaultFileName = "file" + mimeType.Extension()
	}
	if msgType == event.MsgVideo || msgType == event.MsgAudio {
		probe, err := ffmpeg.Probe(ctx, file.Name())
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to probe video")
		} else if probe != nil && probe.Format != nil {
			info.Duration = int(probe.Format.Duration * 1000)
			for _, stream := range probe.Streams {
				if stream.Width != 0 {
					info.Width = stream.Width
					info.Height = stream.Height
					break
				}
			}
		}
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
	}
	return msgType, info, defaultFileName, nil
}

func (gmx *Gomuks) generateVideoThumbnail(ctx context.Context, filePath string, encrypt bool, saveInto *event.FileInfo) error {
	tempPath := filepath.Join(gmx.TempDir, "thumbnail-"+random.String(12)+".jpeg")
	defer os.Remove(tempPath)
	err := ffmpeg.ConvertPathWithDestination(
		ctx, filePath, tempPath, nil,
		[]string{"-frames:v", "1", "-update", "1", "-f", "image2"},
		false,
	)
	if err != nil {
		return err
	}
	tempFile, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tempFile.Close()
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, tempFile)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}
	thumbnailInfo := &event.FileInfo{
		MimeType: "image/jpeg",
		Size:     int(fileInfo.Size()),
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}
	cfg, _, err := image.DecodeConfig(tempFile)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode thumbnail image config")
	} else {
		thumbnailInfo.Width = cfg.Width
		thumbnailInfo.Height = cfg.Height
	}
	_ = tempFile.Close()
	checksum := hasher.Sum(nil)
	cachePath := gmx.cacheEntryToPath(checksum)
	if _, err = os.Stat(cachePath); err == nil {
		zerolog.Ctx(ctx).Debug().Str("path", cachePath).Msg("Media already exists in cache, removing temp file")
	} else {
		err = os.MkdirAll(filepath.Dir(cachePath), 0700)
		if err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		err = os.Rename(tempPath, cachePath)
		if err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}
	tempFile, err = os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open renamed file: %w", err)
	}
	saveInto.ThumbnailFile, saveInto.ThumbnailURL, err = gmx.uploadFile(ctx, checksum, tempFile, encrypt, fileInfo.Size(), "image/jpeg", "thumbnail.jpeg")
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	saveInto.ThumbnailInfo = thumbnailInfo
	return nil
}

func writeMaybeRespError(err error, w http.ResponseWriter) {
	var httpErr mautrix.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.WrappedError != nil {
			ErrBadGateway.WithMessage(httpErr.WrappedError.Error()).Write(w)
		} else if httpErr.RespError != nil {
			httpErr.RespError.Write(w)
		} else {
			mautrix.MUnknown.WithMessage("Server returned non-JSON error").Write(w)
		}
	} else {
		mautrix.MUnknown.WithMessage(err.Error()).Write(w)
	}
}
