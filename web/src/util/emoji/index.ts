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
import { useRef } from "react"
import data from "./data.json"

export interface Emoji {
	u: string // Unicode codepoint or custom emoji mxc:// URI
	c: number | string // Category number or custom emoji pack name
	t: string // Emoji title
	n: string // Primary shortcode
	s: string[] // Shortcodes without underscores
}

export const emojis: Emoji[] = data.e
export const categories = data.c

function filter(emojis: Emoji[], query: string): Emoji[] {
	return emojis.filter(emoji => emoji.s.some(shortcode => shortcode.includes(query)))
}

function filterAndSort(emojis: Emoji[], query: string): Emoji[] {
	return emojis
		.map(emoji => {
			const matchIndex = emoji.s.reduce((minIndex, shortcode) => {
				const index = shortcode.indexOf(query)
				return index !== -1 && (minIndex === -1 || index < minIndex) ? index : minIndex
			}, -1)
			return { emoji, matchIndex }
		})
		.filter(({ matchIndex }) => matchIndex !== -1)
		.sort((e1, e2) => e1.matchIndex - e2.matchIndex)
		.map(({ emoji }) => emoji)
}

export function search(query: string, sorted = false, prev?: Emoji[]): Emoji[] {
	query = query.toLowerCase().replaceAll("_", "")
	if (!query) return emojis
	return (sorted ? filterAndSort : filter)(prev ?? emojis, query)
}

interface filteredEmojiCache {
	query: string
	result: Emoji[]
}

export function useFilteredEmojis(query: string, sorted = false): Emoji[] {
	query = query.toLowerCase().replaceAll("_", "")
	const prev = useRef<filteredEmojiCache>({ query: "", result: emojis })
	if (!query) {
		prev.current.query = ""
		prev.current.result = emojis
	} else if (prev.current.query !== query) {
		prev.current.result = (sorted ? filterAndSort : filter)(
			query.startsWith(prev.current.query) ? prev.current.result : emojis,
			query,
		)
		prev.current.query = query
	}
	return prev.current.result
}
