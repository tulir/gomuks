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

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"
)

type BaseEntity struct {
	// The HTML tag of this entity.
	Tag string
	// Style for this entity.
	Style tcell.Style
	// Whether or not this is a block-type entity.
	Block bool
	// Height to use for entity if both text and children are empty.
	DefaultHeight int

	prevWidth int
	startX    int
	height    int
}

// AdjustStyle changes the style of this text entity.
func (be *BaseEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	be.Style = fn(be.Style)
	return be
}

func (be *BaseEntity) IsEmpty() bool {
	return false
}

// IsBlock returns whether or not this is a block-type entity.
func (be *BaseEntity) IsBlock() bool {
	return be.Block
}

// GetTag returns the HTML tag of this entity.
func (be *BaseEntity) GetTag() string {
	return be.Tag
}

// Height returns the render height of this entity.
func (be *BaseEntity) Height() int {
	return be.height
}

func (be *BaseEntity) getStartX() int {
	return be.startX
}

// Clone creates a copy of this base entity.
func (be *BaseEntity) Clone() Entity {
	return &BaseEntity{
		Tag:           be.Tag,
		Style:         be.Style,
		Block:         be.Block,
		DefaultHeight: be.DefaultHeight,
	}
}

func (be *BaseEntity) PlainText() string {
	return ""
}

// String returns a textual representation of this BaseEntity struct.
func (be *BaseEntity) String() string {
	return fmt.Sprintf(`&html.BaseEntity{Tag="%s", Style=%#v, Block=%t, startX=%d, height=%d}`,
		be.Tag, be.Style, be.Block, be.startX, be.height)
}

// CalculateBuffer prepares this entity for rendering with the given parameters.
func (be *BaseEntity) CalculateBuffer(width, startX int, ctx DrawContext) int {
	be.height = be.DefaultHeight
	be.startX = startX
	if be.Block {
		be.startX = 0
	}
	return be.startX
}

func (be *BaseEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	panic("Called Draw() of BaseEntity")
}
