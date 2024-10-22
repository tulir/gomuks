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
import { useEffect } from "react"
import { RoomStateStore } from "@/api/statestore"
import { useFilteredEmojis } from "@/util/emoji"
import useEvent from "@/util/useEvent.ts"
import type { ComposerState } from "./MessageComposer.tsx"
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

export const EmojiAutocompleter = ({ params, state, setState, setAutocomplete }: AutocompleterProps) => {
	const emojis = useFilteredEmojis((params.frozenQuery ?? params.query).slice(1), true)
	const onSelect = useEvent((index: number) => {
		if (emojis.length === 0) {
			return
		}
		index = positiveMod(index, emojis.length)
		const emoji = emojis[index]
		setState({
			text: state.text.slice(0, params.startPos) + emoji.u + state.text.slice(params.endPos),
		})
		setAutocomplete({
			...params,
			endPos: params.startPos + emoji.u.length,
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
	const selected = params.selected !== undefined	 ? positiveMod(params.selected, emojis.length) : -1
	return <div className="autocompletions">
		{emojis.map((emoji, i) => <div
			onClick={onClick}
			data-index={i}
			className={`autocompletion-item ${selected === i ? "selected" : ""}`}
			key={emoji.u}
		>{emoji.u} :{emoji.n}:</div>)}
	</div>
}

export const UserAutocompleter = ({ params }: AutocompleterProps) => {
	return <div className="autocompletions">
		Autocomplete {params.type} {params.query}
	</div>
}

export const RoomAutocompleter = ({ params }: AutocompleterProps) => {
	return <div className="autocompletions">
		Autocomplete {params.type} {params.query}
	</div>
}
