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
)

func TestContainer_InitClient_Empty(t *testing.T) {
	cfg := config.NewConfig("/tmp/gomuks-mxtest-0", "/tmp/gomuks-mxtest-0")
	cfg.HS = "https://matrix.org"
	c := Container{config: cfg}
	assert.Nil(t, c.InitClient())
}

func TestContainer_renderMarkdown(t *testing.T) {
	text := "**foo** _bar_"
	c := Container{}
	assert.Equal(t, "<strong>foo</strong> <em>bar</em>", c.renderMarkdown(text))
}

func TestContainer_GetCachePath(t *testing.T) {
	cfg := config.NewConfig("/tmp/gomuks-mxtest-1", "/tmp/gomuks-mxtest-1")
	c := Container{config: cfg}
	assert.Equal(t, "/tmp/gomuks-mxtest-1/media/maunium.net/foobar", c.GetCachePath("maunium.net", "foobar"))
}
