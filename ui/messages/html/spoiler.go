// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2022 Tulir Asokan
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
	"go.mau.fi/tcell"
)

type SpoilerEntity struct {
	reason  string
	hidden  *ContainerEntity
	visible *ContainerEntity
}

const SpoilerColor = tcell.ColorYellow

func NewSpoilerEntity(visible *ContainerEntity, reason string) *SpoilerEntity {
	hidden := visible.Clone().(*ContainerEntity)
	hidden.AdjustStyle(func(style tcell.Style) tcell.Style {
		return style.Foreground(SpoilerColor).Background(SpoilerColor)
	}, AdjustStyleReasonHideSpoiler)
	if len(reason) > 0 {
		reasonEnt := NewTextEntity(fmt.Sprintf("(%s)", reason))
		hidden.Children = append([]Entity{reasonEnt}, hidden.Children...)
		visible.Children = append([]Entity{reasonEnt}, visible.Children...)
	}
	return &SpoilerEntity{
		reason:  reason,
		hidden:  hidden,
		visible: visible,
	}
}

func (se *SpoilerEntity) Clone() Entity {
	return &SpoilerEntity{
		reason:  se.reason,
		hidden:  se.hidden.Clone().(*ContainerEntity),
		visible: se.visible.Clone().(*ContainerEntity),
	}
}

func (se *SpoilerEntity) IsBlock() bool {
	return false
}

func (se *SpoilerEntity) GetTag() string {
	return "span"
}

func (se *SpoilerEntity) Draw(screen mauview.Screen, ctx DrawContext) {
	if ctx.IsSelected {
		se.visible.Draw(screen, ctx)
	} else {
		se.hidden.Draw(screen, ctx)
	}
}

func (se *SpoilerEntity) AdjustStyle(fn AdjustStyleFunc, reason AdjustStyleReason) Entity {
	if reason != AdjustStyleReasonHideSpoiler {
		se.hidden.AdjustStyle(func(style tcell.Style) tcell.Style {
			return fn(style).Foreground(SpoilerColor).Background(SpoilerColor)
		}, reason)
		se.visible.AdjustStyle(fn, reason)
	}
	return se
}

func (se *SpoilerEntity) PlainText() string {
	if len(se.reason) > 0 {
		return fmt.Sprintf("spoiler: %s", se.reason)
	} else {
		return "spoiler"
	}
}

func (se *SpoilerEntity) String() string {
	var buf strings.Builder
	_, _ = fmt.Fprintf(&buf, `&html.SpoilerEntity{reason=%s`, se.reason)
	buf.WriteString("\n    visible=")
	buf.WriteString(strings.Join(strings.Split(strings.TrimRight(se.visible.String(), "\n"), "\n"), "\n    "))
	buf.WriteString("\n    hidden=")
	buf.WriteString(strings.Join(strings.Split(strings.TrimRight(se.hidden.String(), "\n"), "\n"), "\n    "))
	buf.WriteString("\n]},")
	return buf.String()
}

func (se *SpoilerEntity) Height() int {
	return se.visible.Height()
}

func (se *SpoilerEntity) CalculateBuffer(width, startX int, ctx DrawContext) int {
	se.hidden.CalculateBuffer(width, startX, ctx)
	return se.visible.CalculateBuffer(width, startX, ctx)
}

func (se *SpoilerEntity) getStartX() int {
	return se.visible.getStartX()
}

func (se *SpoilerEntity) IsEmpty() bool {
	return se.visible.IsEmpty()
}
