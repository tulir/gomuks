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
import { MessageEventContent } from "@/api/types"
import { getDisplayname } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function isImageElement(elem: EventTarget): elem is HTMLImageElement {
	return (elem as HTMLImageElement).tagName === "IMG"
}

function isAnchorElement(elem: EventTarget): elem is HTMLAnchorElement {
	return (elem as HTMLAnchorElement).tagName === "A"
}

function onClickMatrixURI(href: string) {
	const url = new URL(href)
	const pathParts = url.pathname.split("/")
	const decodedPart = decodeURIComponent(pathParts[1])
	switch (pathParts[0]) {
	case "u":
		return window.mainScreenContext.setRightPanel({
			type: "user",
			userID: `@${decodedPart}`,
		})
	case "roomid":
		return window.mainScreenContext.setActiveRoom(`!${decodedPart}`)
	case "r":
		return window.client.rpc.resolveAlias(`#${decodedPart}`).then(
			res => window.mainScreenContext.setActiveRoom(res.room_id),
			err => window.alert(`Failed to resolve room alias #${decodedPart}: ${err}`),
		)
	}
}

const onClickHTML = (evt: React.MouseEvent<HTMLDivElement>) => {
	const targetElem = evt.target as HTMLElement
	if (isImageElement(targetElem)) {
		window.openLightbox({
			src: targetElem.src,
			alt: targetElem.alt,
		})
	} else if (targetElem.closest?.("span.hicli-spoiler")?.classList.toggle("spoiler-revealed")) {
		// When unspoilering, don't trigger links and other clickables inside the spoiler
		evt.preventDefault()
	} else if (isAnchorElement(targetElem) && targetElem.href.startsWith("matrix:")) {
		onClickMatrixURI(targetElem.href)
		evt.preventDefault()
	}
}

let mathImported = false

function importMath() {
	if (mathImported) {
		return
	}
	mathImported = true
	import("./math.ts").then(
		() => console.info("Imported math"),
		err => console.error("Failed to import math", err),
	)
}

const TextMessageBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as MessageEventContent
	const classNames = ["message-text"]
	let eventSenderName: string | undefined
	if (content.msgtype === "m.notice") {
		classNames.push("notice-message")
	} else if (content.msgtype === "m.emote") {
		classNames.push("emote-message")
		eventSenderName = getDisplayname(event.sender, sender?.content)
	}
	if (event.local_content?.big_emoji) {
		classNames.push("big-emoji-body")
	}
	if (event.local_content?.was_plaintext) {
		classNames.push("plaintext-body")
	}
	if (event.local_content?.has_math) {
		classNames.push("math-body")
		importMath()
	}
	if (event.local_content?.sanitized_html) {
		classNames.push("html-body")
		return <div
			onClick={onClickHTML}
			className={classNames.join(" ")}
			data-event-sender={eventSenderName}
			dangerouslySetInnerHTML={{ __html: event.local_content.sanitized_html }}
		/>
	}
	return <div className={classNames.join(" ")} data-event-sender={eventSenderName}>{content.body}</div>
}

export default TextMessageBody
