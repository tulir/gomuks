// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"maunium.net/go/mautrix/id"
	"mvdan.cc/xurls/v2"
)

func tagIsAllowed(tag atom.Atom) bool {
	switch tag {
	case atom.Del, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Blockquote, atom.P,
		atom.A, atom.Ul, atom.Ol, atom.Sup, atom.Sub, atom.Li, atom.B, atom.I, atom.U, atom.Strong,
		atom.Em, atom.S, atom.Code, atom.Hr, atom.Br, atom.Div, atom.Table, atom.Thead, atom.Tbody,
		atom.Tr, atom.Th, atom.Td, atom.Caption, atom.Pre, atom.Span, atom.Font, atom.Img,
		atom.Details, atom.Summary, atom.Input:
		return true
	default:
		return false
	}
}

func isSelfClosing(tag atom.Atom) bool {
	switch tag {
	case atom.Img, atom.Br, atom.Hr, atom.Input:
		return true
	default:
		return false
	}
}

var languageRegex = regexp.MustCompile(`^language-([a-zA-Z0-9-]+)$`)
var allowedColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// This is approximately a mirror of web/src/util/mediasize.ts in gomuks
func calculateMediaSize(widthInt, heightInt int) (width, height float64, ok bool) {
	if widthInt <= 0 || heightInt <= 0 {
		return
	}
	width = float64(widthInt)
	height = float64(heightInt)
	const imageContainerWidth float64 = 320
	const imageContainerHeight float64 = 240
	const imageContainerAspectRatio = imageContainerWidth / imageContainerHeight
	if width > imageContainerWidth || height > imageContainerHeight {
		aspectRatio := width / height
		if aspectRatio > imageContainerAspectRatio {
			width = imageContainerWidth
			height = imageContainerWidth / aspectRatio
		} else if aspectRatio < imageContainerAspectRatio {
			width = imageContainerHeight * aspectRatio
			height = imageContainerHeight
		} else {
			width = imageContainerWidth
			height = imageContainerHeight
		}
	}
	ok = true
	return
}

func getAttribute(attrs []html.Attribute, key string) (string, bool) {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func parseImgAttributes(attrs []html.Attribute) (src, alt, title string, isCustomEmoji bool, width, height int) {
	for _, attr := range attrs {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "alt":
			alt = attr.Val
		case "title":
			title = attr.Val
		case "data-mx-emoticon":
			isCustomEmoji = true
		case "width":
			width, _ = strconv.Atoi(attr.Val)
		case "height":
			height, _ = strconv.Atoi(attr.Val)
		}
	}
	return
}

func parseSpanAttributes(attrs []html.Attribute) (bgColor, textColor, spoiler string, isSpoiler bool) {
	for _, attr := range attrs {
		switch attr.Key {
		case "data-mx-bg-color":
			if allowedColorRegex.MatchString(attr.Val) {
				bgColor = attr.Val
			}
		case "data-mx-color", "color":
			if allowedColorRegex.MatchString(attr.Val) {
				textColor = attr.Val
			}
		case "data-mx-spoiler":
			spoiler = attr.Val
			isSpoiler = true
		}
	}
	return
}

func parseAAttributes(attrs []html.Attribute) (href string) {
	for _, attr := range attrs {
		switch attr.Key {
		case "href":
			href = strings.TrimSpace(attr.Val)
		}
	}
	return
}

func attributeIsAllowed(tag atom.Atom, attr html.Attribute) bool {
	switch tag {
	case atom.Ol:
		switch attr.Key {
		case "start":
			_, err := strconv.Atoi(attr.Val)
			return err == nil
		}
	case atom.Code:
		switch attr.Key {
		case "class":
			return languageRegex.MatchString(attr.Val)
		}
	case atom.Div:
		switch attr.Key {
		case "data-mx-maths":
			return true
		}
	}
	return false
}

// Funny user IDs will just need to be linkified by the sender, no auto-linkification for them.
var plainUserOrAliasMentionRegex = regexp.MustCompile(`[@#][a-zA-Z0-9._=/+-]{0,254}:[a-zA-Z0-9.-]+(?:\d{1,5})?`)

func getNextItem(items [][]int, minIndex int) (index, start, end int, ok bool) {
	for i, item := range items {
		if item[0] >= minIndex {
			return i, item[0], item[1], true
		}
	}
	return -1, -1, -1, false
}

func writeMention(w *strings.Builder, mention []byte) {
	uri := &id.MatrixURI{
		Sigil1: rune(mention[0]),
		MXID1:  string(mention[1:]),
	}
	w.WriteString(`<a`)
	writeAttribute(w, "href", uri.String())
	writeAttribute(w, "class", matrixURIClassName(uri)+" hicli-matrix-uri-plaintext")
	w.WriteByte('>')
	writeEscapedBytes(w, mention)
	w.WriteString("</a>")
}

func writeURL(w *strings.Builder, addr []byte) {
	addrString := string(addr)
	parsedURL, err := url.Parse(addrString)
	if err != nil {
		writeEscapedBytes(w, addr)
		return
	}
	if parsedURL.Scheme == "" && parsedURL.Host == "" {
		if parsedURL.RawQuery == "" && parsedURL.Fragment == "" && strings.LastIndexByte(parsedURL.Path, '/') == -1 && strings.IndexByte(parsedURL.Path, '@') > 0 {
			parsedURL.Scheme = "mailto"
			parsedURL.Opaque = parsedURL.Path
			parsedURL.Path = ""
		} else {
			parsedURL, err = url.Parse("https://" + addrString)
			if err != nil {
				writeEscapedBytes(w, addr)
				return
			}
		}
	} else if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}
	switch parsedURL.Scheme {
	case "bitcoin", "ftp", "geo", "http", "im", "irc", "ircs", "magnet", "mailto",
		"mms", "news", "nntp", "openpgp4fpr", "sip", "sftp", "sms", "smsto", "ssh",
		"tel", "urn", "webcal", "wtai", "xmpp", "https":
		w.WriteString(`<a`)
		if matrixURI, err := id.ProcessMatrixToURL(parsedURL); err == nil {
			writeAttribute(w, "href", matrixURI.String())
			writeAttribute(w, "class", matrixURIClassName(matrixURI)+" hicli-matrix-uri-plaintext")
		} else {
			writeAttribute(w, "target", "_blank")
			writeAttribute(w, "rel", "noreferrer noopener")
			writeAttribute(w, "href", parsedURL.String())
		}
		w.WriteByte('>')
		writeEscapedBytes(w, addr)
		w.WriteString("</a>")
	case "mxc":
		mxc := id.ContentURIString(parsedURL.String()).ParseOrIgnore()
		if !mxc.IsValid() {
			writeEscapedBytes(w, addr)
			return
		}
		w.WriteString("<a")
		writeAttribute(w, "class", "hicli-mxc-url")
		writeAttribute(w, "target", "_blank")
		writeAttribute(w, "data-mxc", mxc.String())
		writeAttribute(w, "href", fmt.Sprintf(HTMLSanitizerImgSrcTemplate, mxc.Homeserver, mxc.FileID))
		w.WriteByte('>')
		writeEscapedBytes(w, addr)
		w.WriteString("</a>")
	case "matrix":
		uri, err := id.ProcessMatrixURI(parsedURL)
		if err != nil {
			writeEscapedBytes(w, addr)
			return
		}
		w.WriteString("<a")
		writeAttribute(w, "class", matrixURIClassName(uri))
		writeAttribute(w, "href", uri.String())
		w.WriteByte('>')
		writeEscapedBytes(w, addr)
		w.WriteString("</a>")
	default:
		writeEscapedBytes(w, addr)
	}
}

func init() {
	if !slices.Contains(xurls.SchemesNoAuthority, "matrix") {
		xurls.SchemesNoAuthority = append(xurls.SchemesNoAuthority, "matrix")
	}
	if !slices.Contains(xurls.Schemes, "mxc") {
		xurls.Schemes = append(xurls.Schemes, "mxc")
	}
}

func linkifyAndWriteBytes(w *strings.Builder, s []byte) {
	mentions := plainUserOrAliasMentionRegex.FindAllIndex(s, -1)
	urls := xurls.Relaxed().FindAllIndex(s, -1)
	minIndex := 0
	for {
		mentionIdx, nextMentionStart, nextMentionEnd, hasMention := getNextItem(mentions, minIndex)
		urlIdx, nextURLStart, nextURLEnd, hasURL := getNextItem(urls, minIndex)
		if hasMention && (!hasURL || nextMentionStart <= nextURLStart) {
			writeEscapedBytes(w, s[minIndex:nextMentionStart])
			writeMention(w, s[nextMentionStart:nextMentionEnd])
			minIndex = nextMentionEnd
			mentions = mentions[mentionIdx:]
		} else if hasURL && (!hasMention || nextURLStart < nextMentionStart) {
			writeEscapedBytes(w, s[minIndex:nextURLStart])
			writeURL(w, s[nextURLStart:nextURLEnd])
			minIndex = nextURLEnd
			urls = urls[urlIdx:]
		} else {
			break
		}
	}
	writeEscapedBytes(w, s[minIndex:])
}

const escapedChars = "&'<>\"\r"

func getEscapeCharacter(b byte) string {
	switch b {
	case '&':
		return "&amp;"
	case '\'':
		// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
		return "&#39;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		// "&#34;" is shorter than "&quot;".
		return "&#34;"
	case '\r':
		return "&#13;"
	default:
		panic("unrecognized escape character")
	}
}

func writeEscapedBytes(w *strings.Builder, s []byte) {
	i := bytes.IndexAny(s, escapedChars)
	for i != -1 {
		w.Write(s[:i])
		w.WriteString(getEscapeCharacter(s[i]))
		s = s[i+1:]
		i = bytes.IndexAny(s, escapedChars)
	}
	w.Write(s)
}

func writeEscapedString(w *strings.Builder, s string) {
	i := strings.IndexAny(s, escapedChars)
	for i != -1 {
		w.WriteString(s[:i])
		w.WriteString(getEscapeCharacter(s[i]))
		s = s[i+1:]
		i = strings.IndexAny(s, escapedChars)
	}
	w.WriteString(s)
}

func writeAttribute(w *strings.Builder, key, value string) {
	w.WriteByte(' ')
	w.WriteString(key)
	w.WriteString(`="`)
	writeEscapedString(w, value)
	w.WriteByte('"')
}

func matrixURIClassName(uri *id.MatrixURI) string {
	switch uri.Sigil1 {
	case '@':
		return "hicli-matrix-uri hicli-matrix-uri-user"
	case '#':
		return "hicli-matrix-uri hicli-matrix-uri-room-alias"
	case '!':
		if uri.Sigil2 == '$' {
			return "hicli-matrix-uri hicli-matrix-uri-event-id"
		}
		return "hicli-matrix-uri hicli-matrix-uri-room-id"
	default:
		return "hicli-matrix-uri hicli-matrix-uri-unknown"
	}
}

func writeA(w *strings.Builder, attr []html.Attribute) (mxc id.ContentURI) {
	w.WriteString("<a")
	href := parseAAttributes(attr)
	if href == "" {
		return
	}
	parsedURL, err := url.Parse(href)
	if err != nil {
		return
	}
	newTab := true
	switch parsedURL.Scheme {
	case "bitcoin", "ftp", "geo", "http", "im", "irc", "ircs", "magnet", "mailto",
		"mms", "news", "nntp", "openpgp4fpr", "sip", "sftp", "sms", "smsto", "ssh",
		"tel", "urn", "webcal", "wtai", "xmpp":
		// allowed
	case "https":
		if parsedURL.Host == "matrix.to" {
			uri, err := id.ProcessMatrixToURL(parsedURL)
			if err != nil {
				return
			}
			href = uri.String()
			newTab = false
			writeAttribute(w, "class", matrixURIClassName(uri))
		}
	case "matrix":
		uri, err := id.ProcessMatrixURI(parsedURL)
		if err != nil {
			return
		}
		href = uri.String()
		newTab = false
		writeAttribute(w, "class", matrixURIClassName(uri))
	case "mxc":
		mxc = id.ContentURIString(href).ParseOrIgnore()
		if !mxc.IsValid() {
			mxc = id.ContentURI{}
			return
		}
		writeAttribute(w, "class", "hicli-mxc-url")
		writeAttribute(w, "target", "_blank")
		writeAttribute(w, "data-mxc", mxc.String())
		href = fmt.Sprintf(HTMLSanitizerImgSrcTemplate, mxc.Homeserver, mxc.FileID)
	default:
		return
	}
	writeAttribute(w, "href", href)
	if newTab {
		writeAttribute(w, "target", "_blank")
		writeAttribute(w, "rel", "noreferrer noopener")
	}
	return
}

var HTMLSanitizerImgSrcTemplate = "mxc://%s/%s"

func writeImg(w *strings.Builder, attr []html.Attribute) id.ContentURI {
	src, alt, title, isCustomEmoji, width, height := parseImgAttributes(attr)
	mxc := id.ContentURIString(src).ParseOrIgnore()
	if !mxc.IsValid() {
		w.WriteString("<span")
		writeAttribute(w, "class", "hicli-inline-img-fallback hicli-invalid-inline-img")
		w.WriteString(">")
		writeEscapedString(w, alt)
		w.WriteString("</span>")
		return id.ContentURI{}
	}
	url := fmt.Sprintf(HTMLSanitizerImgSrcTemplate, mxc.Homeserver, mxc.FileID)

	w.WriteString("<a")
	writeAttribute(w, "class", "hicli-inline-img-fallback hicli-mxc-url")
	writeAttribute(w, "title", title)
	writeAttribute(w, "style", "display: none;")
	writeAttribute(w, "target", "_blank")
	writeAttribute(w, "data-mxc", mxc.String())
	writeAttribute(w, "href", url)
	w.WriteString(">")
	writeEscapedString(w, alt)
	w.WriteString("</a>")

	w.WriteString("<img")
	writeAttribute(w, "alt", alt)
	if title != "" {
		writeAttribute(w, "title", title)
	}
	writeAttribute(w, "src", url)
	writeAttribute(w, "loading", "lazy")
	if isCustomEmoji {
		writeAttribute(w, "class", "hicli-inline-img hicli-custom-emoji")
	} else if cWidth, cHeight, sizeOK := calculateMediaSize(width, height); sizeOK {
		writeAttribute(w, "class", "hicli-inline-img hicli-sized-inline-img")
		writeAttribute(w, "style", fmt.Sprintf("width: %.2fpx; height: %.2fpx;", cWidth, cHeight))
	} else {
		writeAttribute(w, "class", "hicli-inline-img hicli-sizeless-inline-img")
	}
	return mxc
}

func writeSpan(w *strings.Builder, attr []html.Attribute) {
	bgColor, textColor, spoiler, isSpoiler := parseSpanAttributes(attr)
	if isSpoiler && spoiler != "" {
		w.WriteString(`<span class="spoiler-reason">`)
		w.WriteString(spoiler)
		w.WriteString("</span>")
	}
	w.WriteByte('<')
	w.WriteString("span")
	if isSpoiler {
		writeAttribute(w, "class", "hicli-spoiler")
	}
	var style string
	if bgColor != "" {
		style += fmt.Sprintf("background-color: %s;", bgColor)
	}
	if textColor != "" {
		style += fmt.Sprintf("color: %s;", textColor)
	}
	if style != "" {
		writeAttribute(w, "style", style)
	}
}

type tagStack []atom.Atom

func (ts *tagStack) contains(tags ...atom.Atom) bool {
	for i := len(*ts) - 1; i >= 0; i-- {
		for _, tag := range tags {
			if (*ts)[i] == tag {
				return true
			}
		}
	}
	return false
}

func (ts *tagStack) push(tag atom.Atom) {
	*ts = append(*ts, tag)
}

func (ts *tagStack) pop(tag atom.Atom) bool {
	if len(*ts) > 0 && (*ts)[len(*ts)-1] == tag {
		*ts = (*ts)[:len(*ts)-1]
		return true
	}
	return false
}

func getCodeBlockLanguage(token html.Token) string {
	for _, attr := range token.Attr {
		if attr.Key == "class" {
			match := languageRegex.FindStringSubmatch(attr.Val)
			if len(match) == 2 {
				return match[1]
			}
		}
	}
	return ""
}

const builderPreallocBuffer = 100

func sanitizeAndLinkifyHTML(body string) (string, []id.ContentURI, error) {
	tz := html.NewTokenizer(strings.NewReader(body))
	var built strings.Builder
	built.Grow(len(body) + builderPreallocBuffer)
	var codeBlock *strings.Builder
	var codeBlockLanguage string
	var inlineImages []id.ContentURI
	ts := make(tagStack, 0, 2)
Loop:
	for {
		switch tz.Next() {
		case html.ErrorToken:
			err := tz.Err()
			if errors.Is(err, io.EOF) {
				break Loop
			}
			return "", nil, err
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tz.Token()
			if codeBlock != nil {
				if token.DataAtom == atom.Code {
					codeBlockLanguage = getCodeBlockLanguage(token)
				}
				// Don't allow any tags inside code blocks
				continue
			}
			if !tagIsAllowed(token.DataAtom) {
				continue
			}
			switch token.DataAtom {
			case atom.Pre:
				codeBlock = &strings.Builder{}
				continue
			case atom.A:
				mxc := writeA(&built, token.Attr)
				if !mxc.IsEmpty() {
					inlineImages = append(inlineImages, mxc)
				}
			case atom.Img:
				mxc := writeImg(&built, token.Attr)
				if !mxc.IsEmpty() {
					inlineImages = append(inlineImages, mxc)
				}
			case atom.Div:
				math, ok := getAttribute(token.Attr, "data-mx-maths")
				if ok {
					built.WriteString(`<hicli-math displaymode="block"`)
					writeAttribute(&built, "latex", math)
					token.DataAtom = atom.Math
				} else {
					built.WriteString("<div")
				}
			case atom.Span, atom.Font:
				math, ok := getAttribute(token.Attr, "data-mx-maths")
				if ok && token.DataAtom == atom.Span {
					built.WriteString(`<hicli-math displaymode="inline"`)
					writeAttribute(&built, "latex", math)
					token.DataAtom = atom.Math
				} else {
					writeSpan(&built, token.Attr)
				}
			case atom.Code:
				built.WriteString(`<code class="hicli-inline-code"`)
			case atom.Input:
				inputType, ok := getAttribute(token.Attr, "type")
				if !ok || inputType != "checkbox" {
					continue
				}
				_, checked := getAttribute(token.Attr, "checked")
				// TODO allow checking checkboxes on own events
				built.WriteString(`<input type="checkbox" class="hicli-checkbox" disabled`)
				if checked {
					built.WriteString(" checked")
				}
			default:
				built.WriteByte('<')
				built.WriteString(token.Data)
				for _, attr := range token.Attr {
					if attributeIsAllowed(token.DataAtom, attr) {
						writeAttribute(&built, attr.Key, attr.Val)
					}
				}
			}
			if token.Type == html.SelfClosingTagToken {
				built.WriteByte('/')
			}
			built.WriteByte('>')
			if !isSelfClosing(token.DataAtom) && token.Type != html.SelfClosingTagToken {
				ts.push(token.DataAtom)
			}
		case html.EndTagToken:
			tagName, _ := tz.TagName()
			tag := atom.Lookup(tagName)
			if !tagIsAllowed(tag) {
				continue
			}
			if tag == atom.Pre && codeBlock != nil {
				writeCodeBlock(&built, codeBlockLanguage, codeBlock)
				codeBlockLanguage = ""
				codeBlock = nil
			} else if ts.pop(tag) {
				// TODO instead of only popping when the last tag in the stack matches, this should go through the stack
				//      and close all tags until it finds the matching tag
				if tag == atom.Font {
					built.WriteString("</span>")
				} else {
					built.WriteString("</")
					built.Write(tagName)
					built.WriteByte('>')
				}
			} else if (tag == atom.Span || tag == atom.Div) && ts.pop(atom.Math) {
				built.WriteString("</hicli-math>")
			}
		case html.TextToken:
			if codeBlock != nil {
				codeBlock.Write(tz.Text())
			} else if ts.contains(atom.Pre, atom.Code, atom.A) {
				writeEscapedBytes(&built, tz.Text())
			} else {
				linkifyAndWriteBytes(&built, tz.Text())
			}
		case html.DoctypeToken, html.CommentToken:
			// ignore
		}
	}
	slices.Reverse(ts)
	for _, t := range ts {
		built.WriteString("</")
		built.WriteString(t.String())
		built.WriteByte('>')
	}
	return built.String(), inlineImages, nil
}

var CodeBlockFormatter = chromahtml.New(
	chromahtml.WithClasses(true),
	chromahtml.WithLineNumbers(true),
)

type lineRewriter struct {
	w *strings.Builder
}

var lineNumberRewriter = regexp.MustCompile(`<span class="ln">(\s*\d+)</span>`)
var lineNumberReplacement = []byte(`<span class="ln" data-linenum="$1"></span>`)

func (lr *lineRewriter) Write(p []byte) (n int, err error) {
	n = len(p)
	p = lineNumberRewriter.ReplaceAll(p, lineNumberReplacement)
	lr.w.Write(p)
	return
}

func writeCodeBlock(w *strings.Builder, language string, block *strings.Builder) {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	iter, err := lexer.Tokenise(nil, block.String())
	if err != nil {
		w.WriteString("<pre><code")
		if language != "" {
			writeAttribute(w, "class", "language-"+language)
		}
		w.WriteByte('>')
		writeEscapedString(w, block.String())
		w.WriteString("</code></pre>")
		return
	}
	err = CodeBlockFormatter.Format(&lineRewriter{w}, styles.Fallback, iter)
	if err != nil {
		// This should never fail
		panic(err)
	}
}
