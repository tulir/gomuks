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
import React, { useEffect, useInsertionEffect } from "react"
import type Client from "@/api/client.ts"
import { RoomStateStore, usePreferences } from "@/api/statestore"

interface StylePreferencesProps {
	client: Client
	activeRoom: RoomStateStore | null
}

function newStyleSheet(sheet: string): CSSStyleSheet {
	const style = new CSSStyleSheet()
	style.replaceSync(sheet)
	return style
}

function css(strings: TemplateStringsArray, ...values: unknown[]) {
	return newStyleSheet(String.raw(strings, ...values))
}

function pushSheet(sheet: CSSStyleSheet): () => void {
	document.adoptedStyleSheets.push(sheet)
	return () => {
		const idx = document.adoptedStyleSheets.indexOf(sheet)
		if (idx !== -1) {
			document.adoptedStyleSheets.splice(idx, 1)
		}
	}
}

function useStyle(callback: () => CSSStyleSheet | false | undefined | null | "", dependencies: unknown[]) {
	useInsertionEffect(() => {
		const sheet = callback()
		if (!sheet) {
			return
		}
		return pushSheet(sheet)
	}, dependencies)
}

function useAsyncStyle(callback: () => string | false | undefined | null, dependencies: unknown[], id?: string) {
	useInsertionEffect(() => {
		const sheet = callback()
		if (!sheet) {
			return
		}
		if (!sheet.includes("@import")) {
			return pushSheet(newStyleSheet(sheet))
		}
		const styleTags = document.createElement("style")
		if (id) {
			styleTags.id = id
		}
		styleTags.textContent = sheet
		document.head.appendChild(styleTags)
		return () => {
			document.head.removeChild(styleTags)
		}
	}, dependencies)
}

const StylePreferences = ({ client, activeRoom }: StylePreferencesProps) => {
	usePreferences(client.store, activeRoom)
	const preferences = activeRoom?.preferences ?? client.store.preferences
	useStyle(() => css`
		div.html-body a.hicli-matrix-uri-user[href="matrix:u/${CSS.escape(client.userID.slice(1))}"] {
			background-color: var(--highlight-pill-background-color);
			color: var(--highlight-pill-text-color);
		}
	`, [client.userID])
	useStyle(() => preferences.code_block_line_wrap && css`
		pre.chroma {
			text-wrap: wrap;
		}
	`, [preferences.code_block_line_wrap])
	useStyle(() => preferences.pointer_cursor && css`
		:root {
			--clickable-cursor: pointer;
		}
	`, [preferences.pointer_cursor])
	useStyle(() => !preferences.show_hidden_events && css`
		div.timeline-list > div.hidden-event {
			display: none;
		}
	`, [preferences.show_hidden_events])
	useStyle(() => !preferences.show_redacted_events && css`
		div.timeline-list > div.redacted-event {
			display: none;
		}
	`, [preferences.show_redacted_events])
	useStyle(() => !preferences.show_membership_events && css`
		div.timeline-list > div.membership-event {
			display: none;
		}
	`, [preferences.show_membership_events])
	useStyle(() => !preferences.show_date_separators && css`
		div.timeline-list > div.date-separator {
			display: none;
		}
	`, [preferences.show_date_separators])
	useStyle(() => !preferences.display_read_receipts && css`
		:root {
			--timeline-status-size: 2rem;
		}
	`, [preferences.display_read_receipts])
	useStyle(() => !preferences.show_inline_images && css`
		a.hicli-inline-img-fallback {
			display: inline !important;
		}

		img.hicli-inline-img {
			display: none;
		}
	`, [preferences.show_inline_images])
	useAsyncStyle(() => preferences.code_block_theme === "auto" ? `
		@import url("_gomuks/codeblock/github.css") (prefers-color-scheme: light);
		@import url("_gomuks/codeblock/github-dark.css") (prefers-color-scheme: dark);

		pre.chroma {
			background-color: inherit;
		}
	` : `
		@import url("_gomuks/codeblock/${preferences.code_block_theme}.css");
	`, [preferences.code_block_theme], "gomuks-pref-code-block-theme")
	useAsyncStyle(() => preferences.custom_css, [preferences.custom_css], "gomuks-pref-custom-css")
	useEffect(() => {
		favicon.href = preferences.favicon
	}, [preferences.favicon])
	return null
}

const favicon = document.getElementById("favicon") as HTMLLinkElement

export default React.memo(StylePreferences)
