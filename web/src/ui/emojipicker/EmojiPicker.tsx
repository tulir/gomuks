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
import React, { CSSProperties, JSX, use, useCallback, useState } from "react"
import { getMediaURL } from "@/api/media.ts"
import { RoomStateStore, useCustomEmojis } from "@/api/statestore"
import { roomStateGUIDToString } from "@/api/types"
import { CATEGORY_FREQUENTLY_USED, Emoji, PartialEmoji, categories, useFilteredEmojis } from "@/util/emoji"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import { ModalCloseContext } from "../modal/Modal.tsx"
import { EmojiGroup } from "./EmojiGroup.tsx"
import renderEmoji from "./renderEmoji.tsx"
import FallbackPackIcon from "@/icons/category.svg?react"
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

const sortedEmojiCategories: {index: number, icon: JSX.Element}[] = [
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

interface EmojiPickerProps {
	style: CSSProperties
	onSelect: (emoji: PartialEmoji, isSelected?: boolean) => void
	room: RoomStateStore
	allowFreeform?: boolean
	closeOnSelect?: boolean
	selected?: string[]
}

const EmojiPicker = ({ style, selected, onSelect, room, allowFreeform, closeOnSelect }: EmojiPickerProps) => {
	const client = use(ClientContext)!
	const [query, setQuery] = useState("")
	const [previewEmoji, setPreviewEmoji] = useState<Emoji>()
	const watchedEmojiPackKeys = client.store.getEmojiPackKeys().map(roomStateGUIDToString)
	const customEmojiPacks = useCustomEmojis(client.store, room)
	const emojis = useFilteredEmojis(query, {
		frequentlyUsed: client.store.frequentlyUsedEmoji,
		customEmojiPacks,
	})
	const clearQuery = useCallback(() => setQuery(""), [])
	const close = closeOnSelect ? use(ModalCloseContext) : null
	const onSelectWrapped = useCallback((emoji?: PartialEmoji) => {
		if (!emoji) {
			return
		}
		onSelect(emoji, selected?.includes(emoji.u))
		if (emoji.c) {
			client.incrementFrequentlyUsedEmoji(emoji.u)
				.catch(err => console.error("Failed to increment frequently used emoji", err))
		}
		close?.()
	}, [onSelect, selected, client, close])
	const onClickFreeformReact = useEvent(() => onSelectWrapped({ u: query }))
	const onChangeQuery = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => setQuery(evt.target.value), [])
	const onClickCategoryButton = useCallback((evt: React.MouseEvent) => {
		const categoryID = evt.currentTarget.getAttribute("data-category-id")!
		document.getElementById(`emoji-category-${categoryID}`)?.scrollIntoView()
	}, [])
	return <div className="emoji-picker" style={style}>
		<div className="emoji-category-bar">
			<button
				className="emoji-category-icon"
				data-category-id={CATEGORY_FREQUENTLY_USED}
				title={CATEGORY_FREQUENTLY_USED}
				onClick={onClickCategoryButton}
			><RecentIcon/></button>
			{sortedEmojiCategories.map(cat =>
				<button
					key={cat.index}
					className="emoji-category-icon"
					data-category-id={cat.index}
					title={categories[cat.index]}
					onClick={onClickCategoryButton}
				>{cat.icon}</button>,
			)}
			{customEmojiPacks.map(customPack =>
				<button
					key={customPack.id}
					className="emoji-category-icon custom-emoji"
					data-category-id={customPack.id}
					title={customPack.name}
					onClick={onClickCategoryButton}
				>
					{customPack.icon ? <img src={getMediaURL(customPack.icon)} alt="" /> : <FallbackPackIcon/>}
				</button>,
			)}
		</div>
		<div className="emoji-search">
			<input autoFocus onChange={onChangeQuery} value={query} type="search" placeholder="Search emojis"/>
			<button onClick={clearQuery} disabled={query === ""}>
				{query !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
		<div className="emoji-list">
			{/* Chrome is dumb and doesn't allow scrolling without an inner div */}
			<div className="emoji-list-inner">
				{emojis.map(group => {
					if (!group?.length) {
						return null
					}
					const categoryID = group[0].c
					const customPack = customEmojiPacks.find(pack => pack.id === categoryID)
					return <EmojiGroup
						key={categoryID}
						emojis={group}
						categoryID={categoryID}
						selected={selected}
						pack={customPack}
						isWatched={typeof categoryID === "string" && watchedEmojiPackKeys.includes(categoryID)}
						onSelect={onSelectWrapped}
						setPreviewEmoji={setPreviewEmoji}
					/>
				})}
				{allowFreeform && query && <button
					className="freeform-react"
					onClick={onClickFreeformReact}
				>React with "{query}"</button>}
			</div>
		</div>
		{previewEmoji ? <div className="emoji-preview">
			<div className="big-emoji">{renderEmoji(previewEmoji)}</div>
			<div className="emoji-name">{previewEmoji.t}</div>
			<div className="emoji-shortcode">:{previewEmoji.n}:</div>
		</div> : <div className="emoji-preview"/>}
	</div>
}

export default EmojiPicker
