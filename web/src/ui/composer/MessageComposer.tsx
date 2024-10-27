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
import type {
	EventID,
	MediaMessageEventContent,
	MemDBEvent,
	Mentions,
	MessageEventContent,
	RelatesTo,
	RoomID,
} from "@/api/types"
import { PartialEmoji, emojiToMarkdown } from "@/util/emoji"
import { escapeMarkdown } from "@/util/markdown.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import EmojiPicker from "../emojipicker/EmojiPicker.tsx"
import { ModalContext } from "../modal/Modal.tsx"
import { useRoomContext } from "../roomcontext.ts"
import { ReplyBody } from "../timeline/ReplyBody.tsx"
import { useMediaContent } from "../timeline/content/useMediaContent.tsx"
import type { AutocompleteQuery } from "./Autocompleter.tsx"
import { charToAutocompleteType, emojiQueryRegex, getAutocompleter } from "./getAutocompleter.ts"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import EmojiIcon from "@/icons/emoji-categories/smileys-emotion.svg?react"
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
	clear: (roomID: RoomID) => localStorage.removeItem(`draft-${roomID}`),
}

type CaretEvent<T> = React.MouseEvent<T> | React.KeyboardEvent<T> | React.ChangeEvent<T>

const MessageComposer = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const client = use(ClientContext)!
	const openModal = use(ModalContext)
	const [autocomplete, setAutocomplete] = useState<AutocompleteQuery | null>(null)
	const [state, setState] = useReducer(composerReducer, uninitedComposer)
	const [editing, rawSetEditing] = useState<MemDBEvent | null>(null)
	const [loadingMedia, setLoadingMedia] = useState(false)
	const fileInput = useRef<HTMLInputElement>(null)
	const textInput = useRef<HTMLTextAreaElement>(null)
	const composerRef = useRef<HTMLDivElement>(null)
	const textRows = useRef(1)
	const typingSentAt = useRef(0)
	const replyToEvt = useRoomEvent(room, state.replyTo)
	roomCtx.setReplyTo = useCallback((evt: EventID | null) => {
		setState({ replyTo: evt })
		textInput.current?.focus()
	}, [])
	roomCtx.setEditing = useCallback((evt: MemDBEvent | null) => {
		if (evt === null) {
			rawSetEditing(null)
			setState(draftStore.get(room.roomID) ?? emptyComposer)
			return
		}
		const evtContent = evt.content as MessageEventContent
		const mediaMsgTypes = ["m.image", "m.audio", "m.video", "m.file"]
		const isMedia = mediaMsgTypes.includes(evtContent.msgtype)
			&& Boolean(evt.content?.url || evt.content?.file?.url)
		rawSetEditing(evt)
		setState({
			media: isMedia ? evtContent as MediaMessageEventContent : null,
			text: (!evt.content.filename || evt.content.filename !== evt.content.body) ? (evtContent.body ?? "") : "",
		})
		textInput.current?.focus()
	}, [room.roomID])
	const sendMessage = useEvent((evt: React.FormEvent) => {
		evt.preventDefault()
		if (state.text === "" && !state.media) {
			return
		}
		if (editing) {
			setState(draftStore.get(room.roomID) ?? emptyComposer)
		} else {
			setState(emptyComposer)
		}
		rawSetEditing(null)
		setAutocomplete(null)
		const mentions: Mentions = {
			user_ids: [],
			room: false,
		}
		let relates_to: RelatesTo | undefined = undefined
		if (editing) {
			relates_to = {
				rel_type: "m.replace",
				event_id: editing.event_id,
			}
		} else if (replyToEvt) {
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
			if (
				acType && (
					area.selectionStart === 1
					|| newText?.[area.selectionStart - 2] === " "
					|| newText?.[area.selectionStart - 2] === "\n"
				)
			) {
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
		} else if (autocomplete && !evt.ctrlKey && !evt.altKey) {
			if (!evt.shiftKey && (evt.key === "Tab" || evt.key === "ArrowDown")) {
				setAutocomplete({ ...autocomplete, selected: (autocomplete.selected ?? -1) + 1 })
				evt.preventDefault()
			} else if ((evt.shiftKey && evt.key === "Tab") || (!evt.shiftKey && evt.key === "ArrowUp")) {
				setAutocomplete({ ...autocomplete, selected: (autocomplete.selected ?? 0) - 1 })
				evt.preventDefault()
			}
		} else if (!autocomplete && textInput.current) {
			const inp = textInput.current
			if (evt.key === "ArrowUp" && inp.selectionStart === 0 && inp.selectionEnd === 0) {
				const currentlyEditing = editing
					? roomCtx.ownMessages.indexOf(editing.rowid)
					: roomCtx.ownMessages.length
				const prevEventToEditID = roomCtx.ownMessages[currentlyEditing - 1]
				const prevEventToEdit = prevEventToEditID ? room.eventsByRowID.get(prevEventToEditID) : undefined
				if (prevEventToEdit) {
					roomCtx.setEditing(prevEventToEdit)
					evt.preventDefault()
				}
			} else if (editing && evt.key === "ArrowDown" && inp.selectionStart === state.text.length) {
				const currentlyEditingIdx = roomCtx.ownMessages.indexOf(editing.rowid)
				const nextEventToEdit = currentlyEditingIdx
					? room.eventsByRowID.get(roomCtx.ownMessages[currentlyEditingIdx + 1]) : undefined
				roomCtx.setEditing(nextEventToEdit ?? null)
				// This timeout is very hacky and probably doesn't work in every case
				setTimeout(() => inp.setSelectionRange(0, 0), 0)
				evt.preventDefault()
			}
		}
		if (editing && evt.key === "Escape") {
			roomCtx.setEditing(null)
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
	const onPaste = useEvent((evt: React.ClipboardEvent<HTMLTextAreaElement>) => {
		const file = evt.clipboardData?.files?.[0]
		const text = evt.clipboardData.getData("text/plain")
		const input = evt.currentTarget
		if (file) {
			doUploadFile(file)
		} else if (
			input.selectionStart !== input.selectionEnd
			&& (text.startsWith("http://") || text.startsWith("https://") || text.startsWith("matrix:"))
		) {
			setState({
				text: `${state.text.slice(0, input.selectionStart)}[${
					escapeMarkdown(state.text.slice(input.selectionStart, input.selectionEnd))
				}](${escapeMarkdown(text)})${state.text.slice(input.selectionEnd)}`,
			})
		} else {
			return
		}
		evt.preventDefault()
	})
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
		roomCtx.isEditing.emit(editing !== null)
		if (state.uninited || editing) {
			return
		}
		if (!state.text && !state.media && !state.replyTo) {
			draftStore.clear(room.roomID)
		} else {
			draftStore.set(room.roomID, state)
		}
	}, [roomCtx, room, state, editing])
	const openFilePicker = useCallback(() => fileInput.current!.click(), [])
	const clearMedia = useCallback(() => setState({ media: null }), [])
	const closeReply = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		setState({ replyTo: null })
	}, [])
	const stopEditing = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		roomCtx.setEditing(null)
	}, [roomCtx])
	const onSelectEmoji = useEvent((emoji: PartialEmoji) => {
		setState({
			text: state.text.slice(0, textInput.current?.selectionStart ?? 0)
				+ emojiToMarkdown(emoji)
				+ state.text.slice(textInput.current?.selectionEnd ?? 0),
		})
	})
	const openEmojiPicker = useEvent(() => {
		openModal({
			content: <EmojiPicker
				style={{ bottom: (composerRef.current?.clientHeight ?? 32) + 2, right: "1rem" }}
				room={roomCtx.store}
				onSelect={onSelectEmoji}
			/>,
			onClose: () => textInput.current?.focus(),
		})
	})
	const Autocompleter = getAutocompleter(autocomplete)
	return <div className="message-composer" ref={composerRef}>
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
		{editing && <ReplyBody
			room={room}
			event={editing}
			isEditing={true}
			isThread={false}
			onClose={stopEditing}
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
				onPaste={onPaste}
				onChange={onChange}
				placeholder="Send a message"
				id="message-composer"
			/>
			<button onClick={openEmojiPicker}><EmojiIcon/></button>
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
