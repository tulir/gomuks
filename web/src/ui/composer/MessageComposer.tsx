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
import { useRoomEvent } from "@/api/statestore"
import type { EventID, MediaMessageEventContent, Mentions, RelatesTo, RoomID } from "@/api/types"
import useEvent from "@/util/useEvent.ts"
import { ClientContext } from "../ClientContext.ts"
import { useRoomContext } from "../roomcontext.ts"
import { ReplyBody } from "../timeline/ReplyBody.tsx"
import { useMediaContent } from "../timeline/content/useMediaContent.tsx"
import type { AutocompleteQuery } from "./Autocompleter.tsx"
import { charToAutocompleteType, emojiQueryRegex, getAutocompleter } from "./getAutocompleter.ts"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import SendIcon from "@/icons/send.svg?react"
import "./MessageComposer.css"

export interface ComposerState {
	text: string
	media: MediaMessageEventContent | null
	replyTo: EventID | null
	uninited?: boolean
}

const isMobileDevice = window.ontouchstart !== undefined && window.innerWidth < 800

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

type CaretEvent<T> = React.MouseEvent<T> | React.KeyboardEvent<T> | React.ChangeEvent<T>

const MessageComposer = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const client = use(ClientContext)!
	const [autocomplete, setAutocomplete] = useState<AutocompleteQuery | null>(null)
	const [state, setState] = useReducer(composerReducer, uninitedComposer)
	const [loadingMedia, setLoadingMedia] = useState(false)
	const fileInput = useRef<HTMLInputElement>(null)
	const textInput = useRef<HTMLTextAreaElement>(null)
	const textRows = useRef(1)
	const typingSentAt = useRef(0)
	const replyToEvt = useRoomEvent(room, state.replyTo)
	roomCtx.setReplyTo = useCallback((evt: EventID | null) => {
		setState({ replyTo: evt })
		textInput.current?.focus()
	}, [])
	const sendMessage = useEvent((evt: React.FormEvent) => {
		evt.preventDefault()
		if (state.text === "" && !state.media) {
			return
		}
		setState(emptyComposer)
		setAutocomplete(null)
		const mentions: Mentions = {
			user_ids: [],
			room: false,
		}
		let relates_to: RelatesTo | undefined = undefined
		if (replyToEvt) {
			mentions.user_ids.push(replyToEvt.sender)
			relates_to = {
				"m.in_reply_to": {
					event_id: replyToEvt.event_id,
				},
			}
			if (replyToEvt.content?.["m.relates_to"]?.rel_type === "m.thread"
				&& typeof replyToEvt.content?.["m.relates_to"]?.event_id === "string") {
				relates_to.rel_type = "m.thread"
				relates_to.event_id = replyToEvt.content?.["m.relates_to"].event_id
				// TODO set this to true if replying to the last event in a thread?
				relates_to.is_falling_back = false
			}
		}
		client.sendMessage({
			room_id: room.roomID,
			base_content: state.media ?? undefined,
			text: state.text,
			relates_to,
			mentions,
		}).catch(err => window.alert("Failed to send message: " + err))
	})
	const onComposerCaretChange = useEvent((evt: CaretEvent<HTMLTextAreaElement>, newText?: string) => {
		const area = evt.currentTarget
		if (area.selectionStart <= (autocomplete?.startPos ?? 0)) {
			if (autocomplete) {
				setAutocomplete(null)
			}
			return
		}
		if (autocomplete?.frozenQuery) {
			if (area.selectionEnd !== autocomplete.endPos) {
				setAutocomplete(null)
			}
		} else if (autocomplete) {
			const newQuery = (newText ?? state.text).slice(autocomplete.startPos, area.selectionEnd)
			if (newQuery.includes(" ") || (autocomplete.type === "emoji" && !emojiQueryRegex.test(newQuery))) {
				setAutocomplete(null)
			} else if (newQuery !== autocomplete.query) {
				setAutocomplete({ ...autocomplete, query: newQuery, endPos: area.selectionEnd })
			}
		} else if (area.selectionStart === area.selectionEnd) {
			const acType = charToAutocompleteType(newText?.slice(area.selectionStart - 1, area.selectionStart))
			if (acType && (area.selectionStart === 1 || newText?.[area.selectionStart - 2] === " ")) {
				setAutocomplete({
					type: acType,
					query: "",
					startPos: area.selectionStart - 1,
					endPos: area.selectionEnd,
				})
			}
		}
	})
	const onComposerKeyDown = useEvent((evt: React.KeyboardEvent) => {
		if (evt.key === "Enter" && !evt.shiftKey) {
			sendMessage(evt)
		}
		if (autocomplete && !evt.ctrlKey && !evt.altKey) {
			if (!evt.shiftKey && (evt.key === "Tab" || evt.key === "ArrowDown")) {
				setAutocomplete({ ...autocomplete, selected: (autocomplete.selected ?? -1) + 1 })
				evt.preventDefault()
			} else if ((evt.shiftKey && evt.key === "Tab") || (!evt.shiftKey && evt.key === "ArrowUp")) {
				setAutocomplete({ ...autocomplete, selected: (autocomplete.selected ?? 0) - 1 })
				evt.preventDefault()
			}
		}
	})
	const onChange = useEvent((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
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
		onComposerCaretChange(evt, evt.target.value)
	})
	const doUploadFile = useCallback((file: File | null | undefined) => {
		if (!file) {
			return
		}
		setLoadingMedia(true)
		const encrypt = !!room.meta.current.encryption_event
		fetch(`_gomuks/upload?encrypt=${encrypt}&filename=${encodeURIComponent(file.name)}`, {
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
	const onAttachFile = useEvent(
		(evt: React.ChangeEvent<HTMLInputElement>) => doUploadFile(evt.target.files?.[0]),
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
		setAutocomplete(null)
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
		roomCtx.scrollToBottom()
	}, [state, roomCtx])
	// Saving to localStorage could be done in the reducer, but that's not very proper, so do it in an effect.
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
	const closeReply = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		setState({ replyTo: null })
	}, [])
	const Autocompleter = getAutocompleter(autocomplete)
	return <div className="message-composer">
		{Autocompleter && autocomplete && <div className="autocompletions-wrapper"><Autocompleter
			params={autocomplete}
			room={room}
			state={state}
			setState={setState}
			setAutocomplete={setAutocomplete}
		/></div>}
		{replyToEvt && <ReplyBody
			room={room}
			event={replyToEvt}
			onClose={closeReply}
			isThread={replyToEvt.content?.["m.relates_to"]?.rel_type === "m.thread"}
		/>}
		{loadingMedia && <div className="composer-media"><ScaleLoader/></div>}
		{state.media && <ComposerMedia content={state.media} clearMedia={clearMedia}/>}
		<div className="input-area">
			<textarea
				autoFocus={!isMobileDevice}
				ref={textInput}
				rows={textRows.current}
				value={state.text}
				onKeyDown={onComposerKeyDown}
				onKeyUp={onComposerCaretChange}
				onClick={onComposerCaretChange}
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
