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
import { CSSProperties, JSX, use, useCallback, useState } from "react"
import { getMediaURL } from "@/api/media.ts"
import { Emoji, categories, useFilteredEmojis } from "@/util/emoji"
import { ModalCloseContext } from "../modal/Modal.tsx"
import CloseIcon from "@/icons/close.svg?react"
import ActivitiesIcon from "@/icons/emoji-categories/activities.svg?react"
import AnimalsNatureIcon from "@/icons/emoji-categories/animals-nature.svg?react"
import FlagsIcon from "@/icons/emoji-categories/flags.svg?react"
import FoodBeverageIcon from "@/icons/emoji-categories/food-beverage.svg?react"
import ObjectsIcon from "@/icons/emoji-categories/objects.svg?react"
import PeopleBodyIcon from "@/icons/emoji-categories/people-body.svg?react"
import SmileysEmotionIcon from "@/icons/emoji-categories/smileys-emotion.svg?react"
import SymbolsIcon from "@/icons/emoji-categories/symbols.svg?react"
import TravelPlacesIcon from "@/icons/emoji-categories/travel-places.svg?react"
import RecentIcon from "@/icons/schedule.svg?react"
import SearchIcon from "@/icons/search.svg?react"
import "./EmojiPicker.css"

interface EmojiCategory {
	index: number
	name?: string
	icon: JSX.Element
}

const sortedEmojiCategories: EmojiCategory[] = [
	{ index: 7, icon: <SmileysEmotionIcon/> },
	{ index: 6, icon: <PeopleBodyIcon/> },
	{ index: 1, icon: <AnimalsNatureIcon/> },
	{ index: 4, icon: <FoodBeverageIcon/> },
	{ index: 9, icon: <TravelPlacesIcon/> },
	{ index: 0, icon: <ActivitiesIcon/> },
	{ index: 5, icon: <ObjectsIcon/> },
	{ index: 8, icon: <SymbolsIcon/> },
	{ index: 3, icon: <FlagsIcon/> },
]

function renderEmoji(emoji: Emoji): JSX.Element | string {
	if (emoji.u.startsWith("mxc://")) {
		return <img src={getMediaURL(emoji.u)} alt={`:${emoji.n}:`}/>
	}
	return emoji.u
}

interface EmojiPickerProps {
	style: CSSProperties
	onSelect: (emoji: Partial<Emoji>) => void
	allowFreeform?: boolean
	closeOnSelect?: boolean
}

export const EmojiPicker = ({ style, onSelect, allowFreeform, closeOnSelect }: EmojiPickerProps) => {
	const [query, setQuery] = useState("")
	const emojis = useFilteredEmojis(query)
	const [previewEmoji, setPreviewEmoji] = useState<Emoji>()
	const clearQuery = useCallback(() => setQuery(""), [])
	const onChangeQuery = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => setQuery(evt.target.value), [])
	const cats: JSX.Element[] = []
	let currentCat: JSX.Element[] | undefined
	let currentCatNum: number | string = -1
	const close = use(ModalCloseContext)
	const onSelectWrapped = (emoji: Partial<Emoji>) => {
		onSelect(emoji)
		if (closeOnSelect) {
			close()
		}
	}
	for (const emoji of emojis) {
		if (emoji.c === 2) {
			continue
		}
		if (emoji.c !== currentCatNum || !currentCat) {
			if (currentCat) {
				cats.push(<div className="emoji-category" key={currentCatNum} data-emoji-category={currentCatNum}>
					<h4 className="emoji-category-name">{
						typeof currentCatNum === "number" ? categories[currentCatNum] : currentCatNum
					}</h4>
					<div className="emoji-category-list">
						{currentCat}
					</div>
				</div>)
			}
			currentCatNum = emoji.c
			currentCat = []
		}
		currentCat.push(<button
			key={emoji.u}
			className="emoji"
			onMouseOver={() => setPreviewEmoji(emoji)}
			onMouseOut={() => setPreviewEmoji(undefined)}
			onClick={() => onSelectWrapped(emoji)}
		>{renderEmoji(emoji)}</button>)
	}
	return <div className="emoji-picker" style={style}>
		<div className="emoji-category-bar">
			<button
				className="emoji-category-icon"
				title={"Recently used"}
			>{<RecentIcon/>}</button>
			{sortedEmojiCategories.map(cat =>
				<button
					key={cat.index}
					className="emoji-category-icon"
					title={cat.name ?? categories[cat.index]}
				>{cat.icon}</button>,
			)}
		</div>
		<div className="emoji-search">
			<input autoFocus onChange={onChangeQuery} value={query} type="search" placeholder="Search emojis"/>
			<button onClick={clearQuery} disabled={query === ""}>
				{query !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
		<div className="emoji-list">
			{cats}
			{allowFreeform && query && <button
				className="freeform-react"
				onClick={() => onSelectWrapped({ u: query })}
			>React with "{query}"</button>}
		</div>
		{previewEmoji ? <div className="emoji-preview">
			<div className="big-emoji">{renderEmoji(previewEmoji)}</div>
			<div className="emoji-name">{previewEmoji.t}</div>
			<div className="emoji-shortcode">:{previewEmoji.n}:</div>
		</div> : <div className="emoji-preview"/>}
	</div>
}

export default EmojiPicker
