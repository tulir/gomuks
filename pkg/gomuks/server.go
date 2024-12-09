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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rs/zerolog/hlog"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/requestlog"
	"golang.org/x/crypto/bcrypt"
	"maunium.net/go/mautrix"

	"go.mau.fi/gomuks/pkg/hicli"
)

func (gmx *Gomuks) CreateAPIRouter() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("GET /websocket", gmx.HandleWebsocket)
	api.HandleFunc("POST /auth", gmx.Authenticate)
	api.HandleFunc("POST /upload", gmx.UploadMedia)
	api.HandleFunc("GET /sso", gmx.HandleSSOComplete)
	api.HandleFunc("POST /sso", gmx.PrepareSSO)
	api.HandleFunc("GET /media/{server}/{media_id}", gmx.DownloadMedia)
	api.HandleFunc("POST /keys/export", gmx.ExportKeys)
	api.HandleFunc("POST /keys/export/{room_id}", gmx.ExportKeys)
	api.HandleFunc("POST /keys/import", gmx.ImportKeys)
	api.HandleFunc("GET /keys/restorebackup", gmx.RestoreKeyBackup)
	api.HandleFunc("GET /keys/restorebackup/{room_id}", gmx.RestoreKeyBackup)
	api.HandleFunc("GET /codeblock/{style}", gmx.GetCodeblockCSS)
	api.HandleFunc("GET /url_preview", gmx.GetURLPreview)
	return exhttp.ApplyMiddleware(
		api,
		hlog.NewHandler(*gmx.Log),
		hlog.RequestIDHandler("request_id", "Request-ID"),
		requestlog.AccessLogger(false),
	)
}

func (gmx *Gomuks) StartServer() {
	api := gmx.CreateAPIRouter()
	router := http.NewServeMux()
	if gmx.Config.Web.DebugEndpoints {
		router.Handle("/debug/", http.DefaultServeMux)
	}
	router.Handle("/_gomuks/", exhttp.ApplyMiddleware(
		api,
		exhttp.StripPrefix("/_gomuks"),
		gmx.AuthMiddleware,
	))
	if frontend, err := fs.Sub(gmx.FrontendFS, "dist"); err != nil {
		gmx.Log.Warn().Msg("Frontend not found")
	} else {
		router.Handle("/", gmx.FrontendCacheMiddleware(http.FileServerFS(frontend)))
		if gmx.Commit != "unknown" && !gmx.BuildTime.IsZero() {
			gmx.frontendETag = fmt.Sprintf(`"%s-%s"`, gmx.Commit, gmx.BuildTime.Format(time.RFC3339))

			indexFile, err := frontend.Open("index.html")
			if err != nil {
				gmx.Log.Err(err).Msg("Failed to open index.html")
			} else {
				data, err := io.ReadAll(indexFile)
				_ = indexFile.Close()
				if err == nil {
					gmx.indexWithETag = bytes.Replace(
						data,
						[]byte("<!-- etag placeholder -->"),
						[]byte(fmt.Sprintf(`<meta name="gomuks-frontend-etag" content="%s">`, html.EscapeString(gmx.frontendETag))),
						1,
					)
				}
			}
		}
	}
	gmx.Server = &http.Server{
		Addr:    gmx.Config.Web.ListenAddress,
		Handler: router,
	}
	go func() {
		err := gmx.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	gmx.Log.Info().Str("address", gmx.Config.Web.ListenAddress).Msg("Server started")
}

func (gmx *Gomuks) FrontendCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gmx.frontendETag != "" && r.Header.Get("If-None-Match") == gmx.frontendETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "max-age=604800, immutable")
		}
		if gmx.frontendETag != "" {
			w.Header().Set("ETag", gmx.frontendETag)
			if r.URL.Path == "/" && gmx.indexWithETag != nil {
				w.Header().Set("Content-Type", "text/html")
				w.Header().Set("Content-Length", strconv.Itoa(len(gmx.indexWithETag)))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(gmx.indexWithETag)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

var (
	ErrInvalidHeader = mautrix.RespError{ErrCode: "FI.MAU.GOMUKS.INVALID_HEADER", StatusCode: http.StatusForbidden}
	ErrMissingCookie = mautrix.RespError{ErrCode: "FI.MAU.GOMUKS.MISSING_COOKIE", Err: "Missing gomuks_auth cookie", StatusCode: http.StatusUnauthorized}
	ErrInvalidCookie = mautrix.RespError{ErrCode: "FI.MAU.GOMUKS.INVALID_COOKIE", Err: "Invalid gomuks_auth cookie", StatusCode: http.StatusUnauthorized}
)

type tokenData struct {
	Username  string        `json:"username"`
	Expiry    jsontime.Unix `json:"expiry"`
	ImageOnly bool          `json:"image_only,omitempty"`
}

func (gmx *Gomuks) validateToken(token string, output any) bool {
	if len(token) > 4096 {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	rawJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	checksum, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	hasher := hmac.New(sha256.New, []byte(gmx.Config.Web.TokenKey))
	hasher.Write(rawJSON)
	if !hmac.Equal(hasher.Sum(nil), checksum) {
		return false
	}

	err = json.Unmarshal(rawJSON, output)
	return err == nil
}

func (gmx *Gomuks) validateAuth(token string, imageOnly bool) bool {
	if len(token) > 500 {
		return false
	}
	var td tokenData
	return gmx.validateToken(token, &td) &&
		td.Username == gmx.Config.Web.Username &&
		td.Expiry.After(time.Now()) &&
		td.ImageOnly == imageOnly
}

func (gmx *Gomuks) generateToken() (string, time.Time) {
	expiry := time.Now().Add(7 * 24 * time.Hour)
	return gmx.signToken(tokenData{
		Username: gmx.Config.Web.Username,
		Expiry:   jsontime.U(expiry),
	}), expiry
}

func (gmx *Gomuks) generateImageToken(expiry time.Duration) string {
	return gmx.signToken(tokenData{
		Username:  gmx.Config.Web.Username,
		Expiry:    jsontime.U(time.Now().Add(expiry)),
		ImageOnly: true,
	})
}

func (gmx *Gomuks) signToken(td any) string {
	data := exerrors.Must(json.Marshal(td))
	hasher := hmac.New(sha256.New, []byte(gmx.Config.Web.TokenKey))
	hasher.Write(data)
	checksum := hasher.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(checksum)
}

func (gmx *Gomuks) writeTokenCookie(w http.ResponseWriter, created, jsonOutput, insecureCookie bool) {
	token, expiry := gmx.generateToken()
	if !jsonOutput {
		http.SetCookie(w, &http.Cookie{
			Name:     "gomuks_auth",
			Value:    token,
			Expires:  expiry,
			HttpOnly: true,
			Secure:   !insecureCookie,
			SameSite: http.SameSiteLaxMode,
		})
	}
	if created {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	if jsonOutput {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

func (gmx *Gomuks) Authenticate(w http.ResponseWriter, r *http.Request) {
	if gmx.DisableAuth {
		w.WriteHeader(http.StatusOK)
		return
	} else if gmx.Config.Web.Username == "" || gmx.Config.Web.PasswordHash == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	jsonOutput := r.URL.Query().Get("output") == "json"
	allowPrompt := r.URL.Query().Get("no_prompt") != "true"
	insecureCookie := r.URL.Query().Get("insecure_cookie") == "true"
	authCookie, err := r.Cookie("gomuks_auth")
	if err == nil && gmx.validateAuth(authCookie.Value, false) {
		hlog.FromRequest(r).Debug().Msg("Authentication successful with existing cookie")
		gmx.writeTokenCookie(w, false, jsonOutput, insecureCookie)
	} else if found, correct := gmx.doBasicAuth(r); found && correct {
		hlog.FromRequest(r).Debug().Msg("Authentication successful with username and password")
		gmx.writeTokenCookie(w, true, jsonOutput, insecureCookie)
	} else {
		if !found {
			hlog.FromRequest(r).Debug().Msg("Requesting credentials for auth request")
		} else {
			hlog.FromRequest(r).Debug().Msg("Authentication failed with username and password, re-requesting credentials")
		}
		if allowPrompt {
			w.Header().Set("WWW-Authenticate", `Basic realm="gomuks web" charset="UTF-8"`)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func (gmx *Gomuks) doBasicAuth(r *http.Request) (found, correct bool) {
	var username, password string
	username, password, found = r.BasicAuth()
	if !found {
		return
	}
	usernameHash := sha256.Sum256([]byte(username))
	expectedUsernameHash := sha256.Sum256([]byte(gmx.Config.Web.Username))
	usernameCorrect := hmac.Equal(usernameHash[:], expectedUsernameHash[:])
	passwordCorrect := bcrypt.CompareHashAndPassword([]byte(gmx.Config.Web.PasswordHash), []byte(password)) == nil
	correct = passwordCorrect && usernameCorrect
	return
}

func isImageFetch(header http.Header) bool {
	return header.Get("Sec-Fetch-Site") == "cross-site" &&
		header.Get("Sec-Fetch-Mode") == "no-cors" &&
		header.Get("Sec-Fetch-Dest") == "image"
}

func (gmx *Gomuks) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/media") &&
			isImageFetch(r.Header) &&
			gmx.validateAuth(r.URL.Query().Get("image_auth"), true) &&
			r.URL.Query().Get("encrypted") == "false" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path != "/auth" {
			authCookie, err := r.Cookie("gomuks_auth")
			if err != nil {
				ErrMissingCookie.Write(w)
				return
			} else if !gmx.validateAuth(authCookie.Value, false) {
				http.SetCookie(w, &http.Cookie{
					Name:   "gomuks_auth",
					MaxAge: -1,
				})
				ErrInvalidCookie.Write(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (gmx *Gomuks) GetCodeblockCSS(w http.ResponseWriter, r *http.Request) {
	styleName := r.PathValue("style")
	if !strings.HasSuffix(styleName, ".css") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	style := styles.Get(strings.TrimSuffix(styleName, ".css"))
	if style == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/css")
	_ = hicli.CodeBlockFormatter.WriteCSS(w, style)
}
