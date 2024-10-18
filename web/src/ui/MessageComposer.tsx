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
import React, { use, useCallback, useLayoutEffect, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { RoomStateStore } from "@/api/statestore"
import { MediaMessageEventContent, MemDBEvent, Mentions, RoomID } from "@/api/types"
import { ClientContext } from "./ClientContext.ts"
import { ReplyBody } from "./timeline/ReplyBody.tsx"
import { useMediaContent } from "./timeline/content/useMediaContent.tsx"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import SendIcon from "@/icons/send.svg?react"
import "./MessageComposer.css"

interface MessageComposerProps {
	room: RoomStateStore
	setTextRows: (rows: number) => void
	replyTo: MemDBEvent | null
	closeReply: () => void
}

const draftStore = {
	get: (roomID: RoomID) => localStorage.getItem(`draft-${roomID}`) ?? "",
	set: (roomID: RoomID, text: string) => localStorage.setItem(`draft-${roomID}`, text),
	clear: (roomID: RoomID)=> localStorage.removeItem(`draft-${roomID}`),
}

const MessageComposer = ({ room, replyTo, setTextRows, closeReply }: MessageComposerProps) => {
	const client = use(ClientContext)!
	const [text, setText] = useState("")
	const [media, setMedia] = useState(null)
	const [loadingMedia, setLoadingMedia] = useState(false)
	const fileInput = useRef<HTMLInputElement>(null)
	const textRows = useRef(1)
	const typingSentAt = useRef(0)
	const fullSetText = useCallback((text: string, setDraft: boolean) => {
		setText(text)
		textRows.current = text === "" ? 1 : text.split("\n").length
		setTextRows(textRows.current)
		if (setDraft) {
			if (text === "") {
				draftStore.clear(room.roomID)
			} else {
				draftStore.set(room.roomID, text)
			}
		}
	}, [setTextRows, room.roomID])
	const sendMessage = useCallback((evt: React.FormEvent) => {
		evt.preventDefault()
		if (text === "" && !media) {
			return
		}
		fullSetText("", true)
		setMedia(null)
		closeReply()
		const room_id = room.roomID
		const mentions: Mentions = {
			user_ids: [],
			room: false,
		}
		if (replyTo) {
			mentions.user_ids.push(replyTo.sender)
		}
		client.sendMessage({ room_id, base_content: media ?? undefined, text, reply_to: replyTo?.event_id, mentions })
			.catch(err => window.alert("Failed to send message: " + err))
	}, [fullSetText, closeReply, replyTo, media, text, room, client])
	const onKeyDown = useCallback((evt: React.KeyboardEvent) => {
		if (evt.key === "Enter" && !evt.shiftKey) {
			sendMessage(evt)
		}
	}, [sendMessage])
	const onChange = useCallback((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		fullSetText(evt.target.value, true)
		const now = Date.now()
		if (evt.target.value !== "" && typingSentAt.current + 5_000 < now) {
			typingSentAt.current = now
			client.rpc.setTyping(room.roomID, 10_000)
				.catch(err => console.error("Failed to send typing notification:", err))
		} else if (evt.target.value == "" && typingSentAt.current > 0) {
			typingSentAt.current = 0
			client.rpc.setTyping(room.roomID, 0)
				.catch(err => console.error("Failed to send stop typing notification:", err))
		}
	}, [client, room.roomID, fullSetText])
	const openFilePicker = useCallback(() => {
		fileInput.current!.click()
	}, [])
	const clearMedia = useCallback(() => {
		setMedia(null)
	}, [])
	const onAttachFile = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setLoadingMedia(true)
		const file = evt.target.files![0]
		const encrypt = !!room.meta.current.encryption_event
		fetch(`/_gomuks/upload?encrypt=${encrypt}&filename=${encodeURIComponent(file.name)}`, {
			method: "POST",
			body: file,
		})
			.then(async res => {
				const json = await res.json()
				if (!res.ok) {
					throw new Error(json.error)
				} else {
					setMedia(json)
				}
			})
			.catch(err => window.alert("Failed to upload file: " + err))
			.finally(() => setLoadingMedia(false))
	}, [room])
	// To ensure the cursor jumps to the end, do this in an effect rather than as the initial value of useState
	// To try to avoid the input bar flashing, use useLayoutEffect instead of useEffect
	useLayoutEffect(() => {
		fullSetText(draftStore.get(room.roomID), false)
		return () => {
			if (typingSentAt.current > 0) {
				typingSentAt.current = 0
				client.rpc.setTyping(room.roomID, 0)
					.catch(err => console.error("Failed to send stop typing notification due to room switch:", err))
			}
		}
	}, [client, room.roomID, fullSetText])
	return <div className="message-composer">
		{replyTo && <ReplyBody room={room} event={replyTo} onClose={closeReply}/>}
		{loadingMedia && <div className="composer-media"><ScaleLoader/></div>}
		{media && <ComposerMedia content={media} clearMedia={clearMedia}/>}
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
			<button
				onClick={openFilePicker}
				disabled={!!media || loadingMedia}
				title={media ? "You can only attach one file at a time" : ""}
			><AttachIcon/></button>
			<button onClick={sendMessage} disabled={(!text && !media) || loadingMedia}><SendIcon/></button>
			<input ref={fileInput} onChange={onAttachFile} type="file" value="" style={{ display: "none" }}/>
		</div>
	</div>
}

interface ComposerMediaProps {
	content: MediaMessageEventContent
	clearMedia: () => void
}

const ComposerMedia = ({ content, clearMedia }: ComposerMediaProps) => {
	// TODO stickers?
	const [mediaContent, containerClass, containerStyle] = useMediaContent(
		content, "m.room.message", { height: 120, width: 360 },
	)
	return <div className="composer-media">
		<div className={`media-container ${containerClass}`} style={containerStyle}>
			{mediaContent}
		</div>
		<button onClick={clearMedia}><CloseIcon/></button>
	</div>
}

export default MessageComposer
