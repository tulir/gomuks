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
import type { ContentURI } from "../../types"
import { Preference, anyContext, anyGlobalContext } from "./types.ts"

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
export const mapProviders = ["leaflet", "google", "none"] as const
export const gifProviders = ["giphy", "tenor"] as const

export type CodeBlockStyle = typeof codeBlockStyles[number]
export type MapProvider = typeof mapProviders[number]
export type GIFProvider = typeof gifProviders[number]

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
	send_bundled_url_previews: new Preference<boolean>({
		displayName: "Send bundled URL previews",
		description: "Should the composer offer fetching URL previews to bundle in outgoing messages?",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	display_read_receipts: new Preference<boolean>({
		displayName: "Display read receipts",
		description: "Should read receipts be rendered in the timeline?",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	show_media_previews: new Preference<boolean>({
		displayName: "Show image and video previews",
		description: "If disabled, images and videos will only be visible after clicking and will not be downloaded automatically. This will also disable images in URL previews.",
		allowedContexts: anyContext,
		defaultValue: false,
	}),
	show_inline_images: new Preference<boolean>({
		displayName: "Show inline images",
		description: "If disabled, custom emojis and other inline images will not be rendered and the alt attribute will be shown instead.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	show_invite_avatars: new Preference<boolean>({
		displayName: "Show avatars in invites",
		description: "If disabled, the avatar of the room or inviter will not be shown in the invite view.",
		allowedContexts: anyGlobalContext,
		defaultValue: false,
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
		displayName: "Use pointer cursor",
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
	show_redacted_events: new Preference<boolean>({
		displayName: "Show redacted event placeholders",
		description: "Whether redacted events should leave a placeholder behind in the room timeline.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	show_membership_events: new Preference<boolean>({
		displayName: "Show membership events",
		description: "Whether membership and profile changes should be visible in the room timeline.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	render_url_previews: new Preference<boolean>({
		displayName: "Render URL previews",
		description: "Whether to render MSC4095 URL previews in the room timeline.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	small_replies: new Preference<boolean>({
		displayName: "Compact reply style",
		description: "Whether to use a Discord-like compact style for replies instead of the traditional style.",
		allowedContexts: anyContext,
		defaultValue: false,
	}),
	show_date_separators: new Preference<boolean>({
		displayName: "Show date separators",
		description: "Whether messages in different days should have a date separator between them in the room timeline.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	show_room_emoji_packs: new Preference<boolean>({
		displayName: "Show room emoji packs",
		description: "Whether to show custom emoji packs provided by the room. If disabled, only your personal packs are shown in all rooms.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	upload_dialog: new Preference<boolean>({
		displayName: "Show upload dialog",
		description: "Whether to show the dialog that allows adjusting the media before upload (re-encoding, resizing, etc)",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	map_provider: new Preference<MapProvider>({
		displayName: "Map provider",
		description: "The map provider to use for location messages.",
		allowedValues: mapProviders,
		allowedContexts: anyContext,
		defaultValue: "leaflet",
	}),
	leaflet_tile_template: new Preference<string>({
		displayName: "Leaflet tile URL template",
		description: "When using Leaflet for maps, the URL template for map tile images.",
		allowedContexts: anyContext,
		defaultValue: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
	}),
	element_call_base_url: new Preference<string>({
		displayName: "Element call base URL",
		description: "The widget base URL for Element calls.",
		allowedContexts: anyContext,
		defaultValue: "https://call.element.io",
	}),
	gif_provider: new Preference<GIFProvider>({
		displayName: "GIF provider",
		description: "The service to use to search for GIFs",
		allowedValues: gifProviders,
		allowedContexts: anyContext,
		defaultValue: "giphy",
	}),
	// TODO implement
	// reupload_gifs: new Preference<boolean>({
	// 	displayName: "Reupload GIFs",
	// 	description: "Should GIFs be reuploaded to your server's media repo instead of using the proxy?",
	// 	allowedContexts: anyContext,
	// 	defaultValue: false,
	// }),
	message_context_menu: new Preference<boolean>({
		displayName: "Right-click menu on messages",
		description: "Show a context menu when right-clicking on messages.",
		allowedContexts: anyContext,
		defaultValue: true,
	}),
	ctrl_enter_send: new Preference<boolean>({
		displayName: "Use Ctrl+Enter to send",
		description: "Disable sending on enter and use Ctrl+Enter for sending instead",
		allowedContexts: anyContext,
		defaultValue: false,
	}),
	custom_notification_sound: new Preference<ContentURI>({
		displayName: "Custom notification sound",
		description: "The mxc:// URI to a custom notification sound.",
		allowedContexts: anyContext,
		defaultValue: "",
	}),
	room_window_title: new Preference<string>({
		displayName: "In-room window title",
		description: "The title to use for the window when viewing a room. $room will be replaced with the room name",
		allowedContexts: anyContext,
		defaultValue: "$room - gomuks web",
	}),
	window_title: new Preference<string>({
		displayName: "Default window title",
		description: "The title to use for the window when not in a room.",
		allowedContexts: anyGlobalContext,
		defaultValue: "gomuks web",
	}),
	favicon: new Preference<string>({
		displayName: "Favicon",
		description: "The URL to use for the favicon.",
		allowedContexts: anyContext,
		defaultValue: "gomuks.png",
	}),
} as const

export const existingPreferenceKeys = new Set(Object.keys(preferences))

export type Preferences = {
	-readonly [name in keyof typeof preferences]?: typeof preferences[name]["defaultValue"]
}

export function isValidPreferenceKey(key: unknown): key is keyof Preferences {
	return typeof key === "string" && existingPreferenceKeys.has(key)
}
