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
import React, { use, useCallback, useRef, useState } from "react"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, Mentions } from "@/api/types"
import { ClientContext } from "./ClientContext.ts"
import { ReplyBody } from "./timeline/ReplyBody.tsx"
import "./MessageComposer.css"

interface MessageComposerProps {
	room: RoomStateStore
	setTextRows: (rows: number) => void
	replyTo: MemDBEvent | null
	closeReply: () => void
}

const MessageComposer = ({ room, replyTo, setTextRows, closeReply }: MessageComposerProps) => {
	const client = use(ClientContext)!
	const [text, setText] = useState("")
	const textRows = useRef(1)
	const sendMessage = useCallback((evt: React.FormEvent) => {
		evt.preventDefault()
		if (text === "") {
			return
		}
		setText("")
		setTextRows(1)
		textRows.current = 1
		closeReply()
		const room_id = room.roomID
		const mentions: Mentions = {
			user_ids: [],
			room: false,
		}
		if (replyTo) {
			mentions.user_ids.push(replyTo.sender)
		}
		client.sendMessage({ room_id, text, reply_to: replyTo?.event_id, mentions })
			.catch(err => window.alert("Failed to send message: " + err))
	}, [setTextRows, closeReply, replyTo, text, room, client])
	const onKeyDown = useCallback((evt: React.KeyboardEvent) => {
		if (evt.key === "Enter" && !evt.shiftKey) {
			sendMessage(evt)
		}
	}, [sendMessage])
	const onChange = useCallback((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setText(evt.target.value)
		textRows.current = evt.target.value.split("\n").length
		setTextRows(textRows.current)
	}, [setTextRows])
	return <div className="message-composer">
		{replyTo && <ReplyBody room={room} event={replyTo} onClose={closeReply}/>}
		<div className="input-area">
			<textarea
				autoFocus
				rows={textRows.current}
				value={text}
				onKeyDown={onKeyDown}
				onChange={onChange}
				placeholder="Send a message"
				id="message-composer"
			/>
			<button onClick={sendMessage}>Send</button>
		</div>
	</div>
}

export default MessageComposer
