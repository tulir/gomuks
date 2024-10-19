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
		atom.Details, atom.Summary:
		return true
	default:
		return false
	}
}

func isSelfClosing(tag atom.Atom) bool {
	switch tag {
	case atom.Img, atom.Br, atom.Hr:
		return true
	default:
		return false
	}
}

var languageRegex = regexp.MustCompile(`^language-[a-zA-Z0-9-]+$`)
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

func parseSpanAttributes(attrs []html.Attribute) (bgColor, textColor, spoiler, maths string, isSpoiler bool) {
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
		case "data-mx-maths":
			maths = attr.Val
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
	case "mxc", "matrix":
		// TODO
		fallthrough
	default:
		writeEscapedBytes(w, addr)
		return
	}
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

func writeEscapedBytes(w *strings.Builder, s []byte) {
	i := bytes.IndexAny(s, escapedChars)
	for i != -1 {
		w.Write(s[:i])
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '\'':
			// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
			esc = "&#39;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			// "&#34;" is shorter than "&quot;".
			esc = "&#34;"
		case '\r':
			esc = "&#13;"
		default:
			panic("unrecognized escape character")
		}
		s = s[i+1:]
		w.WriteString(esc)
		i = bytes.IndexAny(s, escapedChars)
	}
	w.Write(s)
}

func writeEscapedString(w *strings.Builder, s string) {
	i := strings.IndexAny(s, escapedChars)
	for i != -1 {
		w.WriteString(s[:i])
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '\'':
			// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
			esc = "&#39;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			// "&#34;" is shorter than "&quot;".
			esc = "&#34;"
		case '\r':
			esc = "&#13;"
		default:
			panic("unrecognized escape character")
		}
		s = s[i+1:]
		w.WriteString(esc)
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

func writeA(w *strings.Builder, attr []html.Attribute) {
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
		mxc := id.ContentURIString(href).ParseOrIgnore()
		if !mxc.IsValid() {
			return
		}
		href = fmt.Sprintf(HTMLSanitizerImgSrcTemplate, mxc.Homeserver, mxc.FileID)
	default:
		return
	}
	writeAttribute(w, "href", href)
	if newTab {
		writeAttribute(w, "target", "_blank")
		writeAttribute(w, "rel", "noreferrer noopener")
	}
}

var HTMLSanitizerImgSrcTemplate = "mxc://%s/%s"

func writeImg(w *strings.Builder, attr []html.Attribute) {
	src, alt, title, isCustomEmoji, width, height := parseImgAttributes(attr)
	w.WriteString("<img")
	writeAttribute(w, "alt", alt)
	if title != "" {
		writeAttribute(w, "title", title)
	}
	mxc := id.ContentURIString(src).ParseOrIgnore()
	if !mxc.IsValid() {
		return
	}
	writeAttribute(w, "src", fmt.Sprintf(HTMLSanitizerImgSrcTemplate, mxc.Homeserver, mxc.FileID))
	writeAttribute(w, "loading", "lazy")
	if isCustomEmoji {
		writeAttribute(w, "class", "hicli-custom-emoji")
	} else if cWidth, cHeight, sizeOK := calculateMediaSize(width, height); sizeOK {
		writeAttribute(w, "class", "hicli-sized-inline-img")
		writeAttribute(w, "style", fmt.Sprintf("width: %.2fpx; height: %.2fpx;", cWidth, cHeight))
	} else {
		writeAttribute(w, "class", "hicli-sizeless-inline-img")
	}
}

func writeSpan(w *strings.Builder, attr []html.Attribute) {
	bgColor, textColor, spoiler, _, isSpoiler := parseSpanAttributes(attr)
	if isSpoiler && spoiler != "" {
		w.WriteString(`<span class="spoiler-reason">`)
		w.WriteString(spoiler)
		w.WriteString(" </span>")
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

func sanitizeAndLinkifyHTML(body string) (string, error) {
	tz := html.NewTokenizer(strings.NewReader(body))
	var built strings.Builder
	ts := make(tagStack, 2)
Loop:
	for {
		switch tz.Next() {
		case html.ErrorToken:
			err := tz.Err()
			if errors.Is(err, io.EOF) {
				break Loop
			}
			return "", err
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tz.Token()
			if !tagIsAllowed(token.DataAtom) {
				continue
			}
			tagIsSelfClosing := isSelfClosing(token.DataAtom)
			if token.Type == html.SelfClosingTagToken && !tagIsSelfClosing {
				continue
			}
			switch token.DataAtom {
			case atom.A:
				writeA(&built, token.Attr)
			case atom.Img:
				writeImg(&built, token.Attr)
			case atom.Span, atom.Font:
				writeSpan(&built, token.Attr)
			default:
				built.WriteByte('<')
				built.WriteString(token.Data)
				for _, attr := range token.Attr {
					if attributeIsAllowed(token.DataAtom, attr) {
						writeAttribute(&built, attr.Key, attr.Val)
					}
				}
			}
			built.WriteByte('>')
			if !tagIsSelfClosing {
				ts.push(token.DataAtom)
			}
		case html.EndTagToken:
			tagName, _ := tz.TagName()
			tag := atom.Lookup(tagName)
			if tagIsAllowed(tag) && ts.pop(tag) {
				if tag == atom.Font {
					built.WriteString("</span>")
				} else {
					built.WriteString("</")
					built.Write(tagName)
					built.WriteByte('>')
				}
			}
		case html.TextToken:
			if ts.contains(atom.Pre, atom.Code, atom.A) {
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
	return built.String(), nil
}