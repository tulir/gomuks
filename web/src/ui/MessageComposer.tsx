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
import { RoomStateStore } from "../api/statestore.ts"
import { ClientContext } from "./ClientContext.ts"
import "./MessageComposer.css"

interface MessageComposerProps {
	room: RoomStateStore
}

const MessageComposer = ({ room }: MessageComposerProps) => {
	const client = use(ClientContext)!
	const [text, setText] = useState("")
	const sendMessage = useCallback((evt: React.FormEvent) => {
		evt.preventDefault()
		setText("")
		client.sendMessage(room.roomID, text)
			.catch(err => window.alert("Failed to send message: " + err))
	}, [text, room, client])
	return <div className="message-composer">
		<textarea
			autoFocus
			rows={text.split("\n").length}
			value={text}
			onKeyDown={evt => {
				if (evt.key === "Enter" && !evt.shiftKey) {
					sendMessage(evt)
				}
			}}
			onChange={evt => setText(evt.target.value)}
			placeholder="Send a message"
			id="message-composer"
		/>
		<button onClick={sendMessage}>Send</button>
	</div>
}

export default MessageComposer
