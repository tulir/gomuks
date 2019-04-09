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

	"maunium.net/go/mauview"
)

type BlockquoteEntity struct {
	*BaseEntity
}

const BlockQuoteChar = '>'

func NewBlockquoteEntity(children []Entity) *BlockquoteEntity {
	return &BlockquoteEntity{&BaseEntity{
		Tag:      "blockquote",
		Children: children,
		Block:    true,
		Indent:   2,
	}}
}

func (be *BlockquoteEntity) Draw(screen mauview.Screen) {
	be.BaseEntity.Draw(screen)
	for y := 0; y < be.height; y++ {
		screen.SetContent(0, y, BlockQuoteChar, nil, be.Style)
	}
}

func (be *BlockquoteEntity) String() string {
	return fmt.Sprintf("&html.BlockquoteEntity{%s},\n", be.BaseEntity)
}
