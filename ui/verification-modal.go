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
	"strconv"
	"strings"
	"time"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/mautrix/crypto"
)

type EmojiView struct {
	mauview.SimpleEventHandler
	Numbers *[3]uint
	Emojis  *[7]crypto.VerificationEmoji
}

func (e *EmojiView) Draw(screen mauview.Screen) {
	if e.Emojis != nil {
		width := 10
		for i, emoji := range e.Emojis {
			x := i*width + i
			y := 0
			if i >= 4 {
				x = (i-4)*width + i
				y = 2
			}
			mauview.Print(screen, string(emoji.Emoji), x, y, width, mauview.AlignCenter, tcell.ColorDefault)
			mauview.Print(screen, emoji.Description, x, y+1, width, mauview.AlignCenter, tcell.ColorDefault)
		}
	} else if e.Numbers != nil {
		maxWidth := 43
		for i, number := range e.Numbers {
			mauview.Print(screen, strconv.FormatUint(uint64(number), 10), 0, i, maxWidth, mauview.AlignCenter, tcell.ColorDefault)
		}
	}
}

type VerificationModal struct {
	mauview.Component

	container *mauview.Box

	waitingBar *mauview.ProgressBar
	infoText   *mauview.TextView
	emojiText  *EmojiView
	inputBar   *mauview.InputField

	stopWaiting chan struct{}
	confirmChan chan bool

	parent *MainView
}

func NewVerificationModal(mainView *MainView, device *crypto.DeviceIdentity, timeout time.Duration) *VerificationModal {
	vm := &VerificationModal{
		parent:      mainView,
		stopWaiting: make(chan struct{}),
		confirmChan: make(chan bool),
	}

	progress := int(timeout.Seconds())
	vm.waitingBar = mauview.NewProgressBar().
		SetMax(progress).
		SetProgress(progress).
		SetIndeterminate(false)

	vm.infoText = mauview.NewTextView()
	vm.infoText.SetText(fmt.Sprintf("Waiting for %s to accept", device.UserID))

	vm.emojiText = &EmojiView{}

	vm.inputBar = mauview.NewInputField().SetBackgroundColor(tcell.ColorDefault)

	flex := mauview.NewFlex().
		SetDirection(mauview.FlexRow).
		AddFixedComponent(vm.waitingBar, 1).
		AddFixedComponent(vm.infoText, 4).
		AddFixedComponent(vm.emojiText, 4).
		AddFixedComponent(vm.inputBar, 1)

	vm.container = mauview.NewBox(flex).
		SetBorder(true).
		SetTitle("Interactive verification")

	vm.Component = mauview.Center(vm.container, 45, 12).SetAlwaysFocusChild(true)

	go vm.decrementWaitingBar(progress)

	return vm
}

func (vm *VerificationModal) decrementWaitingBar(progress int) {
	for {
		select {
		case <-time.Tick(time.Second):
			if progress <= 0 {
				vm.parent.HideModal()
				vm.parent.parent.Render()
				return
			}
			progress--
			vm.waitingBar.SetProgress(progress)
			vm.parent.parent.Render()
		case <-vm.stopWaiting:
			vm.waitingBar.SetIndeterminate(true)
			break
		}
	}
}

func (vm *VerificationModal) VerifyEmojisMatch(emojis [7]crypto.VerificationEmoji, _ *crypto.DeviceIdentity) bool {
	vm.infoText.SetText("Check if the other device is showing the same emojis as below, then type \"yes\" to accept, or \"no\" to reject")
	vm.inputBar.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetPlaceholder("Type \"yes\" or \"no\"").
		Focus()
	vm.emojiText.Emojis = &emojis
	vm.parent.parent.Render()
	vm.stopWaiting <- struct{}{}
	confirm := <-vm.confirmChan
	// TODO this should hook into cancel/success of the verification and display a success message instead of just closing
	vm.parent.HideModal()
	vm.parent.parent.Render()
	return confirm
}

func (vm *VerificationModal) VerifyNumbersMatch(numbers [3]uint, _ *crypto.DeviceIdentity) bool {
	vm.infoText.SetText("Check if the other device is showing the same numbers as below, then type \"yes\" to accept, or \"no\" to reject")
	vm.inputBar.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetPlaceholder("Type \"yes\" or \"no\"").
		Focus()
	vm.emojiText.Numbers = &numbers
	vm.parent.parent.Render()
	vm.stopWaiting <- struct{}{}
	confirm := <-vm.confirmChan
	// TODO this should hook into cancel/success of the verification and display a success message instead of just closing
	vm.parent.HideModal()
	vm.parent.parent.Render()
	return confirm
}

func (vm *VerificationModal) OnKeyEvent(event mauview.KeyEvent) bool {
	if vm.emojiText.Emojis == nil && vm.emojiText.Numbers == nil {
		debug.Print("Ignoring pre-emoji key event")
		return false
	}
	if event.Key() == tcell.KeyEnter {
		text := strings.ToLower(strings.TrimSpace(vm.inputBar.GetText()))
		if text == "yes" {
			debug.Print("Confirming verification")
			vm.confirmChan <- true
		} else if text == "no" {
			debug.Print("Rejecting verification")
			vm.confirmChan <- false
		}
		return true
	} else {
		return vm.inputBar.OnKeyEvent(event)
	}
}

func (vm *VerificationModal) Focus() {
	vm.container.Focus()
}

func (vm *VerificationModal) Blur() {
	vm.container.Blur()
}
