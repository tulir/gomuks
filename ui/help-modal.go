package ui

import (
	"strings"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"
)

type HelpModal struct {
	mauview.Component

	container *mauview.Box

	text       *mauview.TextView
	scrollX    int
	scrollY    int
	maxScrollX int
	maxScrollY int
	textWidth  int
	textHeight int

	parent *MainView
}

// There are only math.Min/math.Max for float64
func Max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func Min(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func NewHelpModal(parent *MainView) *HelpModal {
	hm := &HelpModal{
		parent: parent,

		scrollX:    0,
		scrollY:    0,
		maxScrollX: 0,
		maxScrollY: 0,
	}

	helpText := `# General
/help           - Show this "temporary" help message.
/quit           - Quit gomuks.
/clearcache     - Clear cache and quit gomuks.
/logout         - Log out of Matrix.
/toggle <thing> - Temporary command to toggle various UI features.

Things: rooms, users, baremessages, images, typingnotif, unverified

# Sending special messages
/me <message>        - Send an emote message.
/notice <message>    - Send a notice (generally used for bot messages).
/rainbow <message>   - Send rainbow text (markdown not supported).
/rainbowme <message> - Send rainbow text in an emote.
/reply [text]        - Reply to the selected message.
/react <reaction>    - React to the selected message.
/redact [reason]     - Redact the selected message.
/edit                - Edit the selected message.

# Encryption
/fingerprint - View the fingerprint of your device.

/devices <user id>               - View the device list of a user.
/device <user id> <device id>    - Show info about a specific device.
/unverify <user id> <device id>  - Un-verify a device.
/blacklist <user id> <device id> - Blacklist a device.
/verify <user id> <device id> [fingerprint]
    - Verify a device. If the fingerprint is not provided,
      interactive emoji verification will be started.
/reset-session - Reset the outbound Megolm session in the current room.

/import <file> - Import encryption keys
/export <file> - Export encryption keys
/export-room <file> - Export encryption keys for the current room.

# Rooms
/pm <user id> <...>   - Create a private chat with the given user(s).
/create [room name]   - Create a room.

/join <room> [server] - Join a room.
/accept               - Accept the invite.
/reject               - Reject the invite.

/invite <user id>     - Invite the given user to the room.
/roomnick <name>      - Change your per-room displayname.
/tag <tag> <priority> - Add the room to <tag>.
/untag <tag>          - Remove the room from <tag>.
/tags                 - List the tags the room is in.
/alias <act> <name>   - Add or remove local addresses.

/leave                     - Leave the current room.
/kick   <user id> [reason] - Kick a user.
/ban    <user id> [reason] - Ban a user.
/unban  <user id>          - Unban a user.`

	split := strings.Split(helpText, "\n")
	hm.textHeight = len(split)
	hm.textWidth = 0

	for _, line := range split {
		hm.textWidth = Max(hm.textWidth, len(line))
	}

	hm.text = mauview.NewTextView().
		SetText(helpText).
		SetScrollable(true).
		SetWrap(false)

	flex := mauview.NewFlex().
		SetDirection(mauview.FlexRow).
		AddProportionalComponent(hm.text, 1)

	hm.container = mauview.NewBox(flex).
		SetBorder(true).
		SetTitle("Help").
		SetBlurCaptureFunc(func() bool {
			hm.parent.HideModal()
			return true
		})

	hm.Component = mauview.Center(hm.container, 0, 0).
		SetAlwaysFocusChild(true)

	return hm
}

func (hm *HelpModal) Focus() {
	hm.container.Focus()
}

func (hm *HelpModal) Blur() {
	hm.container.Blur()
}

func (hm *HelpModal) Draw(screen mauview.Screen) {
	width, height := screen.Size()

	width /= 2
	if width < 42 {
		width = 42
	}

	if height > 40 {
		height -= 20
	} else if height > 30 {
		height -= 10
	} else if height > 20 {
		height -= 5
	}

	oldMaxScrollY := hm.maxScrollY
	hm.maxScrollY = hm.textHeight - height + 2
	hm.maxScrollX = hm.textWidth - width + 2

	if hm.maxScrollY != oldMaxScrollY {
		// Reset the scroll
		// NOTE: Workarounds a problem where we can no longer scroll
		//       due to hm.scrollY being too big.
		hm.scrollY = 0
		hm.scrollX = 0
		hm.text.ScrollTo(hm.scrollY, hm.scrollX)
	}

	hm.Component = mauview.Center(hm.container, width, height).
		SetAlwaysFocusChild(true)

	hm.Component.Draw(screen)
}

func (hm *HelpModal) OnKeyEvent(event mauview.KeyEvent) bool {
	switch event.Key() {
	case tcell.KeyUp:
		hm.scrollY = Max(0, hm.scrollY-1)
		hm.text.ScrollTo(hm.scrollY, hm.scrollX)
		return true
	case tcell.KeyDown:
		hm.scrollY = Min(hm.maxScrollY, hm.scrollY+1)
		hm.text.ScrollTo(hm.scrollY, hm.scrollX)
		return true

	case tcell.KeyLeft:
		hm.scrollX = Max(0, hm.scrollX-1)
		hm.text.ScrollTo(hm.scrollY, hm.scrollX)
		return true
	case tcell.KeyRight:
		hm.scrollX = Min(hm.maxScrollX, hm.scrollX+1)
		hm.text.ScrollTo(hm.scrollY, hm.scrollX)
		return true
	}

	hm.parent.HideModal()
	return true
}
