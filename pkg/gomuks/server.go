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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
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
	"go.mau.fi/gomuks/web"
)

func (gmx *Gomuks) StartServer() {
	api := http.NewServeMux()
	api.HandleFunc("GET /websocket", gmx.HandleWebsocket)
	api.HandleFunc("POST /auth", gmx.Authenticate)
	api.HandleFunc("POST /upload", gmx.UploadMedia)
	api.HandleFunc("GET /media/{server}/{media_id}", gmx.DownloadMedia)
	api.HandleFunc("GET /codeblock/{style}", gmx.GetCodeblockCSS)
	apiHandler := exhttp.ApplyMiddleware(
		api,
		hlog.NewHandler(*gmx.Log),
		hlog.RequestIDHandler("request_id", "Request-ID"),
		requestlog.AccessLogger(false),
		exhttp.StripPrefix("/_gomuks"),
		gmx.AuthMiddleware,
	)
	router := http.NewServeMux()
	if gmx.Config.Web.DebugEndpoints {
		router.Handle("/debug/", http.DefaultServeMux)
	}
	router.Handle("/_gomuks/", apiHandler)
	if frontend, err := fs.Sub(web.Frontend, "dist"); err != nil {
		gmx.Log.Warn().Msg("Frontend not found")
	} else {
		router.Handle("/", gmx.FrontendCacheMiddleware(http.FileServerFS(frontend)))
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
	var frontendCacheETag string
	if gmx.Commit != "unknown" && !gmx.BuildTime.IsZero() {
		frontendCacheETag = fmt.Sprintf(`"%s-%s"`, gmx.Commit, gmx.BuildTime.Format(time.RFC3339))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == frontendCacheETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "max-age=604800, immutable")
		}
		if frontendCacheETag != "" {
			w.Header().Set("ETag", frontendCacheETag)
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

func (gmx *Gomuks) validateAuth(token string, imageOnly bool) bool {
	if len(token) > 500 {
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

	var td tokenData
	err = json.Unmarshal(rawJSON, &td)
	return err == nil && td.Username == gmx.Config.Web.Username && td.Expiry.After(time.Now()) && td.ImageOnly == imageOnly
}

func (gmx *Gomuks) generateToken() (string, time.Time) {
	expiry := time.Now().Add(7 * 24 * time.Hour)
	return gmx.signToken(tokenData{
		Username: gmx.Config.Web.Username,
		Expiry:   jsontime.U(expiry),
	}), expiry
}

func (gmx *Gomuks) generateImageToken() string {
	return gmx.signToken(tokenData{
		Username:  gmx.Config.Web.Username,
		Expiry:    jsontime.U(time.Now().Add(1 * time.Hour)),
		ImageOnly: true,
	})
}

func (gmx *Gomuks) signToken(td tokenData) string {
	data := exerrors.Must(json.Marshal(td))
	hasher := hmac.New(sha256.New, []byte(gmx.Config.Web.TokenKey))
	hasher.Write(data)
	checksum := hasher.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(checksum)
}

func (gmx *Gomuks) writeTokenCookie(w http.ResponseWriter) {
	token, expiry := gmx.generateToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "gomuks_auth",
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (gmx *Gomuks) Authenticate(w http.ResponseWriter, r *http.Request) {
	authCookie, err := r.Cookie("gomuks_auth")
	if err == nil && gmx.validateAuth(authCookie.Value, false) {
		hlog.FromRequest(r).Debug().Msg("Authentication successful with existing cookie")
		gmx.writeTokenCookie(w)
		w.WriteHeader(http.StatusOK)
	} else if username, password, ok := r.BasicAuth(); !ok {
		hlog.FromRequest(r).Debug().Msg("Requesting credentials for auth request")
		w.Header().Set("WWW-Authenticate", `Basic realm="gomuks web" charset="UTF-8"`)
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		usernameHash := sha256.Sum256([]byte(username))
		expectedUsernameHash := sha256.Sum256([]byte(gmx.Config.Web.Username))
		usernameCorrect := hmac.Equal(usernameHash[:], expectedUsernameHash[:])
		passwordCorrect := bcrypt.CompareHashAndPassword([]byte(gmx.Config.Web.PasswordHash), []byte(password)) == nil
		if usernameCorrect && passwordCorrect {
			hlog.FromRequest(r).Debug().Msg("Authentication successful with username and password")
			gmx.writeTokenCookie(w)
			w.WriteHeader(http.StatusCreated)
		} else {
			hlog.FromRequest(r).Debug().Msg("Authentication failed with username and password, re-requesting credentials")
			w.Header().Set("WWW-Authenticate", `Basic realm="gomuks web" charset="UTF-8"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func isUserFetch(header http.Header) bool {
	return (header.Get("Sec-Fetch-Site") == "none" ||
		header.Get("Sec-Fetch-Site") == "same-site" ||
		header.Get("Sec-Fetch-Site") == "same-origin") &&
		header.Get("Sec-Fetch-Mode") == "navigate" &&
		header.Get("Sec-Fetch-Dest") == "document" &&
		header.Get("Sec-Fetch-User") == "?1"
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
		} else if r.Header.Get("Sec-Fetch-Site") != "" &&
			r.Header.Get("Sec-Fetch-Site") != "same-origin" &&
			!isUserFetch(r.Header) {
			hlog.FromRequest(r).Debug().
				Str("site", r.Header.Get("Sec-Fetch-Site")).
				Str("dest", r.Header.Get("Sec-Fetch-Dest")).
				Str("mode", r.Header.Get("Sec-Fetch-Mode")).
				Str("user", r.Header.Get("Sec-Fetch-User")).
				Msg("Invalid Sec-Fetch-Site header")
			ErrInvalidHeader.WithMessage("Invalid Sec-Fetch-Site header").Write(w)
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
