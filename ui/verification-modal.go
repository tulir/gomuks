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

// +build cgo

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
	"maunium.net/go/mautrix/event"
)

type EmojiView struct {
	mauview.SimpleEventHandler
	Data crypto.SASData
}

func (e *EmojiView) Draw(screen mauview.Screen) {
	if e.Data == nil {
		return
	}
	switch e.Data.Type() {
	case event.SASEmoji:
		width := 10
		for i, emoji := range e.Data.(crypto.EmojiSASData) {
			x := i*width + i
			y := 0
			if i >= 4 {
				x = (i-4)*width + i
				y = 2
			}
			mauview.Print(screen, string(emoji.Emoji), x, y, width, mauview.AlignCenter, tcell.ColorDefault)
			mauview.Print(screen, emoji.Description, x, y+1, width, mauview.AlignCenter, tcell.ColorDefault)
		}
	case event.SASDecimal:
		maxWidth := 43
		for i, number := range e.Data.(crypto.DecimalSASData) {
			mauview.Print(screen, strconv.FormatUint(uint64(number), 10), 0, i, maxWidth, mauview.AlignCenter, tcell.ColorDefault)
		}
	}
}

type VerificationModal struct {
	mauview.Component

	device *crypto.DeviceIdentity

	container *mauview.Box

	waitingBar *mauview.ProgressBar
	infoText   *mauview.TextView
	emojiText  *EmojiView
	inputBar   *mauview.InputField

	progress    int
	progressMax int
	stopWaiting chan struct{}
	confirmChan chan bool
	done        bool

	parent *MainView
}

func NewVerificationModal(mainView *MainView, device *crypto.DeviceIdentity, timeout time.Duration) *VerificationModal {
	vm := &VerificationModal{
		parent:      mainView,
		device:      device,
		stopWaiting: make(chan struct{}),
		confirmChan: make(chan bool),
		done:        false,
	}

	vm.progressMax = int(timeout.Seconds())
	vm.progress = vm.progressMax
	vm.waitingBar = mauview.NewProgressBar().
		SetMax(vm.progressMax).
		SetProgress(vm.progress).
		SetIndeterminate(false)

	vm.infoText = mauview.NewTextView()
	vm.infoText.SetText(fmt.Sprintf("Waiting for %s\nto accept", device.UserID))

	vm.emojiText = &EmojiView{}

	vm.inputBar = mauview.NewInputField().
		SetBackgroundColor(tcell.ColorDefault).
		SetPlaceholderTextColor(tcell.ColorWhite)

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

	go vm.decrementWaitingBar()

	return vm
}

func (vm *VerificationModal) decrementWaitingBar() {
	for {
		select {
		case <-time.Tick(time.Second):
			if vm.progress <= 0 {
				vm.waitingBar.SetIndeterminate(true)
				vm.parent.parent.app.SetRedrawTicker(100 * time.Millisecond)
				return
			}
			vm.progress--
			vm.waitingBar.SetProgress(vm.progress)
			vm.parent.parent.Render()
		case <-vm.stopWaiting:
			return
		}
	}
}

func (vm *VerificationModal) VerificationMethods() []crypto.VerificationMethod {
	return []crypto.VerificationMethod{crypto.VerificationMethodEmoji{}, crypto.VerificationMethodDecimal{}}
}

func (vm *VerificationModal) VerifySASMatch(device *crypto.DeviceIdentity, data crypto.SASData) bool {
	vm.device = device
	var typeName string
	if data.Type() == event.SASDecimal {
		typeName = "numbers"
	} else if data.Type() == event.SASEmoji {
		typeName = "emojis"
	} else {
		return false
	}
	vm.infoText.SetText(fmt.Sprintf(
		"Check if the other device is showing the\n"+
			"same %s as below, then type \"yes\" to\n"+
			"accept, or \"no\" to reject", typeName))
	vm.inputBar.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetPlaceholder("Type \"yes\" or \"no\"").
		Focus()
	vm.emojiText.Data = data
	vm.parent.parent.Render()
	vm.progress = vm.progressMax
	confirm := <-vm.confirmChan
	vm.progress = vm.progressMax
	vm.emojiText.Data = nil
	vm.infoText.SetText(fmt.Sprintf("Waiting for %s\nto confirm", vm.device.UserID))
	vm.parent.parent.Render()
	return confirm
}

func (vm *VerificationModal) OnCancel(cancelledByUs bool, reason string, _ event.VerificationCancelCode) {
	vm.waitingBar.SetIndeterminate(false).SetMax(100).SetProgress(100)
	vm.parent.parent.app.SetRedrawTicker(1 * time.Minute)
	if cancelledByUs {
		vm.infoText.SetText(fmt.Sprintf("Verification failed: %s", reason))
	} else {
		vm.infoText.SetText(fmt.Sprintf("Verification cancelled by %s: %s", vm.device.UserID, reason))
	}
	vm.inputBar.SetPlaceholder("Press enter to close the dialog")
	vm.stopWaiting <- struct{}{}
	vm.done = true
	vm.parent.parent.Render()
}

func (vm *VerificationModal) OnSuccess() {
	vm.waitingBar.SetIndeterminate(false).SetMax(100).SetProgress(100)
	vm.parent.parent.app.SetRedrawTicker(1 * time.Minute)
	vm.infoText.SetText(fmt.Sprintf("Successfully verified %s (%s) of %s", vm.device.Name, vm.device.DeviceID, vm.device.UserID))
	vm.inputBar.SetPlaceholder("Press enter to close the dialog")
	vm.stopWaiting <- struct{}{}
	vm.done = true
	vm.parent.parent.Render()
	if vm.parent.config.SendToVerifiedOnly {
		// Hacky way to make new group sessions after verified
		vm.parent.matrix.Crypto().(*crypto.OlmMachine).OnDevicesChanged(vm.device.UserID)
	}
}

func (vm *VerificationModal) OnKeyEvent(event mauview.KeyEvent) bool {
	if vm.done {
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyEsc {
			vm.parent.HideModal()
			return true
		}
		return false
	} else if vm.emojiText.Data == nil {
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
		vm.inputBar.
			SetPlaceholder("").
			SetTextAndMoveCursor("").
			SetBackgroundColor(tcell.ColorDefault).
			SetTextColor(tcell.ColorDefault)
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
