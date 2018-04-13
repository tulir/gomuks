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
	"strings"

	"golang.org/x/net/html"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/tcell"
)

// TagArray is a reversed queue for remembering what HTML tags are open.
type TagArray []string

// Pushb converts the given byte array into a string and calls Push().
func (ta *TagArray) Pushb(tag []byte) {
	ta.Push(string(tag))
}

// Popb converts the given byte array into a string and calls Pop().
func (ta *TagArray) Popb(tag []byte) {
	ta.Pop(string(tag))
}

// Hasb converts the given byte array into a string and calls Has().
func (ta *TagArray) Hasb(tag []byte) {
	ta.Has(string(tag))
}

// HasAfterb converts the given byte array into a string and calls HasAfter().
func (ta *TagArray) HasAfterb(tag []byte, after int) {
	ta.HasAfter(string(tag), after)
}

// Push adds the given tag to the array.
func (ta *TagArray) Push(tag string) {
	*ta = append(*ta, "")
	copy((*ta)[1:], *ta)
	(*ta)[0] = tag
}

// Pop removes the given tag from the array.
func (ta *TagArray) Pop(tag string) {
	if (*ta)[0] == tag {
		// This is the default case and is lighter than append(), so we handle it separately.
		*ta = (*ta)[1:]
	} else if index := ta.Has(tag); index != -1 {
		*ta = append((*ta)[:index], (*ta)[index+1:]...)
	}
}

// Has returns the first index where the given tag is, or -1 if it's not in the list.
func (ta *TagArray) Has(tag string) int {
	return ta.HasAfter(tag, -1)
}

// HasAfter returns the first index after the given index where the given tag is,
// or -1 if the given tag is not on the list after the given index.
func (ta *TagArray) HasAfter(tag string, after int) int {
	for i := after + 1; i < len(*ta); i++ {
		if (*ta)[i] == tag {
			return i
		}
	}
	return -1
}

// ParseHTMLMessage parses a HTML-formatted Matrix event into a UIMessage.
func ParseHTMLMessage(evt *gomatrix.Event) tstring.TString {
	//textData, _ := evt.Content["body"].(string)
	htmlData, _ := evt.Content["formatted_body"].(string)

	z := html.NewTokenizer(strings.NewReader(htmlData))
	text := tstring.NewTString("")

	openTags := &TagArray{}

Loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break Loop
		case html.TextToken:
			style := tcell.StyleDefault
			for _, tag := range *openTags {
				switch tag {
				case "b", "strong":
					style = style.Bold(true)
				case "i", "em":
					style = style.Italic(true)
				case "s", "del":
					style = style.Strikethrough(true)
				case "u", "ins":
					style = style.Underline(true)
				}
			}
			text = text.AppendStyle(string(z.Text()), style)
		case html.SelfClosingTagToken, html.StartTagToken:
			tagb, _ := z.TagName()
			tag := string(tagb)
			switch tag {
			case "br":
				debug.Print("BR found")
				debug.Print(text.String())
				text = text.Append("\n")
			default:
				if tt == html.StartTagToken {
					openTags.Push(tag)
				}
			}
		case html.EndTagToken:
			tagb, _ := z.TagName()
			openTags.Popb(tagb)
		}
	}

	return text
}
