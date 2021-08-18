package ui

import (
	"maunium.net/go/tcell"

	"maunium.net/go/mauview"
)

const mainHelpText = `# General
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
/ph <word> <word>    - Send text resembling the PornHub logo (/pornhub).
/html[me] <message>  - Send html[in emote] allowing colored chats that work in Element.
           - Example: <b><font color="#FFFFFF" data-mx-bg-color="#000000">black
/reply [text]        - Reply to the selected message.
/react <reaction>    - React to the selected message.
/redact [reason]     - Redact the selected message.
/edit                - Edit the selected message.

# Encryption
/fingerprint - View the fingerprint of your device.
/cross-signing - Sub commands related to encryption key cross-signing.

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

const keyboardHelp = `# Shortuts
Ctrl and Alt are interchangeable in most keybindings,
but the other one may not work depending on your terminal emulator.

    Switch rooms: Ctrl + ↑, Ctrl + ↓
    Scroll chat (page): PgUp, PgDown
    Jump to room: Ctrl + K, type part of a room's name, then Tab and Enter to navigate and select room
    Plaintext mode: Ctrl + L
    Newline: Alt + Enter
    Autocompletion: Tab (emojis, usernames, room aliases and commands)

# Editing messages

↑ and ↓ can be used at the start and end of the input area to jump to edit the previous or next message respectively.
Selecting messages

After using commands that require selecting messages (e.g. /reply and /redact), you can move the selection with ↑ and ↓ confirm with Enter.

# Mouse

    Click to select message (for commands such as /reply that act on a message)
    Ctrl + click on image to open in your default image viewer (xdg-open)
    Click on a username to insert a mention of that user into the composer`

type HelpModal struct {
	mauview.FocusableComponent
	parent *MainView
}

func NewHelpModal(parent *MainView, target string) *HelpModal {
	helpText := mainHelpText
	hm := &HelpModal{parent: parent}

	switch target {
	case "kb":
		helpText = keyboardHelp
	}

	text := mauview.NewTextView().
		SetText(helpText).
		SetScrollable(true).
		SetWrap(false).
		SetTextColor(tcell.ColorDefault)

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
