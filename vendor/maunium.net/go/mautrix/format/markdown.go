// Copyright 2018 Tulir Asokan
package format

import (
	"gopkg.in/russross/blackfriday.v2"
	"maunium.net/go/mautrix"
	"strings"
)

func RenderMarkdown(text string) mautrix.Content {
	parser := blackfriday.New(
		blackfriday.WithExtensions(blackfriday.NoIntraEmphasis |
			blackfriday.Tables |
			blackfriday.FencedCode |
			blackfriday.Strikethrough |
			blackfriday.SpaceHeadings |
			blackfriday.DefinitionLists))
	ast := parser.Parse([]byte(text))

	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.UseXHTML,
	})

	var buf strings.Builder
	renderer.RenderHeader(&buf, ast)
	ast.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		return renderer.RenderNode(&buf, node, entering)
	})
	renderer.RenderFooter(&buf, ast)
	htmlBody := buf.String()

	return mautrix.Content{
		FormattedBody: htmlBody,
		Format:        mautrix.FormatHTML,
		MsgType:       mautrix.MsgText,
		Body:          HTMLToText(htmlBody),
	}
}
