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
import React, { use, useCallback, useEffect, useLayoutEffect, useReducer, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { RoomStateStore, useRoomEvent } from "@/api/statestore"
import { EventID, MediaMessageEventContent, Mentions, RoomID } from "@/api/types"
import { ClientContext } from "./ClientContext.ts"
import { ReplyBody } from "./timeline/ReplyBody.tsx"
import { useMediaContent } from "./timeline/content/useMediaContent.tsx"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import SendIcon from "@/icons/send.svg?react"
import "./MessageComposer.css"

interface MessageComposerProps {
	room: RoomStateStore
	scrollToBottomRef: React.RefObject<() => void>
	setReplyToRef: React.RefObject<(evt: EventID | null) => void>
}

interface ComposerState {
	text: string
	media: MediaMessageEventContent | null
	replyTo: EventID | null
	uninited?: boolean
}

const emptyComposer: ComposerState = { text: "", media: null, replyTo: null }
const uninitedComposer: ComposerState = { ...emptyComposer, uninited: true }
const composerReducer = (state: ComposerState, action: Partial<ComposerState>) =>
	({ ...state, ...action, uninited: undefined })

const draftStore = {
	get: (roomID: RoomID): ComposerState | null => {
		const data = localStorage.getItem(`draft-${roomID}`)
		if (!data) {
			return null
		}
		try {
			return JSON.parse(data)
		} catch {
			return null
		}
	},
	set: (roomID: RoomID, data: ComposerState) => localStorage.setItem(`draft-${roomID}`, JSON.stringify(data)),
	clear: (roomID: RoomID)=> localStorage.removeItem(`draft-${roomID}`),
}

const MessageComposer = ({ room, scrollToBottomRef, setReplyToRef }: MessageComposerProps) => {
	const client = use(ClientContext)!
	const [state, setState] = useReducer(composerReducer, uninitedComposer)
	const [loadingMedia, setLoadingMedia] = useState(false)
	const fileInput = useRef<HTMLInputElement>(null)
	const textInput = useRef<HTMLTextAreaElement>(null)
	const textRows = useRef(1)
	const typingSentAt = useRef(0)
	const replyToEvt = useRoomEvent(room, state.replyTo)
	setReplyToRef.current = useCallback((evt: EventID | null) => {
		setState({ replyTo: evt })
	}, [])
	const sendMessage = useCallback((evt: React.FormEvent) => {
		evt.preventDefault()
		if (state.text === "" && !state.media) {
			return
		}
		setState(emptyComposer)
		const mentions: Mentions = {
			user_ids: [],
			room: false,
		}
		if (replyToEvt) {
			mentions.user_ids.push(replyToEvt.sender)
		}
		client.sendMessage({
			room_id: room.roomID,
			base_content: state.media ?? undefined,
			text: state.text,
			reply_to: replyToEvt?.event_id,
			mentions,
		}).catch(err => window.alert("Failed to send message: " + err))
	}, [replyToEvt, state, room, client])
	const onKeyDown = useCallback((evt: React.KeyboardEvent) => {
		if (evt.key === "Enter" && !evt.shiftKey) {
			sendMessage(evt)
		}
	}, [sendMessage])
	const onChange = useCallback((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setState({ text: evt.target.value })
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
	}, [client, room.roomID])
	const doUploadFile = useCallback((file: File | null | undefined) => {
		if (!file) {
			return
		}
		setLoadingMedia(true)
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
					setState({ media: json })
				}
			})
			.catch(err => window.alert("Failed to upload file: " + err))
			.finally(() => setLoadingMedia(false))
	}, [room])
	const onAttachFile = useCallback(
		(evt: React.ChangeEvent<HTMLInputElement>) => doUploadFile(evt.target.files?.[0]),
		[doUploadFile],
	)
	useEffect(() => {
		const listener = (evt: ClipboardEvent) => doUploadFile(evt.clipboardData?.files?.[0])
		document.addEventListener("paste", listener)
		return () => document.removeEventListener("paste", listener)
	}, [doUploadFile])
	// To ensure the cursor jumps to the end, do this in an effect rather than as the initial value of useState
	// To try to avoid the input bar flashing, use useLayoutEffect instead of useEffect
	useLayoutEffect(() => {
		const draft = draftStore.get(room.roomID)
		setState(draft ?? emptyComposer)
		return () => {
			if (typingSentAt.current > 0) {
				typingSentAt.current = 0
				client.rpc.setTyping(room.roomID, 0)
					.catch(err => console.error("Failed to send stop typing notification due to room switch:", err))
			}
		}
	}, [client, room.roomID])
	useLayoutEffect(() => {
		if (!textInput.current) {
			return
		}
		// This is a hacky way to auto-resize the text area. Setting the rows to 1 and then
		// checking scrollHeight seems to be the only reliable way to get the size of the text.
		textInput.current.rows = 1
		const newTextRows = (textInput.current.scrollHeight - 16) / 20
		textInput.current.rows = newTextRows
		textRows.current = newTextRows
		// This has to be called unconditionally, because setting rows = 1 messes up the scroll state otherwise
		scrollToBottomRef.current?.()
	}, [state, scrollToBottomRef])
	useEffect(() => {
		if (state.uninited) {
			return
		}
		if (!state.text && !state.media && !state.replyTo) {
			draftStore.clear(room.roomID)
		} else {
			draftStore.set(room.roomID, state)
		}
	}, [room, state])
	const openFilePicker = useCallback(() => fileInput.current!.click(), [])
	const clearMedia = useCallback(() => setState({ media: null }), [])
	const closeReply = useCallback(() => setState({ replyTo: null }), [])
	return <div className="message-composer">
		{replyToEvt && <ReplyBody room={room} event={replyToEvt} onClose={closeReply}/>}
		{loadingMedia && <div className="composer-media"><ScaleLoader/></div>}
		{state.media && <ComposerMedia content={state.media} clearMedia={clearMedia}/>}
		<div className="input-area">
			<textarea
				autoFocus
				ref={textInput}
				rows={textRows.current}
				value={state.text}
				onKeyDown={onKeyDown}
				onChange={onChange}
				placeholder="Send a message"
				id="message-composer"
			/>
			<button
				onClick={openFilePicker}
				disabled={!!state.media || loadingMedia}
				title={state.media ? "You can only attach one file at a time" : ""}
			><AttachIcon/></button>
			<button
				onClick={sendMessage}
				disabled={(!state.text && !state.media) || loadingMedia}
			><SendIcon/></button>
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
