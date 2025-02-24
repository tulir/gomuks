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

type CodeBlockEntity struct {
	*ContainerEntity
	Background tcell.Style
}

func NewCodeBlockEntity(children []Entity, background tcell.Style) *CodeBlockEntity {
	return &CodeBlockEntity{
		ContainerEntity: &ContainerEntity{
			BaseEntity: &BaseEntity{
				Tag:   "pre",
				Block: true,
			},
			Children: children,
		},
		Background: background,
	}
}

func (ce *CodeBlockEntity) Clone() Entity {
	return &CodeBlockEntity{
		ContainerEntity: ce.ContainerEntity.Clone().(*ContainerEntity),
		Background:      ce.Background,
	}
}

func (ce *CodeBlockEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	screen.Fill(' ', ce.Background)
	ce.ContainerEntity.Draw(screen, ctx)
}

func (ce *CodeBlockEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	if reason != AdjustStyleReasonNormal {
		ce.ContainerEntity.AdjustStyle(fn, reason)
	}
	return ce
}
