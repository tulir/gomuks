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
import Client from "@/api/client.ts"
import { RoomStateStore, usePreference, useRoomEvent } from "@/api/statestore"
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
import { isMobileDevice } from "@/util/ismobile.ts"
import { escapeMarkdown } from "@/util/markdown.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import EmojiPicker from "../emojipicker/EmojiPicker.tsx"
import GIFPicker from "../emojipicker/GIFPicker.tsx"
import { keyToString } from "../keybindings.ts"
import { LeafletPicker } from "../maps/async.tsx"
import { ModalContext } from "../modal/Modal.tsx"
import { useRoomContext } from "../roomview/roomcontext.ts"
import { ReplyBody } from "../timeline/ReplyBody.tsx"
import { useMediaContent } from "../timeline/content/useMediaContent.tsx"
import type { AutocompleteQuery } from "./Autocompleter.tsx"
import { charToAutocompleteType, emojiQueryRegex, getAutocompleter } from "./getAutocompleter.ts"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import EmojiIcon from "@/icons/emoji-categories/smileys-emotion.svg?react"
import GIFIcon from "@/icons/gif.svg?react"
import LocationIcon from "@/icons/location.svg?react"
import SendIcon from "@/icons/send.svg?react"
import "./MessageComposer.css"

export interface ComposerLocationValue {
	lat: number
	long: number
	prec?: number
}

export interface ComposerState {
	text: string
	media: MediaMessageEventContent | null
	location: ComposerLocationValue | null
	replyTo: EventID | null
	silentReply: boolean
	explicitReplyInThread: boolean
	uninited?: boolean
}

const MAX_TEXTAREA_ROWS = 10

const emptyComposer: ComposerState = {
	text: "",
	media: null,
	replyTo: null,
	location: null,
	silentReply: false,
	explicitReplyInThread: false,
}
const uninitedComposer: ComposerState = { ...emptyComposer, uninited: true }
const composerReducer = (
	state: ComposerState,
	action: Partial<ComposerState> | ((current: ComposerState) => Partial<ComposerState>),
) => ({
	...state,
	...(typeof action === "function" ? action(state) : action),
	uninited: undefined,
})

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
	roomCtx.insertText = useCallback((text: string) => {
		textInput.current?.focus()
		document.execCommand("insertText", false, text)
	}, [])
	roomCtx.setReplyTo = useCallback((evt: EventID | null) => {
		setState({ replyTo: evt, silentReply: false, explicitReplyInThread: false })
		textInput.current?.focus()
	}, [])
	const setSilentReply = useCallback((newVal: boolean | React.MouseEvent) => {
		if (typeof newVal === "boolean") {
			setState({ silentReply: newVal })
		} else {
			newVal.stopPropagation()
			setState(state => ({ silentReply: !state.silentReply }))
		}
	}, [])
	const setExplicitReplyInThread = useCallback((newVal: boolean | React.MouseEvent) => {
		if (typeof newVal === "boolean") {
			setState({ explicitReplyInThread: newVal })
		} else {
			newVal.stopPropagation()
			setState(state => ({ explicitReplyInThread: !state.explicitReplyInThread }))
		}
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
			text: (!evt.content.filename || evt.content.filename !== evt.content.body)
				? (evt.local_content?.edit_source ?? evtContent.body ?? "")
				: "",
			replyTo: null,
			silentReply: false,
			explicitReplyInThread: false,
		})
		textInput.current?.focus()
	}, [room.roomID])
	const sendMessage = useEvent((evt: React.FormEvent) => {
		evt.preventDefault()
		if (state.text === "" && !state.media && !state.location) {
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
			const isThread = replyToEvt.content?.["m.relates_to"]?.rel_type === "m.thread"
				&& typeof replyToEvt.content?.["m.relates_to"]?.event_id === "string"
			if (!state.silentReply && (!isThread || state.explicitReplyInThread)) {
				mentions.user_ids.push(replyToEvt.sender)
			}
			relates_to = {
				"m.in_reply_to": {
					event_id: replyToEvt.event_id,
				},
			}
			if (isThread) {
				relates_to.rel_type = "m.thread"
				relates_to.event_id = replyToEvt.content?.["m.relates_to"].event_id
				relates_to.is_falling_back = !state.explicitReplyInThread
			}
		}
		let base_content: MessageEventContent | undefined
		let extra: Record<string, unknown> | undefined
		if (state.media) {
			base_content = state.media
		} else if (state.location) {
			base_content = {
				body: "Location",
				msgtype: "m.location",
				geo_uri: `geo:${state.location.lat},${state.location.long}`,
			}
			extra = {
				"org.matrix.msc3488.asset": {
					type: "m.pin",
				},
				"org.matrix.msc3488.location": {
					uri: `geo:${state.location.lat},${state.location.long}`,
					description: state.text,
				},
			}
		}
		client.sendMessage({
			room_id: room.roomID,
			base_content,
			extra,
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
	const onComposerKeyDown = useEvent((evt: React.KeyboardEvent<HTMLTextAreaElement>) => {
		const inp = evt.currentTarget
		const fullKey = keyToString(evt)
		if (fullKey === "Enter" && (
			// If the autocomplete already has a selected item or has no results, send message even if it's open.
			// Otherwise, don't send message on enter, select the first autocomplete entry instead.
			!autocomplete
			|| autocomplete.selected !== undefined
			|| !document.getElementById("composer-autocompletions")?.classList.contains("has-items")
		)) {
			sendMessage(evt)
		} else if (autocomplete) {
			let autocompleteUpdate: Partial<AutocompleteQuery> | null | undefined
			if (fullKey === "Tab" || fullKey === "ArrowDown") {
				autocompleteUpdate = { selected: (autocomplete.selected ?? -1) + 1 }
			} else if (fullKey === "Shift+Tab" || fullKey === "ArrowUp") {
				autocompleteUpdate = { selected: (autocomplete.selected ?? 0) - 1 }
			} else if (fullKey === "Enter") {
				autocompleteUpdate = { selected: 0, close: true }
			} else if (fullKey === "Escape") {
				autocompleteUpdate = null
				if (autocomplete.frozenQuery) {
					setState({
						text: state.text.slice(0, autocomplete.startPos)
							+ autocomplete.frozenQuery
							+ state.text.slice(autocomplete.endPos),
					})
				}
			}
			if (autocompleteUpdate !== undefined) {
				setAutocomplete(autocompleteUpdate && { ...autocomplete, ...autocompleteUpdate })
				evt.preventDefault()
			}
		} else if (fullKey === "ArrowUp" && inp.selectionStart === 0 && inp.selectionEnd === 0) {
			const currentlyEditing = editing
				? roomCtx.ownMessages.indexOf(editing.rowid)
				: roomCtx.ownMessages.length
			const prevEventToEditID = roomCtx.ownMessages[currentlyEditing - 1]
			const prevEventToEdit = prevEventToEditID ? room.eventsByRowID.get(prevEventToEditID) : undefined
			if (prevEventToEdit) {
				roomCtx.setEditing(prevEventToEdit)
				evt.preventDefault()
			}
		} else if (editing && fullKey === "ArrowDown" && inp.selectionStart === state.text.length) {
			const currentlyEditingIdx = roomCtx.ownMessages.indexOf(editing.rowid)
			const nextEventToEdit = currentlyEditingIdx
				? room.eventsByRowID.get(roomCtx.ownMessages[currentlyEditingIdx + 1]) : undefined
			roomCtx.setEditing(nextEventToEdit ?? null)
			// This timeout is very hacky and probably doesn't work in every case
			setTimeout(() => inp.setSelectionRange(0, 0), 0)
			evt.preventDefault()
		} else if (editing && fullKey === "Escape") {
			evt.stopPropagation()
			roomCtx.setEditing(null)
		}
	})
	const onChange = useEvent((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setState({ text: evt.target.value })
		const now = Date.now()
		if (evt.target.value !== "" && typingSentAt.current + 5_000 < now) {
			typingSentAt.current = now
			if (room.preferences.send_typing_notifications) {
				client.rpc.setTyping(room.roomID, 10_000)
					.catch(err => console.error("Failed to send typing notification:", err))
			}
		} else if (evt.target.value === "" && typingSentAt.current > 0) {
			typingSentAt.current = 0
			if (room.preferences.send_typing_notifications) {
				client.rpc.setTyping(room.roomID, 0)
					.catch(err => console.error("Failed to send stop typing notification:", err))
			}
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
					setState({ media: json, location: null })
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
			document.execCommand("insertText", false, `[${
				escapeMarkdown(state.text.slice(input.selectionStart, input.selectionEnd))
			}](${escapeMarkdown(text)})`)
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
				if (room.preferences.send_typing_notifications) {
					client.rpc.setTyping(room.roomID, 0)
						.catch(err => console.error("Failed to send stop typing notification due to room switch:", err))
				}
			}
		}
	}, [client, room])
	useLayoutEffect(() => {
		if (!textInput.current) {
			return
		}
		// This is a hacky way to auto-resize the text area. Setting the rows to 1 and then
		// checking scrollHeight seems to be the only reliable way to get the size of the text.
		textInput.current.rows = 1
		const newTextRows = Math.min((textInput.current.scrollHeight - 16) / 20, MAX_TEXTAREA_ROWS)
		if (newTextRows === MAX_TEXTAREA_ROWS) {
			textInput.current.style.overflowY = "auto"
		} else {
			// There's a weird 1px scroll when using line-height, so set overflow to hidden when it's not needed
			textInput.current.style.overflowY = "hidden"
		}
		textInput.current.rows = newTextRows
		textRows.current = newTextRows
		// This has to be called unconditionally, because setting rows = 1 messes up the scroll state otherwise
		roomCtx.scrollToBottom()
		// scrollToBottom needs to be called when replies/attachments/etc change,
		// so listen to state instead of only state.text
	}, [state, roomCtx])
	// Saving to localStorage could be done in the reducer, but that's not very proper, so do it in an effect.
	useEffect(() => {
		roomCtx.isEditing.emit(editing !== null)
		if (state.uninited || editing) {
			return
		}
		if (!state.text && !state.media && !state.replyTo && !state.location) {
			draftStore.clear(room.roomID)
		} else {
			draftStore.set(room.roomID, state)
		}
	}, [roomCtx, room, state, editing])
	const openFilePicker = useCallback(() => fileInput.current!.click(), [])
	const clearMedia = useCallback(() => setState({ media: null, location: null }), [])
	const onChangeLocation = useCallback((location: ComposerLocationValue) => setState({ location }), [])
	const closeReply = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		setState({ replyTo: null })
	}, [])
	const stopEditing = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		roomCtx.setEditing(null)
	}, [roomCtx])
	const openEmojiPicker = useEvent(() => {
		openModal({
			content: <EmojiPicker
				style={{ bottom: (composerRef.current?.clientHeight ?? 32) + 2, right: "1rem" }}
				room={roomCtx.store}
				onSelect={(emoji: PartialEmoji) => setState({
					text: state.text.slice(0, textInput.current?.selectionStart ?? 0)
						+ emojiToMarkdown(emoji)
						+ state.text.slice(textInput.current?.selectionEnd ?? 0),
				})}
			/>,
			onClose: () => textInput.current?.focus(),
		})
	})
	const openGIFPicker = useEvent(() => {
		openModal({
			content: <GIFPicker
				style={{ bottom: (composerRef.current?.clientHeight ?? 32) + 2, right: "1rem" }}
				room={roomCtx.store}
				onSelect={media => setState({ media })}
			/>,
			onClose: () => textInput.current?.focus(),
		})
	})
	const openLocationPicker = useEvent(() => {
		setState({ location: { lat: 0, long: 0, prec: 1 }, media: null })
	})
	const Autocompleter = getAutocompleter(autocomplete, client, room)
	let mediaDisabledTitle: string | undefined
	let locationDisabledTitle: string | undefined
	if (state.media) {
		mediaDisabledTitle = "You can only attach one file at a time"
		locationDisabledTitle = "You can't attach a location to a message with a file"
	} else if (state.location) {
		mediaDisabledTitle = "You can't attach a file to a message with a location"
		locationDisabledTitle = "You can only attach one location at a time"
	} else if (loadingMedia) {
		mediaDisabledTitle = "Uploading file..."
		locationDisabledTitle = "You can't attach a location to a message with a file"
	}
	return <>
		{Autocompleter && autocomplete && <div className="autocompletions-wrapper"><Autocompleter
			params={autocomplete}
			room={room}
			state={state}
			setState={setState}
			setAutocomplete={setAutocomplete}
			textInput={textInput}
		/></div>}
		<div className="message-composer" ref={composerRef}>
			{replyToEvt && <ReplyBody
				room={room}
				event={replyToEvt}
				onClose={closeReply}
				isThread={replyToEvt.content?.["m.relates_to"]?.rel_type === "m.thread"}
				isSilent={state.silentReply}
				onSetSilent={setSilentReply}
				isExplicitInThread={state.explicitReplyInThread}
				onSetExplicitInThread={setExplicitReplyInThread}
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
			{state.location && <ComposerLocation
				room={room} client={client}
				location={state.location} onChange={onChangeLocation} clearLocation={clearMedia}
			/>}
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
				<button onClick={openEmojiPicker} title="Add emoji"><EmojiIcon/></button>
				<button onClick={openGIFPicker} title="Add gif attachment"><GIFIcon/></button>
				<button
					onClick={openLocationPicker}
					disabled={!!locationDisabledTitle}
					title={locationDisabledTitle ?? "Add location"}
				><LocationIcon/></button>
				<button
					onClick={openFilePicker}
					disabled={!!mediaDisabledTitle}
					title={mediaDisabledTitle ?? "Add file attachment"}
				><AttachIcon/></button>
				<button
					onClick={sendMessage}
					disabled={(!state.text && !state.media && !state.location) || loadingMedia}
					title="Send message"
				><SendIcon/></button>
				<input ref={fileInput} onChange={onAttachFile} type="file" value=""/>
			</div>
		</div>
	</>
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

interface ComposerLocationProps {
	room: RoomStateStore
	client: Client
	location: ComposerLocationValue
	onChange: (location: ComposerLocationValue) => void
	clearLocation: () => void
}

const ComposerLocation = ({ client, room, location, onChange, clearLocation }: ComposerLocationProps) => {
	const tileTemplate = usePreference(client.store, room, "leaflet_tile_template")
	return <div className="composer-location">
		<div className="location-container">
			<LeafletPicker tileTemplate={tileTemplate} onChange={onChange} initialLocation={location}/>
		</div>
		<button onClick={clearLocation}><CloseIcon/></button>
	</div>
}

export default MessageComposer
