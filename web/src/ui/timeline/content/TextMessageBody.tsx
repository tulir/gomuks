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
import EventContentProps from "./props.ts"

function isImageElement(elem: EventTarget): elem is HTMLImageElement {
	return (elem as HTMLImageElement).tagName === "IMG"
}

const onClickHTML = (evt: React.MouseEvent<HTMLDivElement>) => {
	if (isImageElement(evt.target)) {
		window.openLightbox({
			src: evt.target.src,
			alt: evt.target.alt,
		})
	} else if ((evt.target as HTMLElement).closest?.("span.hicli-spoiler")?.classList.toggle("spoiler-revealed")) {
		// When unspoilering, don't trigger links and other clickables inside the spoiler
		evt.preventDefault()
	}
}

const TextMessageBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as MessageEventContent
	const classNames = ["message-text"]
	let eventSenderName: string | undefined
	if (content.msgtype === "m.notice") {
		classNames.push("notice-message")
	} else if (content.msgtype === "m.emote") {
		classNames.push("emote-message")
		eventSenderName = sender?.content?.displayname || event.sender
	}
	if (event.local_content?.big_emoji) {
		classNames.push("big-emoji-body")
	}
	if (event.local_content?.was_plaintext) {
		classNames.push("plaintext-body")
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
