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

package html

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/net/html"
	"mvdan.cc/xurls/v2"

	"go.mau.fi/tcell"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
)

type htmlParser struct {
	prefs *config.UserPreferences
	room  *rooms.Room
	evt   *muksevt.Event

	preserveWhitespace bool
	linkIDCounter      int
}

func AdjustStyleBold(style tcell.Style) tcell.Style {
	return style.Bold(true)
}

func AdjustStyleItalic(style tcell.Style) tcell.Style {
	return style.Italic(true)
}

func AdjustStyleUnderline(style tcell.Style) tcell.Style {
	return style.Underline(true)
}

func AdjustStyleStrikethrough(style tcell.Style) tcell.Style {
	return style.StrikeThrough(true)
}

func AdjustStyleTextColor(color tcell.Color) AdjustStyleFunc {
	return func(style tcell.Style) tcell.Style {
		return style.Foreground(color)
	}
}

func AdjustStyleBackgroundColor(color tcell.Color) AdjustStyleFunc {
	return func(style tcell.Style) tcell.Style {
		return style.Background(color)
	}
}

func AdjustStyleLink(url, id string) AdjustStyleFunc {
	return func(style tcell.Style) tcell.Style {
		return style.Url(url).UrlId(id)
	}
}

func (parser *htmlParser) maybeGetAttribute(node *html.Node, attribute string) (string, bool) {
	for _, attr := range node.Attr {
		if attr.Key == attribute {
			return attr.Val, true
		}
	}
	return "", false
}

func (parser *htmlParser) getAttribute(node *html.Node, attribute string) string {
	val, _ := parser.maybeGetAttribute(node, attribute)
	return val
}

func (parser *htmlParser) hasAttribute(node *html.Node, attribute string) bool {
	_, ok := parser.maybeGetAttribute(node, attribute)
	return ok
}

func (parser *htmlParser) listToEntity(node *html.Node) Entity {
	children := parser.nodeToEntities(node.FirstChild)
	ordered := node.Data == "ol"
	start := 1
	if ordered {
		if startRaw := parser.getAttribute(node, "start"); len(startRaw) > 0 {
			var err error
			start, err = strconv.Atoi(startRaw)
			if err != nil {
				start = 1
			}
		}
	}
	listItems := children[:0]
	for _, child := range children {
		if child.GetTag() == "li" {
			listItems = append(listItems, child)
		}
	}
	return NewListEntity(ordered, start, listItems)
}

func (parser *htmlParser) basicFormatToEntity(node *html.Node) Entity {
	entity := &ContainerEntity{
		BaseEntity: &BaseEntity{
			Tag: node.Data,
		},
		Children: parser.nodeToEntities(node.FirstChild),
	}
	switch node.Data {
	case "b", "strong":
		entity.AdjustStyle(AdjustStyleBold, AdjustStyleReasonNormal)
	case "i", "em":
		entity.AdjustStyle(AdjustStyleItalic, AdjustStyleReasonNormal)
	case "s", "del", "strike":
		entity.AdjustStyle(AdjustStyleStrikethrough, AdjustStyleReasonNormal)
	case "u", "ins":
		entity.AdjustStyle(AdjustStyleUnderline, AdjustStyleReasonNormal)
	case "code":
		bgColor := tcell.ColorDarkSlateGray
		fgColor := tcell.ColorWhite
		entity.AdjustStyle(AdjustStyleBackgroundColor(bgColor), AdjustStyleReasonNormal)
		entity.AdjustStyle(AdjustStyleTextColor(fgColor), AdjustStyleReasonNormal)
	case "font", "span":
		fgColor, ok := parser.parseColor(node, "data-mx-color", "color")
		if ok {
			entity.AdjustStyle(AdjustStyleTextColor(fgColor), AdjustStyleReasonNormal)
		}
		bgColor, ok := parser.parseColor(node, "data-mx-bg-color", "background-color")
		if ok {
			entity.AdjustStyle(AdjustStyleBackgroundColor(bgColor), AdjustStyleReasonNormal)
		}
		spoilerReason, isSpoiler := parser.maybeGetAttribute(node, "data-mx-spoiler")
		if isSpoiler {
			return NewSpoilerEntity(entity, spoilerReason)
		}
	}
	return entity
}

func (parser *htmlParser) parseColor(node *html.Node, mainName, altName string) (color tcell.Color, ok bool) {
	hex := parser.getAttribute(node, mainName)
	if len(hex) == 0 {
		hex = parser.getAttribute(node, altName)
		if len(hex) == 0 {
			return
		}
	}

	cful, err := colorful.Hex(hex)
	if err != nil {
		color2, found := colorMap[strings.ToLower(hex)]
		if !found {
			return
		}
		cful, _ = colorful.MakeColor(color2)
	}

	r, g, b := cful.RGB255()
	return tcell.NewRGBColor(int32(r), int32(g), int32(b)), true
}

func (parser *htmlParser) headerToEntity(node *html.Node) Entity {
	return (&ContainerEntity{
		BaseEntity: &BaseEntity{
			Tag: node.Data,
		},
		Children: append(
			[]Entity{NewTextEntity(strings.Repeat("#", int(node.Data[1]-'0')) + " ")},
			parser.nodeToEntities(node.FirstChild)...,
		),
	}).AdjustStyle(AdjustStyleBold, AdjustStyleReasonNormal)
}

func (parser *htmlParser) blockquoteToEntity(node *html.Node) Entity {
	return NewBlockquoteEntity(parser.nodeToEntities(node.FirstChild))
}

func (parser *htmlParser) linkToEntity(node *html.Node) Entity {
	sameURL := false
	href := parser.getAttribute(node, "href")

	entity := &ContainerEntity{
		BaseEntity: &BaseEntity{
			Tag: "a",
		},
		Children: parser.nodeToEntities(node.FirstChild),
	}

	if len(href) == 0 {
		return entity
	}

	if len(entity.Children) == 1 {
		entity, ok := entity.Children[0].(*TextEntity)
		if ok && entity.Text == href {
			sameURL = true
		}
	}

	matrixURI, _ := id.ParseMatrixURIOrMatrixToURL(href)
	if matrixURI != nil && (matrixURI.Sigil1 == '@' || matrixURI.Sigil1 == '#') && matrixURI.Sigil2 == 0 {
		text := NewTextEntity(matrixURI.PrimaryIdentifier())
		if matrixURI.Sigil1 == '@' {
			if member := parser.room.GetMember(matrixURI.UserID()); member != nil {
				text.Text = member.Displayname
				text.Style = text.Style.Foreground(widget.GetHashColor(matrixURI.UserID()))
			}
			entity.Children = []Entity{text}
		} else if matrixURI.Sigil1 == '#' {
			entity.Children = []Entity{text}
		}
	} else if parser.prefs.EnableInlineURLs() {
		linkID := fmt.Sprintf("%s-%d", parser.evt.ID, parser.linkIDCounter)
		parser.linkIDCounter++
		entity.AdjustStyle(AdjustStyleLink(href, linkID), AdjustStyleReasonNormal)
	} else if !sameURL && !parser.prefs.DisableShowURLs && !parser.hasAttribute(node, "data-mautrix-exclude-plaintext") {
		entity.Children = append(entity.Children, NewTextEntity(fmt.Sprintf(" (%s)", href)))
	}
	return entity
}

func (parser *htmlParser) imageToEntity(node *html.Node) Entity {
	alt := parser.getAttribute(node, "alt")
	if len(alt) == 0 {
		alt = parser.getAttribute(node, "title")
		if len(alt) == 0 {
			alt = "[inline image]"
		}
	}
	entity := &TextEntity{
		BaseEntity: &BaseEntity{
			Tag: "img",
		},
		Text: alt,
	}
	// TODO add click action and underline on hover for inline images
	return entity
}

func colourToColor(colour chroma.Colour) tcell.Color {
	if !colour.IsSet() {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(int32(colour.Red()), int32(colour.Green()), int32(colour.Blue()))
}

func styleEntryToStyle(se chroma.StyleEntry) tcell.Style {
	return tcell.StyleDefault.
		Bold(se.Bold == chroma.Yes).
		Italic(se.Italic == chroma.Yes).
		Underline(se.Underline == chroma.Yes).
		Foreground(colourToColor(se.Colour)).
		Background(colourToColor(se.Background))
}

func tokenToTextEntity(style *chroma.Style, token *chroma.Token) *TextEntity {
	return &TextEntity{
		BaseEntity: &BaseEntity{
			Tag:           token.Type.String(),
			Style:         styleEntryToStyle(style.Get(token.Type)),
			DefaultHeight: 1,
		},
		Text: token.Value,
	}
}

func (parser *htmlParser) syntaxHighlight(text, language string) Entity {
	lexer := lexers.Get(strings.ToLower(language))
	if lexer == nil {
		lexer = lexers.Get("plaintext")
	}
	iter, err := lexer.Tokenise(nil, text)
	if err != nil {
		return nil
	}
	// TODO allow changing theme
	style := styles.SolarizedDark

	tokens := iter.Tokens()

	var children []Entity
	for _, token := range tokens {
		lines := strings.SplitAfter(token.Value, "\n")
		for _, line := range lines {
			line_len := len(line)
			if line_len == 0 {
				continue
			}
			t := token.Clone()

			if line[line_len-1:] == "\n" {
				t.Value = line[:line_len-1]
				children = append(children, tokenToTextEntity(style, &t), NewBreakEntity())
			} else {
				t.Value = line
				children = append(children, tokenToTextEntity(style, &t))
			}
		}
	}

	return NewCodeBlockEntity(children, styleEntryToStyle(style.Get(chroma.Background)))
}

func (parser *htmlParser) codeblockToEntity(node *html.Node) Entity {
	lang := "plaintext"
	// TODO allow disabling syntax highlighting
	if node.FirstChild != nil && node.FirstChild.Type == html.ElementNode && node.FirstChild.Data == "code" {
		node = node.FirstChild
		attr := parser.getAttribute(node, "class")
		for _, class := range strings.Split(attr, " ") {
			if strings.HasPrefix(class, "language-") {
				lang = class[len("language-"):]
				break
			}
		}
	}
	parser.preserveWhitespace = true
	text := (&ContainerEntity{
		Children: parser.nodeToEntities(node.FirstChild),
	}).PlainText()
	parser.preserveWhitespace = false
	return parser.syntaxHighlight(text, lang)
}

func (parser *htmlParser) tagNodeToEntity(node *html.Node) Entity {
	switch node.Data {
	case "blockquote":
		return parser.blockquoteToEntity(node)
	case "ol", "ul":
		return parser.listToEntity(node)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return parser.headerToEntity(node)
	case "br":
		return NewBreakEntity()
	case "b", "strong", "i", "em", "s", "strike", "del", "u", "ins", "font", "span", "code":
		return parser.basicFormatToEntity(node)
	case "a":
		return parser.linkToEntity(node)
	case "img":
		return parser.imageToEntity(node)
	case "pre":
		return parser.codeblockToEntity(node)
	case "hr":
		return NewHorizontalLineEntity()
	case "mx-reply":
		return nil
	default:
		return &ContainerEntity{
			BaseEntity: &BaseEntity{
				Tag:   node.Data,
				Block: parser.isBlockTag(node.Data),
			},
			Children: parser.nodeToEntities(node.FirstChild),
		}
	}
}

var spaces = regexp.MustCompile("\\s+")

// textToHTMLEntity converts a plain text string into an HTML Entity while preserving newlines.
func textToHTMLEntity(text string) Entity {
	if strings.Index(text, "\n") == -1 {
		return NewTextEntity(text)
	}
	return &ContainerEntity{
		BaseEntity: &BaseEntity{Tag: "span"},
		Children:   textToHTMLEntities(text),
	}
}

func textToHTMLEntities(text string) []Entity {
	lines := strings.SplitAfter(text, "\n")
	entities := make([]Entity, 0, len(lines))
	for _, line := range lines {
		line_len := len(line)
		if line_len == 0 {
			continue
		}
		if line == "\n" {
			entities = append(entities, NewBreakEntity())
		} else if line[line_len-1:] == "\n" {
			entities = append(entities, NewTextEntity(line[:line_len-1]), NewBreakEntity())
		} else {
			entities = append(entities, NewTextEntity(line))
		}
	}
	return entities
}

func TextToEntity(text string, eventID id.EventID, linkify bool) Entity {
	if len(text) == 0 {
		return nil
	}
	if !linkify {
		return textToHTMLEntity(text)
	}
	indices := xurls.Strict().FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return textToHTMLEntity(text)
	}
	ent := &ContainerEntity{
		BaseEntity: &BaseEntity{Tag: "span"},
	}
	var lastEnd int
	for i, item := range indices {
		start, end := item[0], item[1]
		if start > lastEnd {
			ent.Children = append(ent.Children, textToHTMLEntities(text[lastEnd:start])...)
		}
		link := text[start:end]
		linkID := fmt.Sprintf("%s-%d", eventID, i)
		ent.Children = append(ent.Children, NewTextEntity(link).AdjustStyle(AdjustStyleLink(link, linkID), AdjustStyleReasonNormal))
		lastEnd = end
	}
	if lastEnd < len(text) {
		ent.Children = append(ent.Children, textToHTMLEntities(text[lastEnd:])...)
	}
	return ent
}

func (parser *htmlParser) singleNodeToEntity(node *html.Node) Entity {
	switch node.Type {
	case html.TextNode:
		if !parser.preserveWhitespace {
			node.Data = strings.ReplaceAll(node.Data, "\n", "")
			node.Data = spaces.ReplaceAllLiteralString(node.Data, " ")
		}
		return TextToEntity(node.Data, parser.evt.ID, parser.prefs.EnableInlineURLs())
	case html.ElementNode:
		parsed := parser.tagNodeToEntity(node)
		if parsed != nil && !parsed.IsBlock() && parsed.IsEmpty() {
			return nil
		}
		return parsed
	case html.DocumentNode:
		if node.FirstChild.Data == "html" && node.FirstChild.NextSibling == nil {
			return parser.singleNodeToEntity(node.FirstChild)
		}
		return &ContainerEntity{
			BaseEntity: &BaseEntity{
				Tag:   "html",
				Block: true,
			},
			Children: parser.nodeToEntities(node.FirstChild),
		}
	default:
		return nil
	}
}

func (parser *htmlParser) nodeToEntities(node *html.Node) (entities []Entity) {
	for ; node != nil; node = node.NextSibling {
		if entity := parser.singleNodeToEntity(node); entity != nil {
			entities = append(entities, entity)
		}
	}
	return
}

var BlockTags = []string{"p", "h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "li", "pre", "blockquote", "div", "hr", "table"}

func (parser *htmlParser) isBlockTag(tag string) bool {
	for _, blockTag := range BlockTags {
		if tag == blockTag {
			return true
		}
	}
	return false
}

func (parser *htmlParser) Parse(htmlData string) Entity {
	node, _ := html.Parse(strings.NewReader(htmlData))
	bodyNode := node.FirstChild.FirstChild
	for bodyNode != nil && (bodyNode.Type != html.ElementNode || bodyNode.Data != "body") {
		bodyNode = bodyNode.NextSibling
	}
	if bodyNode != nil {
		return parser.singleNodeToEntity(bodyNode)
	}

	return parser.singleNodeToEntity(node)
}

const TabLength = 4

// Parse parses a HTML-formatted Matrix event into a UIMessage.
func Parse(prefs *config.UserPreferences, room *rooms.Room, content *event.MessageEventContent, evt *muksevt.Event, senderDisplayname string) Entity {
	htmlData := content.FormattedBody

	if content.Format != event.FormatHTML {
		htmlData = strings.Replace(html.EscapeString(content.Body), "\n", "<br/>", -1)
	}
	htmlData = strings.Replace(htmlData, "\t", strings.Repeat(" ", TabLength), -1)

	parser := htmlParser{room: room, prefs: prefs, evt: evt}
	root := parser.Parse(htmlData)
	if root == nil {
		return nil
	}
	beRoot, ok := root.(*ContainerEntity)
	if ok {
		beRoot.Block = false
		if len(beRoot.Children) > 0 {
			beChild, ok := beRoot.Children[0].(*ContainerEntity)
			if ok && beChild.Tag == "p" {
				// Hacky fix for m.emote
				beChild.Block = false
			}
		}
	}

	if content.MsgType == event.MsgEmote {
		root = &ContainerEntity{
			BaseEntity: &BaseEntity{
				Tag: "emote",
			},
			Children: []Entity{
				NewTextEntity("* "),
				NewTextEntity(senderDisplayname).AdjustStyle(AdjustStyleTextColor(widget.GetHashColor(evt.Sender)), AdjustStyleReasonNormal),
				NewTextEntity(" "),
				root,
			},
		}
	}

	return root
}
