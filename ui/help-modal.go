package ui

import (
	"maunium.net/go/tcell"

	"maunium.net/go/mauview"
)

const helpText = `# General
/help           - Show this help dialog.
/quit           - Quit gomuks.
/clearcache     - Clear cache and quit gomuks.
/logout         - Log out of Matrix.
/toggle <thing> - Temporary command to toggle various UI features.

# Media
/download [path] - Downloads file from selected message.
/open [path]     - Download file from selected message and open it with xdg-open.
/upload <path>   - Upload the file at the given path to the current room.

# Sending special messages
/me <message>        - Send an emote message.
/notice <message>    - Send a notice (generally used for bot messages).
/rainbow <message>   - Send rainbow text.
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
/verify <user id> - Verify a user with in-room verification. Probably broken.
/verify-device <user id> <device id> [fingerprint]
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

type HelpModal struct {
	mauview.FocusableComponent
	parent *MainView
}

func NewHelpModal(parent *MainView) *HelpModal {
	hm := &HelpModal{parent: parent}

	text := mauview.NewTextView().
		SetText(helpText).
		SetScrollable(true).
		SetWrap(false)

	box := mauview.NewBox(text).
		SetBorder(true).
		SetTitle("Help").
		SetBlurCaptureFunc(func() bool {
			hm.parent.HideModal()
			return true
		})
	box.Focus()

	hm.FocusableComponent = mauview.FractionalCenter(box, 42, 10, 0.5, 0.5)

	return hm
}

func (hm *HelpModal) OnKeyEvent(event mauview.KeyEvent) bool {
	if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
		hm.parent.HideModal()
		return true
	}
	return hm.FocusableComponent.OnKeyEvent(event)
}
