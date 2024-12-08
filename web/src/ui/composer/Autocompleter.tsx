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
import { JSX, RefObject, use, useEffect } from "react"
import { getAvatarURL, getMediaURL } from "@/api/media.ts"
import { AutocompleteMemberEntry, RoomStateStore, useCustomEmojis } from "@/api/statestore"
import { Emoji, emojiToMarkdown, useSortedAndFilteredEmojis } from "@/util/emoji"
import { escapeMarkdown } from "@/util/markdown.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import type { ComposerState } from "./MessageComposer.tsx"
import { useFilteredMembers } from "./userautocomplete.ts"
import "./Autocompleter.css"

export interface AutocompleteQuery {
	type: "user" | "room" | "emoji"
	query: string
	startPos: number
	endPos: number
	frozenQuery?: string
	selected?: number
	close?: boolean
}

export interface AutocompleterProps {
	setState: (state: Partial<ComposerState>) => void
	setAutocomplete: (params: AutocompleteQuery | null) => void
	textInput: RefObject<HTMLTextAreaElement | null>
	state: ComposerState
	params: AutocompleteQuery
	room: RoomStateStore
}

const positiveMod = (val: number, div: number) => (val % div + div) % div

interface InnerAutocompleterProps<T> extends AutocompleterProps {
	items: T[]
	getText: (item: T) => string
	getKey: (item: T) => string
	render: (item: T) => JSX.Element
}

function useAutocompleter<T>({
	params, state, setState, setAutocomplete, textInput,
	items, getText, getKey, render,
}: InnerAutocompleterProps<T>) {
	const onSelect = useEvent((index: number) => {
		if (items.length === 0) {
			return
		}
		index = positiveMod(index, items.length)
		const replacementText = getText(items[index])
		const newText = state.text.slice(0, params.startPos) + replacementText + state.text.slice(params.endPos)
		const endPos = params.startPos + replacementText.length
		if (textInput.current) {
			// React messes up the selection when changing the value for some reason,
			// so bypass react here to avoid the caret jumping to the end and closing the autocompleter
			textInput.current.value = newText
			textInput.current.setSelectionRange(endPos, endPos)
		}
		setState({
			text: newText,
		})
		setAutocomplete({
			...params,
			endPos,
			frozenQuery: params.frozenQuery ?? params.query,
		})
		document.querySelector(`div.autocompletion-item[data-index='${index}']`)?.scrollIntoView({ block: "nearest" })
	})
	const onClick = useEvent((evt: React.MouseEvent<HTMLDivElement>) => {
		const idx = evt.currentTarget.getAttribute("data-index")
		if (idx) {
			onSelect(+idx)
			setAutocomplete(null)
		}
	})
	useEffect(() => {
		if (params.selected !== undefined) {
			onSelect(params.selected)
			if (params.close) {
				setAutocomplete(null)
			}
		}
	}, [onSelect, setAutocomplete, params.selected, params.close])
	const selected = params.selected !== undefined ? positiveMod(params.selected, items.length) : -1
	return <div
		className={`autocompletions ${items.length === 0 ? "empty" : "has-items"}`}
		id="composer-autocompletions"
	>
		{items.map((item, i) => <div
			onClick={onClick}
			data-index={i}
			className={`autocompletion-item ${selected === i ? "selected" : ""}`}
			key={getKey(item)}
		>{render(item)}</div>)}
	</div>
}

const emojiFuncs = {
	getText: (emoji: Emoji) => emojiToMarkdown(emoji),
	getKey: (emoji: Emoji) => `${emoji.c}-${emoji.u}`,
	render: (emoji: Emoji) => <>{emoji.u.startsWith("mxc://")
		? <img loading="lazy" src={getMediaURL(emoji.u)} alt={`:${emoji.n}:`}/>
		: emoji.u
	} :{emoji.n}:</>,
}

export const EmojiAutocompleter = ({ params, room, ...rest }: AutocompleterProps) => {
	const client = use(ClientContext)!
	const customEmojiPacks = useCustomEmojis(client.store, room)
	const items = useSortedAndFilteredEmojis((params.frozenQuery ?? params.query).slice(1), {
		frequentlyUsed: client.store.frequentlyUsedEmoji,
		customEmojiPacks,
	})
	return useAutocompleter({ params, room, ...rest, items, ...emojiFuncs })
}

const escapeDisplayname = (input: string) => escapeMarkdown(input).replace("\n", " ")

const userFuncs = {
	getText: (user: AutocompleteMemberEntry) =>
		`[${escapeDisplayname(user.displayName)}](https://matrix.to/#/${encodeURIComponent(user.userID)}) `,
	getKey: (user: AutocompleteMemberEntry) => user.userID,
	render: (user: AutocompleteMemberEntry) => <>
		<img
			className="small avatar"
			loading="lazy"
			src={getAvatarURL(user.userID, { displayname: user.displayName, avatar_url: user.avatarURL })}
			alt=""
		/>
		{user.displayName}
	</>,
}

export const UserAutocompleter = ({ params, room, ...rest }: AutocompleterProps) => {
	const items = useFilteredMembers(room, (params.frozenQuery ?? params.query).slice(1))
	return useAutocompleter({ params, room, ...rest, items, ...userFuncs })
}

export const RoomAutocompleter = ({ params }: AutocompleterProps) => {
	return <div className="autocompletions">
		Autocomplete {params.type} {params.query}
	</div>
}
