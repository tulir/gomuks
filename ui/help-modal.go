package ui

import (
	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/config"
)

const helpText = `# General
/help           - Show this help dialog.
/keys           - Show keyboard shortcuts.
/quit           - Quit gomuks.
/clearcache     - Clear cache and quit gomuks.
/logout         - Log out of Matrix.
/toggle <thing> - Temporary command to toggle various UI features.
                  Run /toggle without arguments to see the list of toggles.

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

/cross-signing <subcommand> [...]
    - Cross-signing commands. Somewhat experimental.
      Run without arguments for help. (alias: /cs)
/ssss <subcommand> [...]
    - Secure Secret Storage (and Sharing) commands. Very experimental.
      Run without arguments for help.

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

const keysText = `Note: Ctrl and Alt are interchangeable in most keybindings, but the other one may not work depending on your terminal emulator.

SHORTCUTS:
Ctrl+↑, Ctrl+↓ - Switch rooms
PgUp, PgDown   - Scroll chat (page)
Ctrl+K         - Jump to room, type part of a room's name, then TAB and ENTER to navigate and select room
Ctrl+L         - Plaintext mode
Alt+Enter      - Newline
Tab            - Autocompletion (emojis, usernames, room aliases and commands)

EDITING MESSAGES:
↑ and ↓ can be used at the start and end of the input area to jump to edit the previous or next message respectively.

SELECTING MESSAGES:
After using commands that require selecting messages (e.g. /reply and /redact), you can move the selection with ↑ and ↓ confirm with ENTER

MOUSE:
Click               - select message (for commands such as /reply that act on a message)
CTRL+CLICK on image - open in your default image viewer (xdg-open)
Click on a username - insert a mention of that user into the composer.`

func NewKeysModal(parent *MainView) *HelpModal {
	hm := &HelpModal{parent: parent}

	text := mauview.NewTextView().
		SetText(keysText).
		SetScrollable(true).
		SetWrap(false).
		SetTextColor(tcell.ColorDefault)

	box := mauview.NewBox(text).
		SetBorder(true).
		SetTitle("Keyboard shortcuts").
		SetBlurCaptureFunc(func() bool {
			hm.parent.HideModal()
			return true
		})
	box.Focus()

	hm.FocusableComponent = mauview.FractionalCenter(box, 42, 10, 0.5, 0.5)

	return hm
}

func (hm *HelpModal) OnKeyEvent(event mauview.KeyEvent) bool {
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}
	// TODO unhardcode q
	if hm.parent.config.Keybindings.Modal[kb] == "cancel" || event.Rune() == 'q' {
		hm.parent.HideModal()
		return true
	}
	return hm.FocusableComponent.OnKeyEvent(event)
}
