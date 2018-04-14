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

package htmlparser

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

type HTMLProcessor interface {
	Preprocess()
	HandleStartTag(tagName string, attrs map[string]string)
	HandleSelfClosingTag(tagName string, attrs map[string]string)
	HandleText(text string)
	HandleEndTag(tagName string)
	ReceiveError(err error)
	Postprocess()
}

type HTMLParser struct {
	*html.Tokenizer
	processor HTMLProcessor
}

func NewHTMLParserFromTokenizer(z *html.Tokenizer, processor HTMLProcessor) HTMLParser {
	return HTMLParser{
		z,
		processor,
	}
}

func NewHTMLParserFromReader(reader io.Reader, processor HTMLProcessor) HTMLParser {
	return NewHTMLParserFromTokenizer(html.NewTokenizer(reader), processor)
}

func NewHTMLParserFromString(html string, processor HTMLProcessor) HTMLParser {
	return NewHTMLParserFromReader(strings.NewReader(html), processor)
}

var SelfClosingTags = []string{"img", "br", "hr", "area", "base", "basefont", "input", "link", "meta"}

func (parser HTMLParser) mapAttrs() map[string]string {
	attrs := make(map[string]string)
	hasMore := true
	for hasMore {
		var key, val []byte
		key, val, hasMore = parser.TagAttr()
		attrs[string(key)] = string(val)
	}
	return attrs
}

func (parser HTMLParser) isSelfClosing(tag string) bool {
	for _, selfClosingTag := range SelfClosingTags {
		if tag == selfClosingTag {
			return true
		}
	}
	return false
}

func (parser HTMLParser) Process() {
	parser.processor.Preprocess()
Loop:
	for {
		tt := parser.Next()
		switch tt {
		case html.ErrorToken:
			parser.processor.ReceiveError(parser.Err())
			break Loop
		case html.TextToken:
			parser.processor.HandleText(string(parser.Text()))
		case html.StartTagToken, html.SelfClosingTagToken:
			tagb, _ := parser.TagName()
			attrs := parser.mapAttrs()
			tag := string(tagb)

			selfClosing := tt == html.SelfClosingTagToken || parser.isSelfClosing(tag)

			if selfClosing {
				parser.processor.HandleSelfClosingTag(tag, attrs)
			} else {
				parser.processor.HandleStartTag(tag, attrs)
			}
		case html.EndTagToken:
			tagb, _ := parser.TagName()
			parser.processor.HandleEndTag(string(tagb))
		}
	}

	parser.processor.Postprocess()
}
