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
	"strings"

	"go.mau.fi/mauview"
)

type ContainerEntity struct {
	*BaseEntity

	// The children of this container entity.
	Children []Entity
	// Number of cells to indent children.
	Indent int
}

func (ce *ContainerEntity) IsEmpty() bool {
	return len(ce.Children) == 0
}

// PlainText returns the plaintext content in this entity and all its children.
func (ce *ContainerEntity) PlainText() string {
	if len(ce.Children) == 0 {
		return ""
	}
	var buf strings.Builder
	newlined := false
	for _, child := range ce.Children {
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

// AdjustStyle recursively changes the style of this entity and all its children.
func (ce *ContainerEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	for _, child := range ce.Children {
		child.AdjustStyle(fn, reason)
	}
	ce.Style = fn(ce.Style)
	return ce
}

// Clone creates a deep copy of this base entity.
func (ce *ContainerEntity) Clone() Entity {
	children := make([]Entity, len(ce.Children))
	for i, child := range ce.Children {
		children[i] = child.Clone()
	}
	return &ContainerEntity{
		BaseEntity: ce.BaseEntity.Clone().(*BaseEntity),
		Children:   children,
		Indent:     ce.Indent,
	}
}

// String returns a textual representation of this BaseEntity struct.
func (ce *ContainerEntity) String() string {
	if len(ce.Children) == 0 {
		return fmt.Sprintf(`&html.ContainerEntity{Base=%s, Indent=%d, Children=[]}`, ce.BaseEntity, ce.Indent)
	}
	var buf strings.Builder
	_, _ = fmt.Fprintf(&buf, `&html.ContainerEntity{Base=%s, Indent=%d, Children=[`, ce.BaseEntity, ce.Indent)
	for _, child := range ce.Children {
		buf.WriteString("\n    ")
		buf.WriteString(strings.Join(strings.Split(strings.TrimRight(child.String(), "\n"), "\n"), "\n    "))
	}
	buf.WriteString("\n]},")
	return buf.String()
}

// Draw draws this entity onto the given mauview Screen.
func (ce *ContainerEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	if len(ce.Children) == 0 {
		return
	}
	width, _ := screen.Size()
	prevBreak := false
	proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: ce.Indent, Width: width - ce.Indent, Style: ce.Style}
	for i, entity := range ce.Children {
		if i != 0 && entity.getStartX() == 0 {
			proxyScreen.OffsetY++
		}
		proxyScreen.Height = entity.Height()
		entity.Draw(proxyScreen, ctx)
		proxyScreen.SetStyle(ce.Style)
		proxyScreen.OffsetY += entity.Height() - 1
		_, isBreak := entity.(*BreakEntity)
		if prevBreak && isBreak {
			proxyScreen.OffsetY++
		}
		prevBreak = isBreak
	}
}

// CalculateBuffer prepares this entity and all its children for rendering with the given parameters
func (ce *ContainerEntity) CalculateBuffer(width, startX int, ctx DrawContext) int {
	ce.BaseEntity.CalculateBuffer(width, startX, ctx)
	if len(ce.Children) > 0 {
		ce.height = 0
		childStartX := ce.startX
		prevBreak := false
		for _, entity := range ce.Children {
			if entity.IsBlock() || childStartX == 0 || ce.height == 0 {
				ce.height++
			}
			childStartX = entity.CalculateBuffer(width-ce.Indent, childStartX, ctx)
			ce.height += entity.Height() - 1
			_, isBreak := entity.(*BreakEntity)
			if prevBreak && isBreak {
				ce.height++
			}
			prevBreak = isBreak
		}
		if !ce.Block {
			return childStartX
		}
	}
	return ce.startX
}
