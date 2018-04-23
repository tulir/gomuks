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

package widget

import (
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type FormTextView struct {
	*tview.TextView
}

func (ftv *FormTextView) GetLabel() string {
	return ""
}

func (ftv *FormTextView) SetFormAttributes(label string, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	return ftv
}

func (ftv *FormTextView) GetFieldWidth() int {
	_, _, w, _ := ftv.TextView.GetRect()
	return w
}

func (ftv *FormTextView) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	ftv.SetDoneFunc(handler)
	return ftv
}
