// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"go.mau.fi/util/exerrors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SkinVariation struct {
	Unified        string  `json:"unified"`
	NonQualified   *string `json:"non_qualified"`
	Image          string  `json:"image"`
	SheetX         int     `json:"sheet_x"`
	SheetY         int     `json:"sheet_y"`
	AddedIn        string  `json:"added_in"`
	HasImgApple    bool    `json:"has_img_apple"`
	HasImgGoogle   bool    `json:"has_img_google"`
	HasImgTwitter  bool    `json:"has_img_twitter"`
	HasImgFacebook bool    `json:"has_img_facebook"`
	Obsoletes      string  `json:"obsoletes,omitempty"`
	ObsoletedBy    string  `json:"obsoleted_by,omitempty"`
}

type Emoji struct {
	Name           string                    `json:"name"`
	Unified        string                    `json:"unified"`
	NonQualified   *string                   `json:"non_qualified"`
	Docomo         *string                   `json:"docomo"`
	Au             *string                   `json:"au"`
	Softbank       *string                   `json:"softbank"`
	Google         *string                   `json:"google"`
	Image          string                    `json:"image"`
	SheetX         int                       `json:"sheet_x"`
	SheetY         int                       `json:"sheet_y"`
	ShortName      string                    `json:"short_name"`
	ShortNames     []string                  `json:"short_names"`
	Text           *string                   `json:"text"`
	Texts          []string                  `json:"texts"`
	Category       string                    `json:"category"`
	Subcategory    string                    `json:"subcategory"`
	SortOrder      int                       `json:"sort_order"`
	AddedIn        string                    `json:"added_in"`
	HasImgApple    bool                      `json:"has_img_apple"`
	HasImgGoogle   bool                      `json:"has_img_google"`
	HasImgTwitter  bool                      `json:"has_img_twitter"`
	HasImgFacebook bool                      `json:"has_img_facebook"`
	SkinVariations map[string]*SkinVariation `json:"skin_variations,omitempty"`
	Obsoletes      string                    `json:"obsoletes,omitempty"`
	ObsoletedBy    string                    `json:"obsoleted_by,omitempty"`
}

func unifiedToUnicode(input string) string {
	parts := strings.Split(input, "-")
	output := make([]rune, len(parts))
	for i, part := range parts {
		output[i] = rune(exerrors.Must(strconv.ParseInt(part, 16, 32)))
	}
	return string(output)
}

func getVariationSequences() (output map[string]struct{}) {
	variationSequences := exerrors.Must(http.Get("https://www.unicode.org/Public/15.1.0/ucd/emoji/emoji-variation-sequences.txt"))
	buf := bufio.NewReader(variationSequences.Body)
	output = make(map[string]struct{})
	for {
		line, err := buf.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			panic(err)
		}
		parts := strings.Split(line, "; ")
		if len(parts) < 2 || parts[1] != "emoji style" {
			continue
		}
		unifiedParts := strings.Split(parts[0], " ")
		output[unifiedParts[0]] = struct{}{}
	}
	return
}

type outputEmoji struct {
	Unicode    string   `json:"u"`
	Category   int      `json:"c"`
	Title      string   `json:"t"`
	Name       string   `json:"n"`
	Shortcodes []string `json:"s"`
}

type outputData struct {
	Emojis     []*outputEmoji `json:"e"`
	Categories []string       `json:"c"`
}

type EmojibaseEmoji struct {
	Hexcode string `json:"hexcode"`
	Label   string `json:"label"`
}

var titler = cases.Title(language.English)

func getEmojibaseNames() map[string]string {
	var emojibaseEmojis []EmojibaseEmoji
	resp := exerrors.Must(http.Get("https://github.com/milesj/emojibase/raw/refs/heads/master/packages/data/en/compact.raw.json"))
	exerrors.PanicIfNotNil(json.NewDecoder(resp.Body).Decode(&emojibaseEmojis))
	output := make(map[string]string, len(emojibaseEmojis))
	for _, emoji := range emojibaseEmojis {
		output[emoji.Hexcode] = titler.String(emoji.Label)
	}
	return output
}

func main() {
	var emojis []Emoji
	resp := exerrors.Must(http.Get("https://raw.githubusercontent.com/iamcal/emoji-data/master/emoji.json"))
	exerrors.PanicIfNotNil(json.NewDecoder(resp.Body).Decode(&emojis))
	vs := getVariationSequences()
	names := getEmojibaseNames()
	slices.SortFunc(emojis, func(a, b Emoji) int {
		return a.SortOrder - b.SortOrder
	})

	data := &outputData{
		Emojis:     make([]*outputEmoji, len(emojis)),
		Categories: []string{"Activities", "Animals & Nature", "Component", "Flags", "Food & Drink", "Objects", "People & Body", "Smileys & Emotion", "Symbols", "Travel & Places"},
	}
	for i, emoji := range emojis {
		wrapped := &outputEmoji{
			Unicode:    unifiedToUnicode(emoji.Unified),
			Name:       emoji.ShortName,
			Shortcodes: emoji.ShortNames,
			Category:   slices.Index(data.Categories, emoji.Category),
			Title:      names[emoji.Unified],
		}
		if wrapped.Category == -1 {
			panic(fmt.Errorf("unknown category %q", emoji.Category))
		}
		for i, short := range wrapped.Shortcodes {
			wrapped.Shortcodes[i] = strings.ReplaceAll(short, "_", "")
		}
		if wrapped.Title == "" {
			wrapped.Title = titler.String(emoji.Name)
		}
		if _, needsVariation := vs[emoji.Unified]; needsVariation {
			wrapped.Unicode += "\ufe0f"
		}
		data.Emojis[i] = wrapped
	}
	file := exerrors.Must(os.OpenFile("data.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644))
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	exerrors.PanicIfNotNil(enc.Encode(data))
	exerrors.PanicIfNotNil(file.Close())
}
