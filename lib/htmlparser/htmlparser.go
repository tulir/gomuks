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

// HTMLProcessor contains the functions to process parsed HTML data.
type HTMLProcessor interface {
	// Preprocess is called before the parsing is started.
	Preprocess()

	// HandleStartTag is called with the tag name and attributes when
	// the parser encounters a StartTagToken, except if the tag is
	// always self-closing.
	HandleStartTag(tagName string, attrs map[string]string)
	// HandleSelfClosingTag is called with the tag name and attributes
	// when the parser encounters a SelfClosingTagToken OR a StartTagToken
	// with a tag that's always self-closing.
	HandleSelfClosingTag(tagName string, attrs map[string]string)
	// HandleText is called with the text when the parser encounters
	// a TextToken.
	HandleText(text string)
	// HandleEndTag is called with the tag name when the parser encounters
	// an EndTagToken.
	HandleEndTag(tagName string)

	// ReceiveError is called with the error when the parser encounters
	// an ErrorToken that IS NOT io.EOF.
	ReceiveError(err error)

	// Postprocess is called after parsing is completed successfully.
	// An unsuccessful parsing will trigger a ReceiveError() call.
	Postprocess()
}

// HTMLParser wraps a net/html.Tokenizer and a HTMLProcessor to call
// the HTMLProcessor with data from the Tokenizer.
type HTMLParser struct {
	*html.Tokenizer
	processor HTMLProcessor
}

// NewHTMLParserFromTokenizer creates a new HTMLParser from an existing html Tokenizer.
func NewHTMLParserFromTokenizer(z *html.Tokenizer, processor HTMLProcessor) HTMLParser {
	return HTMLParser{
		z,
		processor,
	}
}

// NewHTMLParserFromReader creates a Tokenizer with the given io.Reader and
// then uses that to create a new HTMLParser.
func NewHTMLParserFromReader(reader io.Reader, processor HTMLProcessor) HTMLParser {
	return NewHTMLParserFromTokenizer(html.NewTokenizer(reader), processor)
}

// NewHTMLParserFromString creates a Tokenizer with a reader of the given
// string and then uses that to create a new HTMLParser.
func NewHTMLParserFromString(html string, processor HTMLProcessor) HTMLParser {
	return NewHTMLParserFromReader(strings.NewReader(html), processor)
}

// SelfClosingTags is the list of tags that always call
// HTMLProcessor.HandleSelfClosingTag() even if it is encountered
// as a html.StartTagToken rather than html.SelfClosingTagToken.
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

// Process parses the HTML using the tokenizer in this parser and
// calls the appropriate functions of the HTML processor.
func (parser HTMLParser) Process() {
	parser.processor.Preprocess()
Loop:
	for {
		tt := parser.Next()
		switch tt {
		case html.ErrorToken:
			if parser.Err() != io.EOF {
				parser.processor.ReceiveError(parser.Err())
				return
			}
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
