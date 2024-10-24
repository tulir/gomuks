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
import { JSX, useEffect } from "react"
import { getAvatarURL } from "@/api/media.ts"
import { RoomStateStore } from "@/api/statestore"
import { Emoji, useFilteredEmojis } from "@/util/emoji"
import useEvent from "@/util/useEvent.ts"
import type { ComposerState } from "./MessageComposer.tsx"
import { AutocompleteUser, useFilteredMembers } from "./userautocomplete.ts"
import "./Autocompleter.css"

export interface AutocompleteQuery {
	type: "user" | "room" | "emoji"
	query: string
	startPos: number
	endPos: number
	frozenQuery?: string
	selected?: number
}

export interface AutocompleterProps {
	setState: (state: Partial<ComposerState>) => void
	setAutocomplete: (params: AutocompleteQuery | null) => void
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
	params, state, setState, setAutocomplete,
	items, getText, getKey, render,
}: InnerAutocompleterProps<T>) {
	const onSelect = useEvent((index: number) => {
		if (items.length === 0) {
			return
		}
		index = positiveMod(index, items.length)
		const replacementText = getText(items[index])
		setState({
			text: state.text.slice(0, params.startPos) + replacementText + state.text.slice(params.endPos),
		})
		setAutocomplete({
			...params,
			endPos: params.startPos + replacementText.length,
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
		}
	}, [onSelect, params.selected])
	const selected = params.selected !== undefined	 ? positiveMod(params.selected, items.length) : -1
	return <div className="autocompletions">
		{items.map((item, i) => <div
			onClick={onClick}
			data-index={i}
			className={`autocompletion-item ${selected === i ? "selected" : ""}`}
			key={getKey(item)}
		>{render(item)}</div>)}
	</div>
}

const emojiFuncs = {
	getText: (emoji: Emoji) => emoji.u,
	getKey: (emoji: Emoji) => emoji.u,
	render: (emoji: Emoji) => <>{emoji.u} :{emoji.n}:</>,
}

export const EmojiAutocompleter = ({ params, ...rest }: AutocompleterProps) => {
	const items = useFilteredEmojis((params.frozenQuery ?? params.query).slice(1), true)
	return useAutocompleter({ params, ...rest, items, ...emojiFuncs })
}

const escapeDisplayname = (input: string) => input
	.replace("\n", " ")
	.replace(/([\\`*_[\]])/g, "\\$1")
	.replace("<", "&lt;")
	.replace(">", "&gt;")

const userFuncs = {
	getText: (user: AutocompleteUser) =>
		`[${escapeDisplayname(user.displayName)}](https://matrix.to/#/${encodeURIComponent(user.userID)}) `,
	getKey: (user: AutocompleteUser) => user.userID,
	render: (user: AutocompleteUser) => <>
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
