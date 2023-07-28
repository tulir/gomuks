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
	"strings"

	"go.mau.fi/mauview"
)

type HorizontalLineEntity struct {
	*BaseEntity
}

const HorizontalLineChar = '━'

func NewHorizontalLineEntity() *HorizontalLineEntity {
	return &HorizontalLineEntity{&BaseEntity{
		Tag:           "hr",
		Block:         true,
		DefaultHeight: 1,
	}}
}

func (he *HorizontalLineEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	he.BaseEntity = he.BaseEntity.AdjustStyle(fn, reason).(*BaseEntity)
	return he
}

func (he *HorizontalLineEntity) Clone() Entity {
	return NewHorizontalLineEntity()
}

func (he *HorizontalLineEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	width, _ := screen.Size()
	for x := 0; x < width; x++ {
		screen.SetContent(x, 0, HorizontalLineChar, nil, he.Style)
	}
}

func (he *HorizontalLineEntity) PlainText() string {
	return strings.Repeat(string(HorizontalLineChar), 5)
}

func (he *HorizontalLineEntity) String() string {
	return "&html.HorizontalLineEntity{},\n"
}
