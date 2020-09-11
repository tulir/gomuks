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

package matrix

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/lib/open"
)

const uiaFallbackPage = `<!DOCTYPE html>
<html lang="en">
<head>
	<title>gomuks user-interactive auth</title>
	<meta charset="utf-8"/>
	<style>
		body {
			text-align: center;
		}
	</style>
</head>
<body>
	<h2>Please complete the login in the popup window</h2>
	<p>Keep this page open while logging in, it will close automatically after the login finishes.</p>
	<button onclick="openPopup()">Open popup</button>
	<button onclick="finish(false)">Cancel</button>
	<script>
		const url = location.hash.substr(1)
		let popupWindow

		function finish(success) {
			if (popupWindow) {
				popupWindow.close()
			}
			fetch("", {method: success ? "POST" : "DELETE"}).then(() => window.close())
		}

		function openPopup() {
			popupWindow = window.open(url)
		}

		window.addEventListener("message", evt => evt.data === "authDone" && finish(true))
	</script>
</body>
</html>
`

func (c *Container) UIAFallback(loginType mautrix.AuthType, sessionID string) error {
	errChan := make(chan error, 1)
	server := &http.Server{Addr: ":29325"}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(uiaFallbackPage))
		} else if r.Method == "POST" || r.Method == "DELETE" {
			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					debug.Printf("Failed to shut down SSO server: %v\n", err)
				}
				if r.Method == "DELETE" {
					errChan <- errors.New("login cancelled")
				} else {
					errChan <- nil
				}
			}()
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	go server.ListenAndServe()
	defer server.Close()
	authURL := c.client.BuildURLWithQuery(mautrix.URLPath{"auth", loginType, "fallback", "web"}, map[string]string{
		"session": sessionID,
	})
	link := url.URL{
		Scheme:   "http",
		Host:     "localhost:29325",
		Path:     "/",
		Fragment: authURL,
	}
	err := open.Open(link.String())
	if err != nil {
		return err
	}
	err = <-errChan
	return err
}
