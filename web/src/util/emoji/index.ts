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
import { useMemo, useRef } from "react"
import { EventID, ReactionEventContent } from "@/api/types"
import data from "./data.json"

export interface EmojiMetadata {
	c: number | string // Category number or custom emoji pack name
	t: string // Emoji title
	n: string // Primary shortcode
	s: string[] // Shortcodes without underscores
}

export interface EmojiText {
	u: string // Unicode codepoint or custom emoji mxc:// URI
}

export type PartialEmoji = EmojiText & Partial<EmojiMetadata>
export type Emoji = EmojiText & EmojiMetadata

export const emojis: Emoji[] = data.e
export const emojiMap = new Map<string, Emoji>()
export const categories = data.c

export const CATEGORY_FREQUENTLY_USED = "Frequently Used"

for (const emoji of emojis) {
	emojiMap.set(emoji.u, emoji)
}

function filter(emojis: Emoji[], query: string): Emoji[] {
	return emojis.filter(emoji => emoji.s.some(shortcode => shortcode.includes(query)))
}

function filterAndSort(emojis: Emoji[], query: string, frequentlyUsed?: Map<string, number>): Emoji[] {
	return emojis
		.map(emoji => {
			const matchIndex = emoji.s.reduce((minIndex, shortcode) => {
				const index = shortcode.indexOf(query)
				return index !== -1 && (minIndex === -1 || index < minIndex) ? index : minIndex
			}, -1)
			return { emoji, matchIndex }
		})
		.filter(({ matchIndex }) => matchIndex !== -1)
		.sort((e1, e2) =>
			e1.matchIndex === e2.matchIndex
				? (frequentlyUsed?.get(e2.emoji.u) ?? 0) - (frequentlyUsed?.get(e1.emoji.u) ?? 0)
				: e1.matchIndex - e2.matchIndex)
		.map(({ emoji }) => emoji)
}

export function search(query: string, sorted = false, prev?: Emoji[]): Emoji[] {
	query = query.toLowerCase().replaceAll("_", "")
	if (!query) return emojis
	return (sorted ? filterAndSort : filter)(prev ?? emojis, query)
}

export function emojiToMarkdown(emoji: PartialEmoji): string {
	if (emoji.u.startsWith("mxc://")) {
		return `<img data-mx-emoticon src="${emoji.u}" alt=":${emoji.n}:" title=":${emoji.n}:"/>`
	}
	return emoji.u
}

export function emojiToReactionContent(emoji: PartialEmoji, evtID: EventID): ReactionEventContent {
	const content: ReactionEventContent = {
		"m.relates_to": {
			rel_type: "m.annotation",
			event_id: evtID,
			key: emoji.u,
		},
	}
	if (emoji.u?.startsWith("mxc://") && emoji.n) {
		content["com.beeper.emoji.shortcode"] = emoji.n
	}
	return content
}

interface filteredEmojiCache {
	query: string
	result: Emoji[]
}

interface useFilteredEmojisParams {
	sorted?: boolean
	frequentlyUsed?: Map<string, number>
	frequentlyUsedAsCategory?: boolean
}

export function useFilteredEmojis(query: string, params: useFilteredEmojisParams = {}): Emoji[] {
	query = query.toLowerCase().replaceAll("_", "")
	const allEmojis: Emoji[] = useMemo(() => {
		let output: Emoji[] = []
		if (params.frequentlyUsedAsCategory && params.frequentlyUsed) {
			output = Array.from(params.frequentlyUsed.keys()
				.map(key => {
					const emoji = emojiMap.get(key)
					if (!emoji) {
						return undefined
					}
					return { ...emoji, c: CATEGORY_FREQUENTLY_USED } as Emoji
				})
				.filter((emoji, index): emoji is Emoji => emoji !== undefined && index < 24))
		}
		if (output.length === 0) {
			return emojis
		}
		return output.concat(emojis)
	}, [params.frequentlyUsed, params.frequentlyUsedAsCategory])
	const prev = useRef<filteredEmojiCache>({ query: "", result: allEmojis })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allEmojis
	} else if (prev.current.query !== query) {
		prev.current.result = (params.sorted ? filterAndSort : filter)(
			query.startsWith(prev.current.query) ? prev.current.result : allEmojis,
			query,
			params.frequentlyUsed,
		)
		prev.current.query = query
	}
	return prev.current.result
}
