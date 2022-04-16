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

	"github.com/mattn/go-runewidth"

	"go.mau.fi/mauview"

	"maunium.net/go/gomuks/ui/widget"
)

type TextEntity struct {
	*BaseEntity
	// Text in this entity.
	Text string

	buffer []string
}

// NewTextEntity creates a new text-only Entity.
func NewTextEntity(text string) *TextEntity {
	return &TextEntity{
		BaseEntity: &BaseEntity{
			Tag: "text",
		},
		Text: text,
	}
}

func (te *TextEntity) IsEmpty() bool {
	return len(te.Text) == 0
}

func (te *TextEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	te.BaseEntity = te.BaseEntity.AdjustStyle(fn, reason).(*BaseEntity)
	return te
}

func (te *TextEntity) Clone() Entity {
	return &TextEntity{
		BaseEntity: te.BaseEntity.Clone().(*BaseEntity),
		Text:       te.Text,
	}
}

func (te *TextEntity) PlainText() string {
	return te.Text
}

func (te *TextEntity) String() string {
	return fmt.Sprintf("&html.TextEntity{Text=%s, Base=%s},\n", te.Text, te.BaseEntity)
}

func (te *TextEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	width, _ := screen.Size()
	x := te.startX
	for y, line := range te.buffer {
		widget.WriteLine(screen, mauview.AlignLeft, line, x, y, width, te.Style)
		x = 0
	}
}

func (te *TextEntity) CalculateBuffer(width, startX int, ctx DrawContext) int {
	te.BaseEntity.CalculateBuffer(width, startX, ctx)
	if len(te.Text) == 0 {
		return te.startX
	}
	te.height = 0
	te.prevWidth = width
	if te.buffer == nil {
		te.buffer = []string{}
	}
	bufPtr := 0
	text := te.Text
	textStartX := te.startX
	for {
		// TODO add option no wrap and character wrap options
		extract := runewidth.Truncate(text, width-textStartX, "")
		extract, wordWrapped := trim(extract, text, ctx.BareMessages)
		if !wordWrapped && textStartX > 0 {
			if bufPtr < len(te.buffer) {
				te.buffer[bufPtr] = ""
			} else {
				te.buffer = append(te.buffer, "")
			}
			bufPtr++
			textStartX = 0
			continue
		}
		if bufPtr < len(te.buffer) {
			te.buffer[bufPtr] = extract
		} else {
			te.buffer = append(te.buffer, extract)
		}
		bufPtr++
		text = text[len(extract):]
		if len(text) == 0 {
			te.buffer = te.buffer[:bufPtr]
			te.height += len(te.buffer)
			// This entity is over, return the startX for the next entity
			if te.Block {
				// ...except if it's a block entity
				return 0
			}
			return textStartX + runewidth.StringWidth(extract)
		}
		textStartX = 0
	}
}

var (
	boundaryPattern     = regexp.MustCompile(`([[:punct:]]\s*|\s+)`)
	bareBoundaryPattern = regexp.MustCompile(`(\s+)`)
	spacePattern        = regexp.MustCompile(`\s+`)
)

func trim(extract, full string, bare bool) (string, bool) {
	if len(extract) == len(full) {
		return extract, true
	}
	if spaces := spacePattern.FindStringIndex(full[len(extract):]); spaces != nil && spaces[0] == 0 {
		extract = full[:len(extract)+spaces[1]]
	}
	regex := boundaryPattern
	if bare {
		regex = bareBoundaryPattern
	}
	matches := regex.FindAllStringIndex(extract, -1)
	if len(matches) > 0 {
		if match := matches[len(matches)-1]; len(match) >= 2 {
			if until := match[1]; until < len(extract) {
				extract = extract[:until]
				return extract, true
			}
		}
	}
	return extract, len(extract) > 0 && extract[len(extract)-1] == ' '
}
