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

	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix/format"
)

type ListEntity struct {
	*ContainerEntity
	Ordered bool
	Start   int
}

func NewListEntity(ordered bool, start int, children []Entity) *ListEntity {
	entity := &ListEntity{
		ContainerEntity: &ContainerEntity{
			BaseEntity: &BaseEntity{
				Tag:   "ul",
				Block: true,
			},
			Indent:   2,
			Children: children,
		},
		Ordered: ordered,
		Start:   start,
	}
	if ordered {
		entity.Tag = "ol"
		entity.Indent += format.Digits(start + len(children) - 1)
	}
	return entity
}

func (le *ListEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	le.BaseEntity = le.BaseEntity.AdjustStyle(fn, reason).(*BaseEntity)
	le.ContainerEntity.AdjustStyle(fn, reason)
	return le
}

func (le *ListEntity) Clone() Entity {
	return &ListEntity{
		ContainerEntity: le.ContainerEntity.Clone().(*ContainerEntity),
		Ordered:         le.Ordered,
		Start:           le.Start,
	}
}

func (le *ListEntity) paddingFor(number int) string {
	padding := le.Indent - 2 - format.Digits(number)
	if padding <= 0 {
		return ""
	}
	return strings.Repeat(" ", padding)
}

func (le *ListEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	width, _ := screen.Size()

	proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: le.Indent, Width: width - le.Indent, Style: le.Style}
	for i, entity := range le.Children {
		proxyScreen.Height = entity.Height()
		if le.Ordered {
			number := le.Start + i
			line := fmt.Sprintf("%d. %s", number, le.paddingFor(number))
			widget.WriteLine(screen, mauview.AlignLeft, line, 0, proxyScreen.OffsetY, le.Indent, le.Style)
		} else {
			screen.SetContent(0, proxyScreen.OffsetY, '●', nil, le.Style)
		}
		entity.Draw(proxyScreen, ctx)
		proxyScreen.SetStyle(le.Style)
		proxyScreen.OffsetY += entity.Height()
	}
}

func (le *ListEntity) PlainText() string {
	if len(le.Children) == 0 {
		return ""
	}
	var buf strings.Builder
	for i, child := range le.Children {
		indent := strings.Repeat(" ", le.Indent)
		if le.Ordered {
			number := le.Start + i
			_, _ = fmt.Fprintf(&buf, "%d. %s", number, le.paddingFor(number))
		} else {
			buf.WriteString("● ")
		}
		for j, row := range strings.Split(child.PlainText(), "\n") {
			if j != 0 {
				buf.WriteRune('\n')
				buf.WriteString(indent)
			}
			buf.WriteString(row)
		}
		buf.WriteRune('\n')
	}
	return strings.TrimSpace(buf.String())
}

func (le *ListEntity) String() string {
	return fmt.Sprintf("&html.ListEntity{Ordered=%t, Start=%d, Base=%s},\n", le.Ordered, le.Start, le.BaseEntity)
}
