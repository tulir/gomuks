// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rainbow

import (
	"fmt"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.mau.fi/util/random"
)

// Extension is a goldmark extension that adds rainbow text coloring to the HTML renderer.
var Extension = &extRainbow{}

type extRainbow struct{}
type rainbowRenderer struct {
	HardWraps bool
	ColorID   string
}

var defaultRB = &rainbowRenderer{HardWraps: true, ColorID: random.String(16)}

func (er *extRainbow) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(defaultRB, 0)))
}

func (rb *rainbowRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindText, rb.renderText)
	reg.Register(ast.KindString, rb.renderString)
}

type rainbowBufWriter struct {
	util.BufWriter
	ColorID string
}

func (rbw rainbowBufWriter) WriteString(s string) (int, error) {
	i := 0
	graphemes := uniseg.NewGraphemes(s)
	for graphemes.Next() {
		runes := graphemes.Runes()
		if len(runes) == 1 && unicode.IsSpace(runes[0]) {
			i2, err := rbw.BufWriter.WriteRune(runes[0])
			i += i2
			if err != nil {
				return i, err
			}
			continue
		}
		i2, err := fmt.Fprintf(rbw.BufWriter, "<font color=\"%s\">%s</font>", rbw.ColorID, graphemes.Str())
		i += i2
		if err != nil {
			return i, err
		}
	}
	return i, nil
}

func (rbw rainbowBufWriter) Write(data []byte) (int, error) {
	return rbw.WriteString(string(data))
}

func (rbw rainbowBufWriter) WriteByte(c byte) error {
	_, err := rbw.WriteRune(rune(c))
	return err
}

func (rbw rainbowBufWriter) WriteRune(r rune) (int, error) {
	if unicode.IsSpace(r) {
		return rbw.BufWriter.WriteRune(r)
	} else {
		return fmt.Fprintf(rbw.BufWriter, "<font color=\"%s\">%c</font>", rbw.ColorID, r)
	}
}

func (rb *rainbowRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	segment := n.Segment
	if n.IsRaw() {
		html.DefaultWriter.RawWrite(rainbowBufWriter{w, rb.ColorID}, segment.Value(source))
	} else {
		html.DefaultWriter.Write(rainbowBufWriter{w, rb.ColorID}, segment.Value(source))
		if n.HardLineBreak() || (n.SoftLineBreak() && rb.HardWraps) {
			_, _ = w.WriteString("<br>\n")
		} else if n.SoftLineBreak() {
			_ = w.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

func (rb *rainbowRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.String)
	if n.IsCode() {
		_, _ = w.Write(n.Value)
	} else {
		if n.IsRaw() {
			html.DefaultWriter.RawWrite(rainbowBufWriter{w, rb.ColorID}, n.Value)
		} else {
			html.DefaultWriter.Write(rainbowBufWriter{w, rb.ColorID}, n.Value)
		}
	}
	return ast.WalkContinue, nil
}
