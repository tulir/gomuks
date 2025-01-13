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
import { ContentURI, EventID, ImagePack, MediaInfo, ReactionEventContent } from "@/api/types"
import data from "./data.json"

export interface EmojiMetadata {
	c: number | string // Category number or custom emoji pack name
	t: string // Emoji title
	n: string // Primary shortcode
	s: string[] // Shortcodes without underscores
	i?: MediaInfo // Media info for stickers only
}

export interface EmojiText {
	u: string // Unicode codepoint or custom emoji mxc:// URI
}

export type PartialEmoji = EmojiText & Partial<EmojiMetadata>
export type Emoji = EmojiText & EmojiMetadata

export const emojis: Emoji[] = data.e
export const emojiMap = new Map<string, Emoji>()
export const emojisByCategory: Emoji[][] = []
export const categories = data.c

export const CATEGORY_FREQUENTLY_USED = "Frequently Used"

function initEmojiMaps() {
	let building: Emoji[] = []
	let buildingCat: number = -1
	for (const emoji of emojis) {
		emojiMap.set(emoji.u, emoji)
		if (emoji.c === 2) {
			continue
		}
		if (emoji.c !== buildingCat) {
			if (building.length) {
				emojisByCategory.push(building)
			}
			buildingCat = emoji.c as number
			building = []
		}
		building.push(emoji)
	}
	if (building.length) {
		emojisByCategory.push(building)
	}
}

initEmojiMaps()

function filter(emojis: Emoji[], query: string): Emoji[] {
	return emojis.filter(emoji => emoji.s.some(shortcode => shortcode.includes(query)))
}

function filterAndSort(
	emojis: Emoji[],
	query: string,
	frequentlyUsed?: Map<string, number>,
	customEmojis?: CustomEmojiPack[],
	stickers?: true,
): Emoji[] {
	const filteredStandardEmojis = emojis
		.map(emoji => {
			const matchIndex = emoji.s.reduce((minIndex, shortcode) => {
				const index = shortcode.indexOf(query)
				return index !== -1 && (minIndex === -1 || index < minIndex) ? index : minIndex
			}, -1)
			return { emoji, matchIndex }
		})
		.filter(({ matchIndex }) => matchIndex !== -1)
	const filteredCustomEmojis = customEmojis
		?.flatMap(pack => (stickers ? pack.stickers : pack.emojis)
			.map(emoji => {
				const matchIndex = emoji.s.reduce((minIndex, shortcode) => {
					const index = shortcode.indexOf(query)
					return index !== -1 && (minIndex === -1 || index < minIndex) ? index : minIndex
				}, -1)
				return { emoji, matchIndex }
			})
			.filter(({ matchIndex }) => matchIndex !== -1)) ?? []
	const allEmojis =
		filteredStandardEmojis.length
			? filteredCustomEmojis.length
				? filteredStandardEmojis.concat(filteredCustomEmojis)
				: filteredStandardEmojis
			: filteredCustomEmojis
	return allEmojis
		.sort((e1, e2) =>
			e1.matchIndex === e2.matchIndex
				? (frequentlyUsed?.get(e2.emoji.u) ?? 0) - (frequentlyUsed?.get(e1.emoji.u) ?? 0)
				: e1.matchIndex - e2.matchIndex)
		.map(({ emoji }) => emoji)
}

export function emojiToMarkdown(emoji: PartialEmoji): string {
	if (emoji.u.startsWith("mxc://")) {
		const title = emoji.t && emoji.t !== emoji.n ? emoji.t : `:${emoji.n}:`
		const escapedTitle = title.replaceAll(`\\`, `\\\\`).replaceAll(`"`, `\\"`)
		return `![:${emoji.n}:](${emoji.u} "Emoji: ${escapedTitle}")`
		//return `<img data-mx-emoticon height="32" src="${emoji.u}" alt=":${emoji.n}:" title="${title}"/>`
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
		content["com.beeper.reaction.shortcode"] = `:${emoji.n}:`
	}
	return content
}

export interface CustomEmojiPack {
	id: string
	name: string
	icon?: ContentURI
	emojis: Emoji[]
	stickers: Emoji[]
	emojiMap: Map<string, Emoji>
}

export function parseCustomEmojiPack(
	pack: ImagePack,
	id: string,
	fallbackName?: string,
): CustomEmojiPack | null {
	try {
		const name = pack.pack.display_name || fallbackName || "Unnamed pack"
		const emojiMap = new Map<string, Emoji>()
		const stickers: Emoji[] = []
		const emojis: Emoji[] = []
		const defaultIsEmoji = !pack.pack.usage || pack.pack.usage.includes("emoticon")
		const defaultIsSticker = !pack.pack.usage || pack.pack.usage.includes("sticker")
		for (const [shortcode, image] of Object.entries(pack.images)) {
			if (!image.url) {
				continue
			}
			let converted = emojiMap.get(image.url)
			if (converted) {
				converted.s.push(shortcode.toLowerCase().replaceAll("_", "").replaceAll(" ", ""))
			} else {
				converted = {
					c: id,
					u: image.url,
					n: shortcode,
					s: [shortcode.toLowerCase().replaceAll("_", "").replaceAll(" ", "")],
					t: image.body || shortcode,
					i: image.info,
				}
				emojiMap.set(image.url, converted)
				const isSticker = image.usage ? image.usage.includes("sticker") : defaultIsSticker
				const isEmoji = image.usage ? image.usage.includes("emoticon") : defaultIsEmoji
				if (isEmoji) {
					emojis.push(converted)
				}
				if (isSticker) {
					stickers.push(converted)
				}
			}
		}
		const icon = pack.pack.avatar_url || (emojis[0] ?? stickers[0])?.u
		return {
			id,
			name,
			icon,
			emojis,
			stickers,
			emojiMap,
		}
	} catch (err) {
		console.warn("Failed to parse custom emoji pack", pack, err)
		return null
	}
}

interface filteredEmojiCache {
	query: string
	result: Emoji[][]
}

interface filteredAndSortedEmojiCache {
	query: string
	result: Emoji[] | null
}

interface useEmojisParams {
	frequentlyUsed?: Map<string, number>
	customEmojiPacks?: CustomEmojiPack[]
	stickers?: true
}

export function useFilteredEmojis(query: string, params: useEmojisParams = {}): Emoji[][] {
	query = query.toLowerCase().replaceAll("_", "").replaceAll(" ", "")
	const frequentlyUsedCategory: Emoji[] = useMemo(() => {
		if (!params.frequentlyUsed?.size) {
			return []
		}
		return Array.from(params.frequentlyUsed.keys()
			.map(key => {
				let emoji: Emoji | undefined
				if (key.startsWith("mxc://")) {
					for (const pack of params.customEmojiPacks?.values() ?? []) {
						emoji = pack.emojiMap.get(key)
						if (emoji) {
							break
						}
					}
				} else {
					emoji = emojiMap.get(key)
				}
				if (!emoji) {
					return undefined
				}
				return { ...emoji, c: CATEGORY_FREQUENTLY_USED } as Emoji
			})
			.filter(emoji => emoji !== undefined))
			.filter((_emoji, index) => index < 24)
	}, [params.frequentlyUsed, params.customEmojiPacks])
	const prev = useRef<filteredEmojiCache>({
		query: "",
		result: [],
	})
	const baseEmojis = params.stickers ? [] : emojisByCategory
	const categoriesChanged = prev.current.result.length !==
		(1 + baseEmojis.length + (params.customEmojiPacks?.length ?? 0))
	if (prev.current.query !== query || categoriesChanged) {
		if (!query.startsWith(prev.current.query) || categoriesChanged) {
			prev.current.result = [
				frequentlyUsedCategory,
				...baseEmojis,
				...(params.customEmojiPacks?.map(pack => params.stickers ? pack.stickers : pack.emojis) ?? []),
			]
		}
		if (query !== "") {
			prev.current.result = prev.current.result.map(pack => filter(pack, query))
		}
		prev.current.query = query
	}
	return prev.current.result
}

export function useSortedAndFilteredEmojis(query: string, params: useEmojisParams = {}): Emoji[] {
	if (!query) {
		throw new Error("useSortedAndFilteredEmojis requires a query")
	}
	query = query.toLowerCase().replaceAll("_", "")

	const prev = useRef<filteredAndSortedEmojiCache>({ query: "", result: null })
	if (prev.current.query !== query) {
		if (prev.current.result != null && query.startsWith(prev.current.query)) {
			prev.current.result = filterAndSort(prev.current.result, query, params.frequentlyUsed)
		} else {
			prev.current.result = filterAndSort(emojis, query, params.frequentlyUsed, params.customEmojiPacks)
		}
		prev.current.query = query
	}
	return prev.current.result ?? []
}
