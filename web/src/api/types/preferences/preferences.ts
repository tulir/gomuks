// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
import { Preference, anyContext } from "./types.ts"

export const codeBlockStyles = [
	"auto", "abap", "algol_nu", "algol", "arduino", "autumn", "average", "base16-snazzy", "borland", "bw",
	"catppuccin-frappe", "catppuccin-latte", "catppuccin-macchiato", "catppuccin-mocha", "colorful", "doom-one2",
	"doom-one", "dracula", "emacs", "friendly", "fruity", "github-dark", "github", "gruvbox-light", "gruvbox",
	"hrdark", "hr_high_contrast", "igor", "lovelace", "manni", "modus-operandi", "modus-vivendi", "monokailight",
	"monokai", "murphy", "native", "nord", "onedark", "onesenterprise", "paraiso-dark", "paraiso-light", "pastie",
	"perldoc", "pygments", "rainbow_dash", "rose-pine-dawn", "rose-pine-moon", "rose-pine", "rrt", "solarized-dark256",
	"solarized-dark", "solarized-light", "swapoff", "tango", "tokyonight-day", "tokyonight-moon", "tokyonight-night",
	"tokyonight-storm", "trac", "vim", "vs", "vulcan", "witchhazel", "xcode-dark", "xcode",
] as const

export type CodeBlockStyle = typeof codeBlockStyles[number]

/* eslint-disable max-len */
export const preferences = {
	send_read_receipts: new Preference<boolean>({
		displayName: "Send read receipts",
		description: "Should read receipts be sent to other users? If disabled, read receipts will use the `m.read.private` type, which only syncs to your own devices.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	send_typing_notifications: new Preference<boolean>({
		displayName: "Send typing notifications",
		description: "Should typing notifications be sent to other users?",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	code_block_line_wrap: new Preference<boolean>({
		displayName: "Code block line wrap",
		description: "Whether to wrap long lines in code blocks instead of scrolling horizontally.",
		allowedContexts: anyContext,
		defaultValue: false,
	}),
	code_block_theme: new Preference<CodeBlockStyle>({
		displayName: "Code block theme",
		description: "The syntax highlighting theme to use for code blocks.",
		allowedContexts: anyContext,
		defaultValue: "auto",
		allowedValues: codeBlockStyles,
	}),
	pointer_cursor: new Preference<boolean>({
		displayName: "Pointer cursor",
		description: "Whether to use a pointer cursor for clickable elements.",
		allowedContexts: anyContext,
		defaultValue: false,
	}),
	custom_css: new Preference<string>({
		displayName: "Custom CSS",
		description: "Arbitrary custom CSS to apply to the client.",
		allowedContexts: anyContext,
		defaultValue: "",
	}),
	show_hidden_events: new Preference<boolean>({
		displayName: "Show hidden events",
		description: "Whether hidden events should be visible in the room timeline.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	show_room_emoji_packs: new Preference<boolean>({
		displayName: "Show room emoji packs",
		description: "Whether to show custom emoji packs provided by the room. If disabled, only your personal packs are shown in all rooms.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
} as const

export const existingPreferenceKeys = new Set(Object.keys(preferences))

export type Preferences = {
	-readonly [name in keyof typeof preferences]?: typeof preferences[name]["defaultValue"]
}
