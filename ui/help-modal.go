package ui

import (
	"maunium.net/go/tcell"

	"maunium.net/go/mauview"
)

const mainHelpText = `

# General

/help [kb]      - Show this help dialog. (/help [kb] to show keyboard shortcuts)
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

   e.g: '/html <font color="#FFF" data-mx-bg-color="#000">white on black</font>'
/ph <word> <word>    - Send text resembling the PornHub logo (/pornhub).
/html[me] <message>  - Send html[in emote] allowing colored chats that work in Element.
            + Example: <b><font color="#FFFFFF" data-mx-bg-color="#000000">black

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

/verify-device <user id> <device id> [fingerprint]
    - if no fingerprint is passed, it will attempt
      interactive verifcation, but this is mostly broken.

/reset-session - Reset the outbound Megolm session in the current room.

/import <file> - Import encryption keys
    - it's very likely yo will need to run /clearcache after importing keys.

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

const keyboardHelp = `NOTE: Ctrl and Alt are interchangeable in most keybindings,
      this may change based on your terminal emulator.


## Navigation

 # By Movement

Ctrl +    ↑|↓       - Navigate through rooms.

Ctrl + PgUp|PgDown  - Scroll through current room.

##  By Search:

1) Ctrl + [K]      - Open a fuzzy finder to locate rooms by typing.
        |  ≍
        ╰▹[F]

2) [TAB]⮆ [Enter]  - Use tab to navigate selections, enter to select.

## Editing messages

Your arrow keys (↑ and ↓) may be used for editing previous messages.

 A) While your cursor is positioned the beginning of a new composer;
  ╰▶ press [↑] arrow to edit your previous message.

 B) While your cursor is positioned at the end of an editing composer;
  ╰▶ press [↓] arrow to edit a new message.

## Selecting messages

After using commands that require selecting messages (e.g. /reply and /redact);
navigate the selections with ↑ and ↓ and press Enter to confirm.

(++) TAB will autocomplete man different items.
     (e.g: emojis, usernames, room aliases, commands, files)

## Mouse

 * Selecting messages can be done by mouse click (for commands such as /reply)

 * Using Ctrl + click on an image will open it in your default image viewer (xdg-open)

 * Clicking on a username will insert a mention of that user into the composer


## General

Ctrl + L           - Switch to plaintext view for easy copy+paste.
Alt  + Enter       - Start a newline in your current message.`

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
		SetTitle("(F1)Help - (k)eyboard shortcuts - (q)uit").
		SetBlurCaptureFunc(func() bool {
			hm.parent.HideModal()
			return true
		})
	box.Focus()

	hm.FocusableComponent = mauview.FractionalCenter(box, 42, 10, 0.5, 0.5)

	return hm
}

func (hm *HelpModal) OnKeyEvent(event mauview.KeyEvent) bool {
	k := event.Key()
	c := event.Rune()

	switch {
	case k == tcell.KeyEscape:
		fallthrough
	case c == 'q':
		hm.parent.HideModal()
		return true
	case c == 'k':
		hm.parent.HideModal()
		hm.parent.ShowModal(NewHelpModal(hm.parent, "kb"))
		return true
	case k == tcell.KeyF1:
		hm.parent.HideModal()
		hm.parent.ShowModal(NewHelpModal(hm.parent, "main"))
		return true
	}
	return hm.FocusableComponent.OnKeyEvent(event)
}
