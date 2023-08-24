package beeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"maunium.net/go/mautrix"
)

var cli = &http.Client{Timeout: 30 * time.Second}

func newRequest(token, method, path string) *http.Request {
	req := &http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "api.beeper.com",
			Path:   path,
		},
		Method: method,
		Header: http.Header{
			"Authorization": {fmt.Sprintf("Bearer %s", token)},
			"User-Agent":    {mautrix.DefaultUserAgent},
		},
	}
	if method == http.MethodPut || method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func encodeContent(into *http.Request, body any) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(body)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	into.Body = io.NopCloser(&buf)
	return nil
}

func doRequest(req *http.Request, reqData, resp any) (err error) {
	if reqData != nil {
		err = encodeContent(req, reqData)
		if err != nil {
			return
		}
	}
	r, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body != nil {
			retryCount, ok := body["retries"].(float64)
			if ok && retryCount > 0 && r.StatusCode == 403 && req.URL.Path == "/user/login/response" {
				return fmt.Errorf("%w (%d retries left)", ErrInvalidLoginCode, int(retryCount))
			}
			errorMsg, ok := body["error"].(string)
			if ok {
				return fmt.Errorf("server returned error (HTTP %d): %s", r.StatusCode, errorMsg)
			}
		}
		return fmt.Errorf("unexpected status code %d", r.StatusCode)
	}
	if resp != nil {
		err = json.NewDecoder(r.Body).Decode(resp)
		if err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}
	return nil
}
