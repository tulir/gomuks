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
import React, { use } from "react"
import { stringToRoomStateGUID } from "@/api/types"
import useContentVisibility from "@/util/contentvisibility.ts"
import { CATEGORY_FREQUENTLY_USED, CustomEmojiPack, Emoji, categories } from "@/util/emoji"
import ClientContext from "../ClientContext.ts"
import renderEmoji from "./renderEmoji.tsx"

interface EmojiGroupProps {
	emojis: Emoji[]
	categoryID: number | string
	selected?: string[]
	pack?: CustomEmojiPack
	isWatched?: boolean
	onSelect: (emoji?: Emoji) => void
	setPreviewEmoji?: (emoji?: Emoji) => void
	imageType: "emoji" | "sticker"
}

export const EmojiGroup = ({
	emojis,
	categoryID,
	selected,
	pack,
	isWatched,
	onSelect,
	setPreviewEmoji,
	imageType,
}: EmojiGroupProps) => {
	const client = use(ClientContext)!
	const [isVisible, divRef] = useContentVisibility<HTMLDivElement>(true)

	const getEmojiFromAttrs = (elem: HTMLButtonElement) => {
		const idx = elem.getAttribute("data-emoji-index")
		if (!idx) {
			return
		}
		const emoji = emojis[+idx]
		if (!emoji) {
			return
		}
		return emoji
	}
	const onMouseOverEmoji = setPreviewEmoji && ((evt: React.MouseEvent<HTMLButtonElement>) =>
		setPreviewEmoji(getEmojiFromAttrs(evt.currentTarget)))
	const onMouseOutEmoji = setPreviewEmoji && (() => setPreviewEmoji(undefined))
	const onClickSubscribePack = (evt: React.MouseEvent<HTMLButtonElement>) => {
		const guid = stringToRoomStateGUID(evt.currentTarget.getAttribute("data-pack-id"))
		if (!guid) {
			return
		}
		client.subscribeToEmojiPack(guid, true)
			.catch(err => window.alert(`Failed to subscribe to emoji pack: ${err}`))
	}
	const onClickUnsubscribePack = (evt: React.MouseEvent<HTMLButtonElement>) => {
		const guid = stringToRoomStateGUID(evt.currentTarget.getAttribute("data-pack-id"))
		if (!guid) {
			return
		}
		client.subscribeToEmojiPack(guid, false)
			.catch(err => window.alert(`Failed to unsubscribe from emoji pack: ${err}`))
	}

	let categoryName: string
	if (typeof categoryID === "number") {
		categoryName = categories[categoryID]
	} else if (categoryID === CATEGORY_FREQUENTLY_USED) {
		categoryName = CATEGORY_FREQUENTLY_USED
	} else if (pack) {
		categoryName = pack.name
	} else {
		categoryName = "Unknown category"
	}
	const itemsPerRow = imageType === "sticker" ? 4 : 8
	const rowSize = imageType === "sticker" ? 5 : 2.5
	return <div
		ref={divRef}
		className="emoji-category"
		id={`emoji-category-${categoryID}`}
		data-category-id={categoryID}
		style={{ containIntrinsicHeight: `${1.5 + Math.ceil(emojis.length / itemsPerRow) * rowSize}rem` }}
	>
		<h4 className="emoji-category-name">
			<span title={categoryName}>{categoryName}</span>
			{pack && <button
				className="emoji-category-add"
				onClick={isWatched ? onClickUnsubscribePack : onClickSubscribePack}
				data-pack-id={categoryID}
			>{isWatched ? "Unsubscribe" : "Subscribe"}</button>}
		</h4>
		<div className="emoji-category-list">
			{isVisible ? emojis.map((emoji, idx) => <button
				key={emoji.u}
				className={`${imageType} ${selected?.includes(emoji.u) ? "selected" : ""}`}
				data-emoji-index={idx}
				onMouseOver={onMouseOverEmoji}
				onMouseOut={onMouseOutEmoji}
				onClick={evt => onSelect(getEmojiFromAttrs(evt.currentTarget))}
				title={emoji.t}
			>{renderEmoji(emoji)}</button>) : null}
		</div>
	</div>
}
