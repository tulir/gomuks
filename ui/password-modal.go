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

package ui

import (
	"fmt"
	"strings"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"
)

type PasswordModal struct {
	mauview.Component

	outputChan chan string
	cancelChan chan struct{}

	form *mauview.Form

	text        *mauview.TextField
	confirmText *mauview.TextField

	input        *mauview.InputField
	confirmInput *mauview.InputField

	cancel *mauview.Button
	submit *mauview.Button

	parent *MainView
}

func (view *MainView) AskPassword(title, thing, placeholder string, isNew bool) (string, bool) {
	pwm := NewPasswordModal(view, title, thing, placeholder, isNew)
	view.ShowModal(pwm)
	view.parent.Render()
	return pwm.Wait()
}

func NewPasswordModal(parent *MainView, title, thing, placeholder string, isNew bool) *PasswordModal {
	if placeholder == "" {
		placeholder = "correct horse battery staple"
	}
	if thing == "" {
		thing = strings.ToLower(title)
	}
	pwm := &PasswordModal{
		parent:     parent,
		form:       mauview.NewForm(),
		outputChan: make(chan string, 1),
		cancelChan: make(chan struct{}, 1),
	}

	pwm.form.
		SetColumns([]int{1, 20, 1, 20, 1}).
		SetRows([]int{1, 1, 1, 0, 0, 0, 1, 1, 1})

	width := 45
	height := 8

	pwm.text = mauview.NewTextField()
	if isNew {
		pwm.text.SetText(fmt.Sprintf("Create a %s", thing))
	} else {
		pwm.text.SetText(fmt.Sprintf("Enter the %s", thing))
	}
	pwm.input = mauview.NewInputField().
		SetMaskCharacter('*').
		SetPlaceholder(placeholder)
	pwm.form.AddComponent(pwm.text, 1, 1, 3, 1)
	pwm.form.AddFormItem(pwm.input, 1, 2, 3, 1)

	if isNew {
		height += 3
		pwm.confirmInput = mauview.NewInputField().
			SetMaskCharacter('*').
			SetPlaceholder(placeholder).
			SetChangedFunc(pwm.HandleChange)
		pwm.input.SetChangedFunc(pwm.HandleChange)
		pwm.confirmText = mauview.NewTextField().SetText(fmt.Sprintf("Confirm %s", thing))

		pwm.form.SetRow(3, 1).SetRow(4, 1).SetRow(5, 1)
		pwm.form.AddComponent(pwm.confirmText, 1, 4, 3, 1)
		pwm.form.AddFormItem(pwm.confirmInput, 1, 5, 3, 1)
	}

	pwm.cancel = mauview.NewButton("Cancel").SetOnClick(pwm.ClickCancel)
	pwm.submit = mauview.NewButton("Submit").SetOnClick(pwm.ClickSubmit)

	pwm.form.AddFormItem(pwm.submit, 3, 7, 1, 1)
	pwm.form.AddFormItem(pwm.cancel, 1, 7, 1, 1)

	box := mauview.NewBox(pwm.form).SetTitle(title)
	center := mauview.Center(box, width, height).SetAlwaysFocusChild(true)
	center.Focus()
	pwm.form.FocusNextItem()
	pwm.Component = center

	return pwm
}

func (pwm *PasswordModal) HandleChange(_ string) {
	if pwm.input.GetText() == pwm.confirmInput.GetText() {
		pwm.submit.SetBackgroundColor(mauview.Styles.ContrastBackgroundColor)
	} else {
		pwm.submit.SetBackgroundColor(tcell.ColorDefault)
	}
}

func (pwm *PasswordModal) ClickCancel() {
	pwm.parent.HideModal()
	pwm.cancelChan <- struct{}{}
}

func (pwm *PasswordModal) ClickSubmit() {
	if pwm.confirmInput == nil || pwm.input.GetText() == pwm.confirmInput.GetText() {
		pwm.parent.HideModal()
		pwm.outputChan <- pwm.input.GetText()
	}
}

func (pwm *PasswordModal) Wait() (string, bool) {
	select {
	case result := <-pwm.outputChan:
		return result, true
	case <-pwm.cancelChan:
		return "", false
	}
}
