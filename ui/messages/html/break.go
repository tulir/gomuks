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
	"go.mau.fi/mauview"
)

type BreakEntity struct {
	*BaseEntity
}

func NewBreakEntity() *BreakEntity {
	return &BreakEntity{&BaseEntity{
		Tag:   "br",
		Block: true,
	}}
}

// AdjustStyle changes the style of this text entity.
func (be *BreakEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	be.BaseEntity = be.BaseEntity.AdjustStyle(fn, reason).(*BaseEntity)
	return be
}

func (be *BreakEntity) Clone() Entity {
	return NewBreakEntity()
}

func (be *BreakEntity) PlainText() string {
	return "\n"
}

func (be *BreakEntity) String() string {
	return "&html.BreakEntity{},\n"
}

func (be *BreakEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	// No-op, the logic happens in containers
}
