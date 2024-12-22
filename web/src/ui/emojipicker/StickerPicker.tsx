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
import React, { use, useCallback, useState } from "react"
import { getMediaURL } from "@/api/media.ts"
import { useCustomEmojis } from "@/api/statestore"
import { roomStateGUIDToString } from "@/api/types"
import { Emoji, useFilteredEmojis } from "@/util/emoji"
import { isMobileDevice } from "@/util/ismobile.ts"
import ClientContext from "../ClientContext.ts"
import { ModalCloseContext } from "../modal"
import { EmojiGroup } from "./EmojiGroup.tsx"
import { MediaPickerProps } from "./GIFPicker.tsx"
import useCategoryUnderline from "./useCategoryUnderline.ts"
import FallbackPackIcon from "@/icons/category.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import SearchIcon from "@/icons/search.svg?react"

const StickerPicker = ({ style, onSelect, room }: MediaPickerProps) => {
	const client = use(ClientContext)!
	const [query, setQuery] = useState("")
	const [emojiCategoryBarRef, emojiListRef] = useCategoryUnderline()
	const watchedEmojiPackKeys = client.store.getEmojiPackKeys().map(roomStateGUIDToString)
	const customEmojiPacks = useCustomEmojis(client.store, room, "stickers")
	const emojis = useFilteredEmojis(query, {
		// frequentlyUsed: client.store.frequentlyUsedStickers,
		customEmojiPacks,
		stickers: true,
	})
	const close = use(ModalCloseContext)
	const onSelectWrapped = useCallback((emoji?: Emoji) => {
		if (!emoji) {
			return
		}
		onSelect({
			msgtype: "m.sticker",
			body: emoji.t,
			info: emoji.i,
			url: emoji.u,
		})
		close()
	}, [onSelect, close])
	const onClickCategoryButton = (evt: React.MouseEvent) => {
		const categoryID = evt.currentTarget.getAttribute("data-category-id")!
		document.getElementById(`emoji-category-${categoryID}`)?.scrollIntoView()
	}

	return <div className="sticker-picker" style={style}>
		<div className="emoji-category-bar" ref={emojiCategoryBarRef}>
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
			<input
				autoFocus={!isMobileDevice}
				onChange={evt => setQuery(evt.target.value)}
				value={query}
				type="search"
				placeholder="Search stickers"
			/>
			<button onClick={() => setQuery("")} disabled={query === ""}>
				{query !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
		<div className="emoji-list">
			{/* Chrome is dumb and doesn't allow scrolling without an inner div */}
			<div className="emoji-list-inner" ref={emojiListRef}>
				{emojis.map(group => {
					if (!group?.length) {
						return null
					}
					const categoryID = group[0].c as string
					const customPack = customEmojiPacks.find(pack => pack.id === categoryID)
					return <EmojiGroup
						key={categoryID}
						emojis={group}
						categoryID={categoryID}
						pack={customPack}
						isWatched={watchedEmojiPackKeys.includes(categoryID)}
						onSelect={onSelectWrapped}
						imageType="sticker"
					/>
				})}
			</div>
		</div>
	</div>
}

export default StickerPicker
