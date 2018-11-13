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

package parser

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/net/html"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix"
	"maunium.net/go/tcell"
	"strconv"
)

var matrixToURL = regexp.MustCompile("^(?:https?://)?(?:www\\.)?matrix\\.to/#/([#@!].*)")

type htmlParser struct {
	room *rooms.Room
}

type taggedTString struct {
	tstring.TString
	tag string
}

var AdjustStyleBold = func(style tcell.Style) tcell.Style {
	return style.Bold(true)
}

var AdjustStyleItalic = func(style tcell.Style) tcell.Style {
	return style.Italic(true)
}

var AdjustStyleUnderline = func(style tcell.Style) tcell.Style {
	return style.Underline(true)
}

var AdjustStyleStrikethrough = func(style tcell.Style) tcell.Style {
	return style.Strikethrough(true)
}

func (parser *htmlParser) getAttribute(node *html.Node, attribute string) string {
	for _, attr := range node.Attr {
		if attr.Key == attribute {
			return attr.Val
		}
	}
	return ""
}

func digits(num int) int {
	if num <= 0 {
		return 0
	}
	return int(math.Floor(math.Log10(float64(num))) + 1)
}

func (parser *htmlParser) listToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	ordered := node.Data == "ol"
	taggedChildren := parser.nodeToTaggedTStrings(node.FirstChild, stripLinebreak)
	counter := 1
	indentLength := 0
	if ordered {
		start := parser.getAttribute(node, "start")
		if len(start) > 0 {
			counter, _ = strconv.Atoi(start)
		}

		longestIndex := (counter - 1) + len(taggedChildren)
		indentLength = digits(longestIndex)
	}
	indent := strings.Repeat(" ", indentLength+2)
	var children []tstring.TString
	for _, child := range taggedChildren {
		if child.tag != "li" {
			continue
		}
		var prefix string
		if ordered {
			indexPadding := indentLength - digits(counter)
			prefix = fmt.Sprintf("%d. %s", counter, strings.Repeat(" ", indexPadding))
		} else {
			prefix = "● "
		}
		str := child.TString.Prepend(prefix)
		counter++
		parts := str.Split('\n')
		for i, part := range parts[1:] {
			parts[i+1] = part.Prepend(indent)
		}
		str = tstring.Join(parts, "\n")
		children = append(children, str)
	}
	return tstring.Join(children, "\n")
}

func (parser *htmlParser) basicFormatToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	str := parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak)
	switch node.Data {
	case "b", "strong":
		str.AdjustStyleFull(AdjustStyleBold)
	case "i", "em":
		str.AdjustStyleFull(AdjustStyleItalic)
	case "s", "del":
		str.AdjustStyleFull(AdjustStyleStrikethrough)
	case "u", "ins":
		str.AdjustStyleFull(AdjustStyleUnderline)
	}
	return str
}

func (parser *htmlParser) fontToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	str := parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak)
	hex := parser.getAttribute(node, "color")
	if len(hex) == 0 {
		return str
	}

	color, err := colorful.Hex(hex)
	if err != nil {
		return str
	}

	r, g, b := color.RGB255()
	tcellColor := tcell.NewRGBColor(int32(r), int32(g), int32(b))
	str.AdjustStyleFull(func(style tcell.Style) tcell.Style {
		return style.Foreground(tcellColor)
	})
	return str
}

func (parser *htmlParser) headerToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	children := parser.nodeToTStrings(node.FirstChild, stripLinebreak)
	length := int(node.Data[1] - '0')
	prefix := strings.Repeat("#", length) + " "
	return tstring.Join(children, "").Prepend(prefix)
}

func (parser *htmlParser) blockquoteToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	str := parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak)
	childrenArr := str.TrimSpace().Split('\n')
	for index, child := range childrenArr {
		childrenArr[index] = child.Prepend("> ")
	}
	return tstring.Join(childrenArr, "\n")
}

func (parser *htmlParser) linkToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	str := parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak)
	href := parser.getAttribute(node, "href")
	if len(href) == 0 {
		return str
	}
	match := matrixToURL.FindStringSubmatch(href)
	if len(match) == 2 {
		pillTarget := match[1]
		if pillTarget[0] == '@' {
			if member := parser.room.GetMember(pillTarget); member != nil {
				return tstring.NewColorTString(member.Displayname, widget.GetHashColor(pillTarget))
			}
		}
		return tstring.NewTString(pillTarget)
	}
	return str.Append(fmt.Sprintf(" (%s)", href))
}

func (parser *htmlParser) tagToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	switch node.Data {
	case "blockquote":
		return parser.blockquoteToTString(node, stripLinebreak)
	case "ol", "ul":
		return parser.listToTString(node, stripLinebreak)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return parser.headerToTString(node, stripLinebreak)
	case "br":
		return tstring.NewTString("\n")
	case "b", "strong", "i", "em", "s", "del", "u", "ins":
		return parser.basicFormatToTString(node, stripLinebreak)
	case "font":
		return parser.fontToTString(node, stripLinebreak)
	case "a":
		return parser.linkToTString(node, stripLinebreak)
	case "p":
		return parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak).Append("\n")
	case "pre":
		return parser.nodeToTString(node.FirstChild, false)
	default:
		return parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak)
	}
}

func (parser *htmlParser) singleNodeToTString(node *html.Node, stripLinebreak bool) taggedTString {
	switch node.Type {
	case html.TextNode:
		if stripLinebreak {
			node.Data = strings.Replace(node.Data, "\n", "", -1)
		}
		return taggedTString{tstring.NewTString(node.Data), "text"}
	case html.ElementNode:
		return taggedTString{parser.tagToTString(node, stripLinebreak), node.Data}
	case html.DocumentNode:
		return taggedTString{parser.nodeToTagAwareTString(node.FirstChild, stripLinebreak), "html"}
	default:
		return taggedTString{tstring.NewBlankTString(), "unknown"}
	}
}

func (parser *htmlParser) nodeToTaggedTStrings(node *html.Node, stripLinebreak bool) (strs []taggedTString) {
	for ; node != nil; node = node.NextSibling {
		strs = append(strs, parser.singleNodeToTString(node, stripLinebreak))
	}
	return
}

var BlockTags = []string{"p", "h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "pre", "blockquote", "div", "hr", "table"}

func (parser *htmlParser) isBlockTag(tag string) bool {
	for _, blockTag := range BlockTags {
		if tag == blockTag {
			return true
		}
	}
	return false
}

func (parser *htmlParser) nodeToTagAwareTString(node *html.Node, stripLinebreak bool) tstring.TString {
	strs := parser.nodeToTaggedTStrings(node, stripLinebreak)
	output := tstring.NewBlankTString()
	for _, str := range strs {
		tstr := str.TString
		if parser.isBlockTag(str.tag) {
			tstr = tstr.Prepend("\n").Append("\n")
		}
		output = output.AppendTString(tstr)
	}
	return output.TrimSpace()
}

func (parser *htmlParser) nodeToTStrings(node *html.Node, stripLinebreak bool) (strs []tstring.TString) {
	for ; node != nil; node = node.NextSibling {
		strs = append(strs, parser.singleNodeToTString(node, stripLinebreak).TString)
	}
	return
}

func (parser *htmlParser) nodeToTString(node *html.Node, stripLinebreak bool) tstring.TString {
	return tstring.Join(parser.nodeToTStrings(node, stripLinebreak), "")
}

func (parser *htmlParser) Parse(htmlData string) tstring.TString {
	node, _ := html.Parse(strings.NewReader(htmlData))
	return parser.nodeToTagAwareTString(node, true)
}

// ParseHTMLMessage parses a HTML-formatted Matrix event into a UIMessage.
func ParseHTMLMessage(room *rooms.Room, evt *mautrix.Event, senderDisplayname string) tstring.TString {
	htmlData := evt.Content.FormattedBody
	htmlData = strings.Replace(htmlData, "\t", "    ", -1)

	parser := htmlParser{room}
	str := parser.Parse(htmlData)

	if evt.Content.MsgType == mautrix.MsgEmote {
		str = tstring.Join([]tstring.TString{
			tstring.NewTString("* "),
			tstring.NewColorTString(senderDisplayname, widget.GetHashColor(evt.Sender)),
			tstring.NewTString(" "),
			str,
		}, "")
	}

	return str
}
