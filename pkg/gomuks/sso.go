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
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"time"

	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
)

const ssoErrorPage = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>gomuks web</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
			margin: 0;
			padding: 0;
			display: flex;
			justify-content: center;
			align-items: center;
		}
	</style>
</head>
<body>
	<h1>Failed to log in</h1>
	<p><code>%s</code></p>
</body>
</html>`

func (gmx *Gomuks) parseSSOServerURL(r *http.Request) error {
	cookie, _ := r.Cookie("gomuks_sso_session")
	if cookie == nil {
		return fmt.Errorf("no SSO session cookie")
	}
	var cookieData SSOCookieData
	if !gmx.validateToken(cookie.Value, &cookieData) {
		return fmt.Errorf("invalid SSO session cookie")
	} else if cookieData.SessionID != r.URL.Query().Get("gomuksSession") {
		return fmt.Errorf("session ID mismatch in query param and cookie")
	} else if time.Until(cookieData.Expiry) < 0 {
		return fmt.Errorf("SSO session cookie expired")
	}
	var err error
	gmx.Client.Client.HomeserverURL, err = url.Parse(cookieData.HomeserverURL)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %w", err)
	}
	return nil
}

func (gmx *Gomuks) HandleSSOComplete(w http.ResponseWriter, r *http.Request) {
	err := gmx.parseSSOServerURL(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, ssoErrorPage, html.EscapeString(err.Error()))
		return
	}
	err = gmx.Client.Login(r.Context(), &mautrix.ReqLogin{
		Type:  mautrix.AuthTypeToken,
		Token: r.URL.Query().Get("loginToken"),
	})
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, ssoErrorPage, html.EscapeString(err.Error()))
	} else {
		w.Header().Set("Location", "..")
		w.WriteHeader(http.StatusFound)
	}
}

type SSOCookieData struct {
	SessionID     string    `json:"session_id"`
	HomeserverURL string    `json:"homeserver_url"`
	Expiry        time.Time `json:"expiry"`
}

func (gmx *Gomuks) PrepareSSO(w http.ResponseWriter, r *http.Request) {
	var data SSOCookieData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		mautrix.MBadJSON.WithMessage("Failed to decode request JSON").Write(w)
		return
	}
	data.SessionID = random.String(16)
	data.Expiry = time.Now().Add(30 * time.Minute)
	cookieData, err := json.Marshal(&data)
	if err != nil {
		mautrix.MUnknown.WithMessage("Failed to encode cookie data").Write(w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "gomuks_sso_session",
		Value:    gmx.signToken(json.RawMessage(cookieData)),
		Expires:  data.Expiry,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(cookieData)
}
