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
	"strings"

	"maunium.net/go/mauview"
)

type BlockquoteEntity struct {
	*ContainerEntity
}

const BlockQuoteChar = '>'

func NewBlockquoteEntity(children []Entity) *BlockquoteEntity {
	return &BlockquoteEntity{&ContainerEntity{
		BaseEntity: &BaseEntity{
			Tag:   "blockquote",
			Block: true,
		},
		Children: children,
		Indent:   2,
	}}
}

func (be *BlockquoteEntity) AdjustStyle(fn AdjustStyleFunc) Entity {
	be.BaseEntity = be.BaseEntity.AdjustStyle(fn).(*BaseEntity)
	return be
}

func (be *BlockquoteEntity) Clone() Entity {
	return &BlockquoteEntity{ContainerEntity: be.ContainerEntity.Clone().(*ContainerEntity)}
}

func (be *BlockquoteEntity) Draw(screen mauview.Screen) {
	be.ContainerEntity.Draw(screen)
	for y := 0; y < be.height; y++ {
		screen.SetContent(0, y, BlockQuoteChar, nil, be.Style)
	}
}

func (be *BlockquoteEntity) PlainText() string {
	if len(be.Children) == 0 {
		return ""
	}
	var buf strings.Builder
	newlined := false
	for i, child := range be.Children {
		if i != 0 && child.IsBlock() && !newlined {
			buf.WriteRune('\n')
		}
		newlined = false
		for i, row := range strings.Split(child.PlainText(), "\n") {
			if i != 0 {
				buf.WriteRune('\n')
			}
			buf.WriteRune('>')
			buf.WriteRune(' ')
			buf.WriteString(row)
		}
		if child.IsBlock() {
			buf.WriteRune('\n')
			newlined = true
		}
	}
	return strings.TrimSpace(buf.String())
}

func (be *BlockquoteEntity) String() string {
	return fmt.Sprintf("&html.BlockquoteEntity{%s},\n", be.BaseEntity)
}
