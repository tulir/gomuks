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

package messages

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/lib/htmlparser"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
)

var matrixToURL = regexp.MustCompile("^(?:https?://)?(?:www\\.)?matrix\\.to/#/([#@!].*)")

type MatrixHTMLProcessor struct {
	text tstring.TString

	indent    string
	listType  string
	lineIsNew bool
	openTags  *TagArray

	room *rooms.Room
}

func (parser *MatrixHTMLProcessor) newline() {
	if !parser.lineIsNew {
		parser.text = parser.text.Append("\n" + parser.indent)
		parser.lineIsNew = true
	}
}

func (parser *MatrixHTMLProcessor) Preprocess() {}

func (parser *MatrixHTMLProcessor) HandleText(text string) {
	style := tcell.StyleDefault
	for _, tag := range *parser.openTags {
		switch tag.Tag {
		case "b", "strong":
			style = style.Bold(true)
		case "i", "em":
			style = style.Italic(true)
		case "s", "del":
			style = style.Strikethrough(true)
		case "u", "ins":
			style = style.Underline(true)
		case "a":
			tag.Text += text
			return
		}
	}

	if parser.openTags.Has("pre", "code") {
		text = strings.Replace(text, "\n", "", -1)
	}
	parser.text = parser.text.AppendStyle(text, style)
	parser.lineIsNew = false
}

func (parser *MatrixHTMLProcessor) HandleStartTag(tagName string, attrs map[string]string) {
	tag := &TagWithMeta{Tag: tagName}
	switch tag.Tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		length := int(tag.Tag[1] - '0')
		parser.text = parser.text.Append(strings.Repeat("#", length) + " ")
		parser.lineIsNew = false
	case "a":
		tag.Meta, _ = attrs["href"]
	case "ol", "ul":
		parser.listType = tag.Tag
	case "li":
		indentSize := 2
		if parser.listType == "ol" {
			list := parser.openTags.Get(parser.listType)
			list.Counter++
			parser.text = parser.text.Append(fmt.Sprintf("%d. ", list.Counter))
			indentSize = int(math.Log10(float64(list.Counter))+1) + len(". ")
		} else {
			parser.text = parser.text.Append("* ")
		}
		parser.indent += strings.Repeat(" ", indentSize)
		parser.lineIsNew = false
	case "blockquote":
		parser.indent += "> "
		parser.text = parser.text.Append("> ")
		parser.lineIsNew = false
	}
	parser.openTags.PushMeta(tag)
}

func (parser *MatrixHTMLProcessor) HandleSelfClosingTag(tagName string, attrs map[string]string) {
	if tagName == "br" {
		parser.newline()
	}
}

func (parser *MatrixHTMLProcessor) HandleEndTag(tagName string) {
	tag := parser.openTags.Pop(tagName)

	switch tag.Tag {
	case "li", "blockquote":
		indentSize := 2
		if tag.Tag == "li" && parser.listType == "ol" {
			list := parser.openTags.Get(parser.listType)
			indentSize = int(math.Log10(float64(list.Counter))+1) + len(". ")
		}
		if len(parser.indent) >= indentSize {
			parser.indent = parser.indent[0 : len(parser.indent)-indentSize]
		}
		// TODO this newline is sometimes not good
		parser.newline()
	case "a":
		match := matrixToURL.FindStringSubmatch(tag.Meta)
		if len(match) == 2 {
			pillTarget := match[1]
			if pillTarget[0] == '@' {
				if member := parser.room.GetMember(pillTarget); member != nil {
					parser.text = parser.text.AppendColor(member.DisplayName, widget.GetHashColor(member.DisplayName))
				} else {
					parser.text = parser.text.Append(pillTarget)
				}
			} else {
				parser.text = parser.text.Append(pillTarget)
			}
		} else {
			// TODO make text clickable rather than printing URL
			parser.text = parser.text.Append(fmt.Sprintf("%s (%s)", tag.Text, tag.Meta))
		}
		parser.lineIsNew = false
	case "p", "pre", "ol", "ul", "h1", "h2", "h3", "h4", "h5", "h6", "div":
		// parser.newline()
	}
}

func (parser *MatrixHTMLProcessor) ReceiveError(err error) {
	if err != io.EOF {
		debug.Print("Unexpected error parsing HTML:", err)
	}
}

func (parser *MatrixHTMLProcessor) Postprocess() {
	if len(parser.text) > 0 && parser.text[len(parser.text)-1].Char == '\n' {
		parser.text = parser.text[:len(parser.text)-1]
	}
}

// ParseHTMLMessage parses a HTML-formatted Matrix event into a UIMessage.
func ParseHTMLMessage(room *rooms.Room, evt *gomatrix.Event) tstring.TString {
	htmlData, _ := evt.Content["formatted_body"].(string)

	processor := &MatrixHTMLProcessor{
		room:      room,
		text:      tstring.NewBlankTString(),
		indent:    "",
		listType:  "",
		lineIsNew: true,
		openTags:  &TagArray{},
	}

	parser := htmlparser.NewHTMLParserFromString(htmlData, processor)
	parser.Process()

	return processor.text
}
