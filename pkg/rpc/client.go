// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/net/publicsuffix"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type GomuksRPC struct {
	EventHandler func(ctx context.Context, evt any)
	UserAgent    string

	BaseURL *url.URL
	http    *http.Client
	conn    atomic.Pointer[websocket.Conn]
	connCtx atomic.Pointer[context.Context]
	stop    atomic.Pointer[context.CancelFunc]

	pendingRequestsLock sync.RWMutex
	reqIDCounter        int64
	pendingRequests     map[int64]chan<- *jsoncmd.Container[json.RawMessage]
}

func NewGomuksRPC(rawBaseURL string) (*GomuksRPC, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	cli := &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   20 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
		},
		Jar:     jar,
		Timeout: 180 * time.Second,
	}
	return &GomuksRPC{
		EventHandler:    func(_ context.Context, _ any) {},
		BaseURL:         baseURL,
		UserAgent:       "gomuks-rpc " + mautrix.DefaultUserAgent,
		http:            cli,
		pendingRequests: make(map[int64]chan<- *jsoncmd.Container[json.RawMessage]),
	}, nil
}

type GomuksURLPath []any

func (gup GomuksURLPath) FullPath() []any {
	return append([]any{"_gomuks"}, gup...)
}

func (gr *GomuksRPC) BuildURL(path ...any) string {
	return gr.BuildURLWithQuery(path, nil)
}

func (gr *GomuksRPC) BuildRawURL(path GomuksURLPath) *url.URL {
	return mautrix.BuildURL(gr.BaseURL, path.FullPath()...)
}

func (gr *GomuksRPC) BuildURLWithQuery(path GomuksURLPath, query url.Values) string {
	built := mautrix.BuildURL(gr.BaseURL, path.FullPath()...)
	built.RawQuery = query.Encode()
	return built.String()
}

func (gr *GomuksRPC) Authenticate(ctx context.Context, username, password string) error {
	addr := gr.BuildURLWithQuery(GomuksURLPath{"auth"}, url.Values{"insecure_cookie": {"true"}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %w", err)
	}
	req.Header.Set("User-Agent", gr.UserAgent)
	req.SetBasicAuth(username, password)
	resp, err := gr.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to authenticate: HTTP %d", resp.StatusCode)
	}
	return nil
}

type DownloadMediaParams struct {
	MXC               id.ContentURI
	FallbackColor     string
	FallbackCharacter string
	AvatarThumbnail   bool
	Encrypted         bool
}

func (gr *GomuksRPC) DownloadMedia(ctx context.Context, params DownloadMediaParams) (*http.Response, error) {
	query := url.Values{}
	if params.FallbackColor != "" && params.FallbackCharacter != "" {
		query.Set("fallback", fmt.Sprintf("%s:%s", params.FallbackColor, params.FallbackCharacter))
	}
	if params.AvatarThumbnail {
		query.Set("thumbnail", "avatar")
	}
	if params.Encrypted {
		query.Set("encrypted", "true")
	}
	url := gr.BuildURLWithQuery(GomuksURLPath{"media", params.MXC.Homeserver, params.MXC.FileID}, query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %w", err)
	}
	resp, err := gr.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	} else if resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to download media: HTTP %d", resp.StatusCode)
	}
	return resp, err
}
