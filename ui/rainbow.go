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

package ui

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"math/rand"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/russross/blackfriday/v2"
)

type RainbowRenderer struct {
	*blackfriday.HTMLRenderer
	sr *blackfriday.SPRenderer

	ColorID string
}

func Rand(n int) (str string) {
	b := make([]byte, n)
	rand.Read(b)
	str = fmt.Sprintf("%x", b)
	return
}

func NewRainbowRenderer(html *blackfriday.HTMLRenderer) *RainbowRenderer {
	return &RainbowRenderer{
		HTMLRenderer: html,
		sr:           blackfriday.NewSmartypantsRenderer(html.Flags),
		ColorID:      Rand(16),
	}
}

func (r *RainbowRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if node.Type == blackfriday.Text {
		var buf bytes.Buffer
		if r.Flags&blackfriday.Smartypants != 0 {
			var tmp bytes.Buffer
			escapeHTML(&tmp, node.Literal)
			r.sr.Process(&buf, tmp.Bytes())
		} else {
			if node.Parent.Type == blackfriday.Link {
				escLink(&buf, node.Literal)
			} else {
				escapeHTML(&buf, node.Literal)
			}
		}
		graphemes := uniseg.NewGraphemes(buf.String())
		buf.Reset()
		for graphemes.Next() {
			runes := graphemes.Runes()
			if len(runes) == 1 && unicode.IsSpace(runes[0]) {
				buf.WriteRune(runes[0])
				continue
			}
			_, _ = fmt.Fprintf(&buf, "<font color=\"%s\">%s</font>", r.ColorID, graphemes.Str())
		}
		_, _ = w.Write(buf.Bytes())
		return blackfriday.GoToNext
	}
	return r.HTMLRenderer.RenderNode(w, node, entering)
}

// This stuff is copied directly from blackfriday
var htmlEscaper = [256][]byte{
	'&': []byte("&amp;"),
	'<': []byte("&lt;"),
	'>': []byte("&gt;"),
	'"': []byte("&quot;"),
}

func escapeHTML(w io.Writer, s []byte) {
	var start, end int
	for end < len(s) {
		escSeq := htmlEscaper[s[end]]
		if escSeq != nil {
			w.Write(s[start:end])
			w.Write(escSeq)
			start = end + 1
		}
		end++
	}
	if start < len(s) && end <= len(s) {
		w.Write(s[start:end])
	}
}

func escLink(w io.Writer, text []byte) {
	unesc := html.UnescapeString(string(text))
	escapeHTML(w, []byte(unesc))
}
