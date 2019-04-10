// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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
	"strings"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"
)

// AdjustStyleFunc is a lambda function type to edit an existing tcell Style.
type AdjustStyleFunc func(tcell.Style) tcell.Style

type Entity interface {
	// AdjustStyle recursively changes the style of the entity and all its children.
	AdjustStyle(AdjustStyleFunc) Entity
	// Draw draws the entity onto the given mauview Screen.
	Draw(screen mauview.Screen)
	// IsBlock returns whether or not it's a block-type entity.
	IsBlock() bool
	// GetTag returns the HTML tag of the entity.
	GetTag() string
	// PlainText returns the plaintext content in the entity and all its children.
	PlainText() string
	// String returns a string representation of the entity struct.
	String() string
	// Clone creates a deep copy of the entity.
	Clone() Entity

	// Height returns the render height of the entity.
	Height() int
	// CalculateBuffer prepares the entity and all its children for rendering with the given parameters
	CalculateBuffer(width, startX int, bare bool) int

	getStartX() int
}

type BaseEntity struct {
	// The HTML tag of this entity.
	Tag string
	// Text in this entity.
	Text string
	// Style for this entity.
	Style tcell.Style
	// Child entities.
	Children []Entity
	// Whether or not this is a block-type entity.
	Block bool
	// Number of cells to indent children.
	Indent int

	// Height to use for entity if both text and children are empty.
	DefaultHeight int

	buffer    []string
	prevWidth int
	startX    int
	height    int
}

// NewTextEntity creates a new text-only Entity.
func NewTextEntity(text string) *BaseEntity {
	return &BaseEntity{
		Tag:  "text",
		Text: text,
	}
}

// AdjustStyle recursively changes the style of this entity and all its children.
func (he *BaseEntity) AdjustStyle(fn AdjustStyleFunc) Entity {
	for _, child := range he.Children {
		child.AdjustStyle(fn)
	}
	he.Style = fn(he.Style)
	return he
}

// IsBlock returns whether or not this is a block-type entity.
func (he *BaseEntity) IsBlock() bool {
	return he.Block
}

// GetTag returns the HTML tag of this entity.
func (he *BaseEntity) GetTag() string {
	return he.Tag
}

// Height returns the render height of this entity.
func (he *BaseEntity) Height() int {
	return he.height
}

func (he *BaseEntity) getStartX() int {
	return he.startX
}

// Clone creates a deep copy of this entity.
func (he *BaseEntity) Clone() Entity {
	children := make([]Entity, len(he.Children))
	for i, child := range he.Children {
		children[i] = child.Clone()
	}
	return &BaseEntity{
		Tag:           he.Tag,
		Text:          he.Text,
		Style:         he.Style,
		Children:      children,
		Block:         he.Block,
		Indent:        he.Indent,
		DefaultHeight: he.DefaultHeight,
	}
}

// String returns a textual representation of this BaseEntity struct.
func (he *BaseEntity) String() string {
	var buf strings.Builder
	buf.WriteString("&html.BaseEntity{\n")
	_, _ = fmt.Fprintf(&buf, `    Tag="%s", Style=%d, Block=%t, Indent=%d, startX=%d, height=%d,`,
		he.Tag, he.Style, he.Block, he.Indent, he.startX, he.height)
	buf.WriteRune('\n')
	_, _ = fmt.Fprintf(&buf, `    Buffer=["%s"]`, strings.Join(he.buffer, "\", \""))
	if len(he.Text) > 0 {
		buf.WriteString(",\n")
		_, _ = fmt.Fprintf(&buf, `    Text="%s"`, he.Text)
	}
	if len(he.Children) > 0 {
		buf.WriteString(",\n")
		buf.WriteString("    Children={")
		for _, child := range he.Children {
			buf.WriteString("\n        ")
			buf.WriteString(strings.Join(strings.Split(strings.TrimRight(child.String(), "\n"), "\n"), "\n        "))
		}
		buf.WriteString("\n    },")
	}
	buf.WriteString("\n},\n")
	return buf.String()
}

// PlainText returns the plaintext content in this entity and all its children.
func (he *BaseEntity) PlainText() string {
	if len(he.Children) == 0 {
		return he.Text
	}
	var buf strings.Builder
	buf.WriteString(he.Text)
	newlined := false
	for _, child := range he.Children {
		text := child.PlainText()
		if !strings.HasPrefix(text, "\n") && child.IsBlock() && !newlined {
			buf.WriteRune('\n')
		}
		newlined = false
		buf.WriteString(text)
		if child.IsBlock() {
			if !strings.HasSuffix(text, "\n") {
				buf.WriteRune('\n')
			}
			newlined = true
		}
	}
	return strings.TrimSpace(buf.String())
}

// Draw draws this entity onto the given mauview Screen.
func (he *BaseEntity) Draw(screen mauview.Screen) {
	width, _ := screen.Size()
	if len(he.buffer) > 0 {
		x := he.startX
		for y, line := range he.buffer {
			widget.WriteLine(screen, mauview.AlignLeft, line, x, y, width, he.Style)
			x = 0
		}
	}
	if len(he.Children) > 0 {
		prevBreak := false
		proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: he.Indent, Width: width - he.Indent, Style: he.Style}
		for i, entity := range he.Children {
			if i != 0 && entity.getStartX() == 0 {
				proxyScreen.OffsetY++
			}
			proxyScreen.Height = entity.Height()
			entity.Draw(proxyScreen)
			proxyScreen.SetStyle(he.Style)
			proxyScreen.OffsetY += entity.Height() - 1
			_, isBreak := entity.(*BreakEntity)
			if prevBreak && isBreak {
				proxyScreen.OffsetY++
			}
			prevBreak = isBreak
		}
	}
}

// CalculateBuffer prepares this entity and all its children for rendering with the given parameters
func (he *BaseEntity) CalculateBuffer(width, startX int, bare bool) int {
	he.startX = startX
	if he.Block {
		he.startX = 0
	}
	he.height = 0
	if len(he.Children) > 0 {
		childStartX := he.startX
		prevBreak := false
		for _, entity := range he.Children {
			if entity.IsBlock() || childStartX == 0 || he.height == 0 {
				he.height++
			}
			childStartX = entity.CalculateBuffer(width-he.Indent, childStartX, bare)
			he.height += entity.Height() - 1
			_, isBreak := entity.(*BreakEntity)
			if prevBreak && isBreak {
				he.height++
			}
			prevBreak = isBreak
		}
		if len(he.Text) == 0 && !he.Block {
			return childStartX
		}
	}
	if len(he.Text) > 0 {
		he.prevWidth = width
		if he.buffer == nil {
			he.buffer = []string{}
		}
		bufPtr := 0
		text := he.Text
		textStartX := he.startX
		for {
			// TODO add option no wrap and character wrap options
			extract := runewidth.Truncate(text, width-textStartX, "")
			extract, wordWrapped := trim(extract, text, bare)
			if !wordWrapped && textStartX > 0 {
				if bufPtr < len(he.buffer) {
					he.buffer[bufPtr] = ""
				} else {
					he.buffer = append(he.buffer, "")
				}
				bufPtr++
				textStartX = 0
				continue
			}
			if bufPtr < len(he.buffer) {
				he.buffer[bufPtr] = extract
			} else {
				he.buffer = append(he.buffer, extract)
			}
			bufPtr++
			text = text[len(extract):]
			if len(text) == 0 {
				he.buffer = he.buffer[:bufPtr]
				he.height += len(he.buffer)
				// This entity is over, return the startX for the next entity
				if he.Block {
					// ...except if it's a block entity
					return 0
				}
				return textStartX + runewidth.StringWidth(extract)
			}
			textStartX = 0
		}
	}
	if len(he.Text) == 0 && len(he.Children) == 0 {
		he.height = he.DefaultHeight
	}
	return he.startX
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
