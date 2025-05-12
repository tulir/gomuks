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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buckket/go-blurhash"
	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	cwebp "go.mau.fi/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/orientation"
)

var ErrBadGateway = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.BAD_GATEWAY",
	StatusCode: http.StatusBadGateway,
}

func (gmx *Gomuks) downloadMediaFromCache(ctx context.Context, w http.ResponseWriter, r *http.Request, entry *database.Media, force, useThumbnail bool) bool {
	if !entry.UseCache() {
		if force {
			mautrix.MNotFound.WithMessage("Media not found in cache").Write(w)
			return true
		}
		return false
	}
	etag := entry.ETag(useThumbnail)
	if entry.Error != nil {
		w.Header().Set("Mau-Cached-Error", "true")
		entry.Error.Write(w)
		return true
	} else if etag != "" && r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	} else if entry.MimeType != "" && r.URL.Query().Has("fallback") && !isAllowedAvatarMime(entry.MimeType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return true
	}
	log := zerolog.Ctx(ctx)
	hash := entry.Hash
	if useThumbnail {
		if entry.ThumbnailError != "" {
			log.Debug().Str(zerolog.ErrorFieldName, entry.ThumbnailError).Msg("Returning cached thumbnail error")
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		if entry.ThumbnailHash == nil {
			err := gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
			if errors.Is(err, os.ErrNotExist) && !force {
				return false
			} else if err != nil {
				log.Err(err).Msg("Failed to generate avatar thumbnail")
				gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
				w.WriteHeader(http.StatusInternalServerError)
				return true
			} else {
				gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
			}
		}
		hash = entry.ThumbnailHash
	}
	cacheFile, err := os.Open(gmx.cacheEntryToPath(hash[:]))
	if useThumbnail && errors.Is(err, os.ErrNotExist) {
		err = gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
		if errors.Is(err, os.ErrNotExist) && !force {
			return false
		} else if err != nil {
			log.Err(err).Msg("Failed to generate avatar thumbnail")
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
			w.WriteHeader(http.StatusInternalServerError)
			return true
		} else {
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
			cacheFile, err = os.Open(gmx.cacheEntryToPath(hash[:]))
		}
	}
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
	cacheEntryToHeaders(w, entry, useThumbnail)
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

func cacheEntryToHeaders(w http.ResponseWriter, entry *database.Media, thumbnail bool) {
	if thumbnail {
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Content-Length", strconv.FormatInt(entry.ThumbnailSize, 10))
		w.Header().Set("Content-Disposition", "inline; filename=thumbnail.webp")
	} else {
		w.Header().Set("Content-Type", entry.MimeType)
		w.Header().Set("Content-Length", strconv.FormatInt(entry.Size, 10))
		w.Header().Set("Content-Disposition", mime.FormatMediaType(entry.ContentDisposition(), map[string]string{"filename": entry.FileName}))
	}
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none'; media-src 'self';")
	w.Header().Set("Cache-Control", "max-age=2592000, immutable")
	w.Header().Set("ETag", entry.ETag(thumbnail))
}

func (gmx *Gomuks) saveMediaCacheEntryWithThumbnail(ctx context.Context, entry *database.Media, err error) {
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		entry.ThumbnailError = err.Error()
	}
	err = gmx.Client.DB.Media.Put(ctx, entry)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to save cache entry after generating thumbnail")
	}
}

func (gmx *Gomuks) generateAvatarThumbnail(entry *database.Media, size int) error {
	cacheFile, err := os.Open(gmx.cacheEntryToPath(entry.Hash[:]))
	if err != nil {
		return fmt.Errorf("failed to open full file: %w", err)
	}
	img, err := imaging.Decode(cacheFile, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	tempFile, err := os.CreateTemp(gmx.TempDir, "thumbnail-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	thumbnailImage := imaging.Thumbnail(img, size, size, imaging.Lanczos)
	fileHasher := sha256.New()
	wrappedWriter := io.MultiWriter(fileHasher, tempFile)
	err = cwebp.Encode(wrappedWriter, thumbnailImage, &cwebp.Options{Quality: 80})
	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}
	entry.ThumbnailHash = (*[32]byte)(fileHasher.Sum(nil))
	entry.ThumbnailError = ""
	entry.ThumbnailSize = fileInfo.Size()
	cachePath := gmx.cacheEntryToPath(entry.ThumbnailHash[:])
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}

type noErrorWriter struct {
	io.Writer
}

func (new *noErrorWriter) Write(p []byte) (n int, err error) {
	n, _ = new.Writer.Write(p)
	return
}

// note: this should stay in sync with makeAvatarFallback in web/src/api/media.ts
const fallbackAvatarTemplate = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">
  <rect x="0" y="0" width="1000" height="1000" fill="%s"/>
  <text x="500" y="750" text-anchor="middle" fill="#fff" font-weight="bold" font-size="666"
    font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
  >%s</text>
</svg>`

type avatarResponseWriter struct {
	http.ResponseWriter
	bgColor   string
	character string
	errored   bool
}

func isAllowedAvatarMime(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func (w *avatarResponseWriter) WriteHeader(statusCode int) {
	if statusCode != http.StatusOK && statusCode != http.StatusNotModified {
		data := []byte(fmt.Sprintf(fallbackAvatarTemplate, w.bgColor, html.EscapeString(w.character)))
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Del("Content-Disposition")
		w.ResponseWriter.WriteHeader(http.StatusOK)
		_, _ = w.ResponseWriter.Write(data)
		w.errored = true
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *avatarResponseWriter) Write(p []byte) (n int, err error) {
	if w.errored {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
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
	fallback := query.Get("fallback")
	if fallback != "" {
		fallbackParts := strings.Split(fallback, ":")
		if len(fallbackParts) == 2 {
			w = &avatarResponseWriter{
				ResponseWriter: w,
				bgColor:        fallbackParts[0],
				character:      fallbackParts[1],
			}
		}
	}

	encrypted, _ := strconv.ParseBool(query.Get("encrypted"))
	useThumbnail := query.Get("thumbnail") == "avatar"

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
	} else if cacheEntry != nil && cacheEntry.EncFile != nil && !encrypted {
		mautrix.MNotFound.WithMessage("Tried to download encrypted media without encrypted flag").Write(w)
		return
	}

	if gmx.downloadMediaFromCache(ctx, w, r, cacheEntry, false, useThumbnail) {
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

	_, resp, err := gmx.Client.Client.MakeFullRequestWithResp(ctx, mautrix.FullRequest{
		Method:           http.MethodGet,
		URL:              gmx.Client.Client.BuildClientURL("v1", "media", "download", mxc.Homeserver, mxc.FileID),
		DontReadResponse: true,
		MaxAttempts:      1,
	})
	if err != nil {
		if ctx.Err() != nil {
			w.WriteHeader(499)
			return
		}
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
				Attempts:   1,
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
	if cacheEntry.Size > 0 && cacheEntry.EncFile == nil && !useThumbnail {
		cacheEntryToHeaders(w, cacheEntry, useThumbnail)
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
		gmx.downloadMediaFromCache(ctx, w, r, cacheEntry, true, useThumbnail)
	}
}

func (gmx *Gomuks) reencodeMedia(ctx context.Context, query url.Values, tempFile *os.File) ([]byte, error) {
	defer func() {
		_ = tempFile.Close()
	}()
	encTo := query.Get("encode_to")
	if encTo == "" {
		return nil, nil
	}
	resizeWidthVal := query.Get("resize_width")
	resizeHeightVal := query.Get("resize_height")
	resizePercentVal := query.Get("resize_percent")
	var resizeWidth, resizeHeight, resizePercent int
	if resizeWidthVal != "" && resizeHeightVal != "" {
		var err error
		resizeWidth, err = strconv.Atoi(resizeWidthVal)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resize width: %w", err)
		}
		resizeHeight, err = strconv.Atoi(resizeHeightVal)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resize height: %w", err)
		}
	} else if resizePercentVal != "" {
		var err error
		resizePercent, err = strconv.Atoi(resizePercentVal)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resize percent: %w", err)
		} else if resizePercent < 1 || resizePercent > 100 {
			return nil, fmt.Errorf("resize percent must be between 1 and 100")
		}
	}
	switch encTo {
	case "image/webp", "image/jpeg", "image/png", "image/gif":
		_, err := tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		qualityVal := query.Get("quality")
		quality := 80
		if qualityVal != "" {
			quality, err = strconv.Atoi(qualityVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse quality: %w", err)
			}
		}
		decoded, decodedFrom, err := image.Decode(tempFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
		var o orientation.Orientation
		if decodedFrom == "jpeg" {
			_, err = tempFile.Seek(0, io.SeekStart)
			if err != nil {
				return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
			}
			o = orientation.Read(tempFile)
		} // TODO heic orientation?
		if o != orientation.Unspecified {
			decoded = o.Fix(decoded)
		}
		if resizeWidth > 0 && resizeHeight > 0 {
			decoded = imaging.Resize(decoded, resizeWidth, resizeHeight, imaging.Lanczos)
		} else if resizePercent != 0 {
			resizeWidth = int(float64(decoded.Bounds().Dx()) * float64(resizePercent) / 100)
			resizeHeight = int(float64(decoded.Bounds().Dy()) * float64(resizePercent) / 100)
			decoded = imaging.Resize(decoded, resizeWidth, resizeHeight, imaging.Lanczos)
		}
		_, err = tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		err = tempFile.Truncate(0)
		if err != nil {
			return nil, fmt.Errorf("failed to truncate temp file: %w", err)
		}
		switch encTo {
		case "image/webp":
			err = cwebp.Encode(tempFile, decoded, &cwebp.Options{
				Quality:  float32(quality),
				Lossless: quality >= 100,
			})
		case "image/jpeg":
			err = jpeg.Encode(tempFile, decoded, &jpeg.Options{Quality: quality})
		case "image/png":
			err = png.Encode(tempFile, decoded)
		case "image/gif":
			err = gif.Encode(tempFile, decoded, nil)
		default:
			panic("unreachable")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to encode image: %w", err)
		}
		_, err = tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
	case "video/webm", "video/mp4", "image/webp+anim":
		_ = tempFile.Close()
		var encToExtension string
		var inputArgs, outputArgs []string
		switch encTo {
		case "video/webm":
			encToExtension = ".webm"
			outputArgs = []string{"-c:v", "libvpx-vp9", "-c:a", "libopus", "-pix_fmt", "yuva420p"}
		case "video/mp4":
			encToExtension = ".mp4"
			outputArgs = []string{"-c:v", "libx264", "-c:a", "aac", "-pix_fmt", "yuv420p"}
		case "image/webp+anim":
			encToExtension = ".webp"
			outputArgs = []string{"-c:v", "libwebp_anim", "-pix_fmt", "yuva420p", "-loop", "0"}
		default:
			panic("unreachable")
		}
		if resizeWidth > 0 && resizeHeight > 0 {
			outputArgs = append(outputArgs, "-vf", fmt.Sprintf("scale=%d:%d,setsar=1:1", resizeWidth, resizeHeight))
		}
		outputPath, err := ffmpeg.ConvertPath(ctx, tempFile.Name(), encToExtension, inputArgs, outputArgs, true)
		if err != nil {
			return nil, fmt.Errorf("failed to convert video: %w", err)
		}
		err = os.Rename(outputPath, tempFile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to rename converted video: %w", err)
		}
		tempFile, err = os.OpenFile(tempFile.Name(), os.O_RDONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to reopen converted video: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported encoding target %q", encTo)
	}
	hasher := sha256.New()
	_, err := io.Copy(hasher, tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to hash re-encoded image: %w", err)
	}
	checksum := hasher.Sum(nil)
	return checksum, nil
}

func (gmx *Gomuks) UploadMedia(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)
	encrypt, _ := strconv.ParseBool(r.URL.Query().Get("encrypt"))
	content, err := gmx.cacheAndUploadMedia(r.Context(), r.Body, encrypt, r.URL.Query())
	if err != nil {
		log.Err(err).Msg("Failed to upload media")
		writeMaybeRespError(err, w)
		return
	}
	exhttp.WriteJSONResponse(w, http.StatusOK, content)
}

func (gmx *Gomuks) GetURLPreview(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)
	url := r.URL.Query().Get("url")
	if url == "" {
		mautrix.MInvalidParam.WithMessage("URL must be provided to preview").Write(w)
		return
	}
	linkPreview, err := gmx.Client.Client.GetURLPreview(r.Context(), url)
	if err != nil {
		log.Err(err).Msg("Failed to get URL preview")
		writeMaybeRespError(err, w)
		return
	}

	preview := event.BeeperLinkPreview{
		LinkPreview: *linkPreview,
		MatchedURL:  url,
	}

	if preview.ImageURL != "" {
		encrypt, _ := strconv.ParseBool(r.URL.Query().Get("encrypt"))

		var content *event.MessageEventContent

		if encrypt {
			if fileInfo, ok := gmx.temporaryMXCToEncryptedFileInfo[preview.ImageURL]; ok {
				content = &event.MessageEventContent{File: fileInfo}
			}
		} else {
			if mxc, ok := gmx.temporaryMXCToPermanent[preview.ImageURL]; ok {
				content = &event.MessageEventContent{URL: mxc}
			}
		}

		parsedImageURL, err := preview.ImageURL.Parse()
		if content == nil && (err != nil || parsedImageURL.IsEmpty()) {
			log.Warn().Err(err).Str("image_url", string(preview.ImageURL)).Msg("Failed to parse URL preview image mxc")
		} else if content == nil && !parsedImageURL.IsEmpty() {
			resp, err := gmx.Client.Client.Download(r.Context(), parsedImageURL)
			if err != nil {
				log.Err(err).Msg("Failed to download URL preview image")
				writeMaybeRespError(err, w)
				return
			}
			defer resp.Body.Close()

			content, err = gmx.cacheAndUploadMedia(r.Context(), resp.Body, encrypt, nil)
			if err != nil {
				log.Err(err).Msg("Failed to upload URL preview image")
				writeMaybeRespError(err, w)
				return
			}

			if encrypt {
				gmx.temporaryMXCToEncryptedFileInfo[preview.ImageURL] = content.File
			} else {
				gmx.temporaryMXCToPermanent[preview.ImageURL] = content.URL
			}
		}

		if content != nil {
			preview.ImageURL = content.URL
			preview.ImageEncryption = content.File
		}
	}

	exhttp.WriteJSONResponse(w, http.StatusOK, preview)
}

func (gmx *Gomuks) cacheAndUploadMedia(ctx context.Context, reader io.Reader, encrypt bool, query url.Values) (*event.MessageEventContent, error) {
	log := zerolog.Ctx(ctx)
	tempFile, err := os.CreateTemp(gmx.TempDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	hasher := sha256.New()
	_, err = io.Copy(tempFile, io.TeeReader(reader, hasher))
	if err != nil {
		return nil, fmt.Errorf("failed to copy upload media to temp file: %w", err)
	}
	checksum := hasher.Sum(nil)
	if newHash, err := gmx.reencodeMedia(ctx, query, tempFile); err != nil {
		return nil, fmt.Errorf("failed to reencode media: %w", err)
	} else if newHash != nil {
		checksum = newHash
	}

	cachePath := gmx.cacheEntryToPath(checksum)
	if _, err = os.Stat(cachePath); err == nil {
		log.Debug().Str("path", cachePath).Msg("Media already exists in cache, removing temp file")
	} else {
		err = os.MkdirAll(filepath.Dir(cachePath), 0700)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
		err = os.Rename(tempFile.Name(), cachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to rename temp file: %w", err)
		}
	}

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}

	msgType, info, defaultFileName, err := gmx.generateFileInfo(ctx, cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file info: %w", err)
	}
	if msgType == event.MsgVideo {
		err = gmx.generateVideoThumbnail(ctx, cacheFile.Name(), encrypt, info)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate video thumbnail")
		}
	}
	fileName := query.Get("filename")
	if fileName == "" {
		fileName = defaultFileName
	}
	content := &event.MessageEventContent{
		MsgType:  msgType,
		Body:     fileName,
		Info:     info,
		FileName: fileName,
	}
	content.File, content.URL, err = gmx.uploadFile(ctx, checksum, cacheFile, encrypt, int64(info.Size), info.MimeType, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}
	return content, nil
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

var magickPath string

func init() {
	magickPath, _ = exec.LookPath("magick")
}

func getDimensionsWithMagick(ctx context.Context, file *os.File) (w, h int) {
	if stdout, err := exec.CommandContext(ctx, magickPath, "identify", "-format", "%w %h", file.Name()+"[0]").Output(); err != nil {
		var stderr []byte
		var e *exec.ExitError
		if errors.As(err, &e) {
			stderr = e.Stderr
		}
		zerolog.Ctx(ctx).Err(err).Bytes("stderr", stderr).Msg("Failed to get image dimensions with magick")
	} else if spaceIdx := bytes.IndexByte(stdout, ' '); spaceIdx == -1 {
		zerolog.Ctx(ctx).Error().Bytes("stdout", stdout).Msg("Failed to parse magick output")
	} else if width, err := strconv.Atoi(string(stdout[:spaceIdx])); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("stdout", stdout).Msg("Failed to parse width in magick output")
	} else if height, err := strconv.Atoi(string(stdout[spaceIdx+1:])); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("stdout", stdout).Msg("Failed to parse height in magick output")
	} else {
		return width, height
	}
	return 0, 0
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
		img, _, err := image.Decode(file)
		if err != nil {
			if magickPath != "" {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode image config, trying with magick")
				info.Width, info.Height = getDimensionsWithMagick(ctx, file)
			} else {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode image config and magick not installed")
			}
		} else {
			bounds := img.Bounds()
			info.Width = bounds.Dx()
			info.Height = bounds.Dy()
			hash, err := blurhash.Encode(4, 3, img)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to generate image blurhash")
			}
			info.AnoaBlurhash = hash
			if mimeType.String() == "image/jpeg" {
				_, err = file.Seek(0, io.SeekStart)
				if err != nil {
					return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
				}
				info.Width, info.Height = orientation.Read(file).ApplyToDimensions(info.Width, info.Height)
			}
		}
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
	img, _, err := image.Decode(tempFile)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode thumbnail image config")
	} else {
		bounds := img.Bounds()
		thumbnailInfo.Width = bounds.Dx()
		thumbnailInfo.Height = bounds.Dy()
		hash, err := blurhash.Encode(4, 3, img)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to generate image blurhash")
		}
		thumbnailInfo.AnoaBlurhash = hash
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
