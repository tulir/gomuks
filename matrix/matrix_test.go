// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package matrix

import (
	"testing"
	"maunium.net/go/gomuks/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"maunium.net/go/gomatrix"
	"strings"
	"fmt"
	"io/ioutil"
	"encoding/json"
)

func TestContainer_InitClient_Empty(t *testing.T) {
	cfg := config.NewConfig("/tmp/gomuks-mxtest-0", "/tmp/gomuks-mxtest-0")
	cfg.HS = "https://matrix.org"
	c := Container{config: cfg}
	assert.Nil(t, c.InitClient())
}

func TestContainer_GetCachePath(t *testing.T) {
	cfg := config.NewConfig("/tmp/gomuks-mxtest-1", "/tmp/gomuks-mxtest-1")
	c := Container{config: cfg}
	assert.Equal(t, "/tmp/gomuks-mxtest-1/media/maunium.net/foobar", c.GetCachePath("maunium.net", "foobar"))
}

func TestContainer_SendMarkdownMessage_NoMarkdown(t *testing.T) {
	c := Container{client: mockClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || !strings.HasPrefix(req.URL.Path, "/_matrix/client/r0/rooms/!foo:example.com/send/m.room.message/") {
			return nil, fmt.Errorf("unexpected query: %s %s", req.Method, req.URL.Path)
		}

		body := parseBody(req)
		assert.Equal(t, "m.text", body["msgtype"])
		assert.Equal(t, "test message", body["body"])
		return mockResponse(http.StatusOK, `{"event_id": "!foobar1:example.com"}`), nil
	})}

	evtID, err := c.SendMarkdownMessage("!foo:example.com", "m.text", "test message")
	assert.Nil(t, err)
	assert.Equal(t, "!foobar1:example.com", evtID)
}

func TestContainer_SendMarkdownMessage_WithMarkdown(t *testing.T) {
	c := Container{client: mockClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || !strings.HasPrefix(req.URL.Path, "/_matrix/client/r0/rooms/!foo:example.com/send/m.room.message/") {
			return nil, fmt.Errorf("unexpected query: %s %s", req.Method, req.URL.Path)
		}

		body := parseBody(req)
		assert.Equal(t, "m.text", body["msgtype"])
		assert.Equal(t, "**formatted** <u>test</u> _message_", body["body"])
		assert.Equal(t, "<strong>formatted</strong> <u>test</u> <em>message</em>", body["formatted_body"])
		return mockResponse(http.StatusOK, `{"event_id": "!foobar2:example.com"}`), nil
	})}

	evtID, err := c.SendMarkdownMessage("!foo:example.com", "m.text", "**formatted** <u>test</u> _message_")
	assert.Nil(t, err)
	assert.Equal(t, "!foobar2:example.com", evtID)
}

func TestContainer_SendTyping(t *testing.T) {
	var calls []gomatrix.ReqTyping
	c := Container{client: mockClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || !strings.HasPrefix(req.URL.Path, "/_matrix/client/r0/rooms/!foo:example.com/typing/@user:example.com") {
			return nil, fmt.Errorf("unexpected query: %s %s", req.Method, req.URL.Path)
		}

		rawBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		call := gomatrix.ReqTyping{}
		err = json.Unmarshal(rawBody, &call)
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)

		return mockResponse(http.StatusOK, `{}`), nil
	})}

	c.SendTyping("!foo:example.com", true)
	c.SendTyping("!foo:example.com", true)
	c.SendTyping("!foo:example.com", true)
	c.SendTyping("!foo:example.com", false)
	c.SendTyping("!foo:example.com", true)
	c.SendTyping("!foo:example.com", false)
	assert.Len(t, calls, 4)
	assert.True(t, calls[0].Typing)
	assert.False(t, calls[1].Typing)
	assert.True(t, calls[2].Typing)
	assert.False(t, calls[3].Typing)
}

func mockClient(fn func(*http.Request) (*http.Response, error)) *gomatrix.Client {
	client, _ := gomatrix.NewClient("https://example.com", "@user:example.com", "foobar")
	client.Client = &http.Client{Transport: MockRoundTripper{RT: fn}}
	return client
}

func parseBody(req *http.Request) map[string]interface{} {
	rawBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	data := make(map[string]interface{})

	err = json.Unmarshal(rawBody, &data)
	if err != nil {
		panic(err)
	}

	return data
}

func mockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
	}
}

type MockRoundTripper struct {
	RT func(*http.Request) (*http.Response, error)
}

func (t MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.RT(req)
}
