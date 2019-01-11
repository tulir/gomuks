// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package bfhtml

import (
	"io"

	bf "github.com/russross/blackfriday/v2"
)

type HTMLRenderer struct {
	*bf.HTMLRenderer
}

func (r *HTMLRenderer) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	if node.Type == bf.Paragraph {
		return bf.GoToNext
	}
	return r.HTMLRenderer.RenderNode(w, node, entering)
}
