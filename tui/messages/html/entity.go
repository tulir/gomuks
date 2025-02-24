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
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

// AdjustStyleFunc is a lambda function type to edit an existing tcell Style.
type AdjustStyleFunc func(tcell.Style) tcell.Style

type AdjustStyleReason int

const (
	AdjustStyleReasonNormal AdjustStyleReason = iota
	AdjustStyleReasonHideSpoiler
)

type DrawContext struct {
	IsSelected   bool
	BareMessages bool
}

type Entity interface {
	// AdjustStyle recursively changes the style of the entity and all its children.
	AdjustStyle(AdjustStyleFunc, AdjustStyleReason) Entity
	// Draw draws the entity onto the given mauview Screen.
	Draw(screen mauview.Screen, ctx DrawContext)
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
	CalculateBuffer(width, startX int, ctx DrawContext) int

	getStartX() int

	IsEmpty() bool
}
