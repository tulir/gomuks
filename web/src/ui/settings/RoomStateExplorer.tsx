// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { use, useCallback, useState } from "react"
import { RoomStateStore, useRoomState } from "@/api/statestore"
import ClientContext from "../ClientContext.ts"
import JSONView from "../util/JSONView"
import "./RoomStateExplorer.css"

interface StateExplorerProps {
	room: RoomStateStore
}

interface StateEventViewProps {
	room: RoomStateStore
	type?: string
	stateKey?: string
	onBack: () => void
	onDone?: (type: string, stateKey: string) => void
}

interface NewMessageEventViewProps {
	room: RoomStateStore
	onBack: () => void
}

interface StateKeyListProps {
	room: RoomStateStore
	type: string
	onSelectStateKey: (stateKey: string) => void
	onBack: () => void
}

const StateEventView = ({ room, type, stateKey, onBack, onDone }: StateEventViewProps) => {
	const event = useRoomState(room, type, stateKey)
	const isNewEvent = type === undefined
	const [editingContent, setEditingContent] = useState<string | null>(isNewEvent ? "{\n\n}" : null)
	const [newType, setNewType] = useState<string>("")
	const [newStateKey, setNewStateKey] = useState<string>("")
	const client = use(ClientContext)!

	const sendEdit = () => {
		let parsedContent
		try {
			parsedContent = JSON.parse(editingContent || "{}")
		} catch (err) {
			window.alert(`Failed to parse JSON: ${err}`)
			return
		}
		client.rpc.setState(
			room.roomID,
			type ?? newType,
			stateKey ?? newStateKey,
			parsedContent,
		).then(
			() => {
				console.log("Updated room state", room.roomID, type, stateKey)
				setEditingContent(null)
				if (isNewEvent) {
					onDone?.(newType, newStateKey)
				}
			},
			err => {
				console.error("Failed to update room state", err)
				window.alert(`Failed to update room state: ${err}`)
			},
		)
	}
	const stopEdit = () => setEditingContent(null)
	const startEdit = () => setEditingContent(JSON.stringify(event?.content || {}, null, 4))

	return (
		<div className="state-explorer state-event-view">
			<div className="state-header">
				{isNewEvent
					? <>
						<h3>New state event</h3>
						<div className="new-event-type">
							<input
								autoFocus
								type="text"
								value={newType}
								onChange={evt => setNewType(evt.target.value)}
								placeholder="Event type"
							/>
							<input
								type="text"
								value={newStateKey}
								onChange={evt => setNewStateKey(evt.target.value)}
								placeholder="State key"
							/>
						</div>
					</>
					: <h3><code>{type}</code> ({stateKey ? <code>{stateKey}</code> : "no state key"})</h3>
				}
			</div>
			<div className="state-event-content">
				{editingContent !== null
					? <textarea rows={10} value={editingContent} onChange={evt => setEditingContent(evt.target.value)}/>
					: <JSONView data={event}/>
				}
			</div>
			<div className="nav-buttons">
				{editingContent !== null ? <>
					<button onClick={isNewEvent ? onBack : stopEdit}>Back</button>
					<div className="spacer"/>
					<button onClick={sendEdit}>Send</button>
				</> : <>
					<button onClick={onBack}>Back</button>
					<div className="spacer"/>
					<button onClick={startEdit}>Edit</button>
				</>}
			</div>
		</div>
	)
}

const NewMessageEventView = ({ room, onBack }: NewMessageEventViewProps) => {
	const [content, setContent] = useState<string>("{\n\n}")
	const [type, setType] = useState<string>("")
	const [disableEncryption, setDisableEncryption] = useState<boolean>(false)
	const client = use(ClientContext)!

	const sendEvent = () => {
		let parsedContent
		try {
			parsedContent = JSON.parse(content || "{}")
		} catch (err) {
			window.alert(`Failed to parse JSON: ${err}`)
			return
		}
		client.sendEvent(room.roomID, type, parsedContent, disableEncryption).then(
			() => {
				console.log("Successfully sent message event", room.roomID, type)
				onBack()
			},
			err => {
				console.error("Failed to send message event", err)
				window.alert(`Failed to send message event: ${err}`)
			},
		)
	}

	return (
		<div className="state-explorer state-event-view">
			<div className="state-header">
				<h3>New message event</h3>
				<div className="new-event-type">
					<input
						autoFocus
						type="text"
						value={type}
						onChange={evt => setType(evt.target.value)}
						placeholder="Event type"
					/>
				</div>
			</div>
			<div className="state-event-content">
				<textarea rows={10} value={content} onChange={evt => setContent(evt.target.value)}/>
			</div>
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
				<button onClick={sendEvent}>Send</button>
				{room.meta.current.encryption_event ? <label>
					<input
						type="checkbox"
						checked={disableEncryption}
						onChange={evt => setDisableEncryption(evt.target.checked)}
					/>
					Disable encryption
				</label> : null}
			</div>
		</div>
	)
}

const StateKeyList = ({ room, type, onSelectStateKey, onBack }: StateKeyListProps) => {
	const stateMap = room.state.get(type)
	return (
		<div className="state-explorer state-key-list">
			<div className="state-header">
				<h3>State keys under <code>{type}</code></h3>
			</div>
			<div className="state-button-list">
				{Array.from(stateMap?.keys().map(stateKey => (
					<button key={stateKey} onClick={() => onSelectStateKey(stateKey)}>
						{stateKey ? <code>{stateKey}</code> : "<empty>"}
					</button>
				)) ?? [])}
			</div>
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
			</div>
		</div>
	)
}

export const StateExplorer = ({ room }: StateExplorerProps) => {
	const [creatingNew, setCreatingNew] = useState<"message" | "state" | null>(null)
	const [selectedType, setSelectedType] = useState<string | null>(null)
	const [selectedStateKey, setSelectedStateKey] = useState<string | null>(null)
	const [loadingState, setLoadingState] = useState(false)
	const client = use(ClientContext)!

	const handleTypeSelect = (type: string) => {
		const stateKeysMap = room.state.get(type)
		if (!stateKeysMap) {
			return
		}

		const stateKeys = Array.from(stateKeysMap.keys())
		if (stateKeys.length === 1 && stateKeys[0] === "") {
			// If there's only one state event with an empty key, view it directly
			setSelectedType(type)
			setSelectedStateKey("")
		} else {
			// Otherwise show the list of state keys
			setSelectedType(type)
			setSelectedStateKey(null)
		}
	}

	const handleBack = useCallback(() => {
		if (creatingNew) {
			setCreatingNew(null)
		} else if (selectedStateKey !== null && selectedType !== null) {
			setSelectedStateKey(null)
			const stateKeysMap = room.state.get(selectedType)
			if (stateKeysMap?.size === 1 && stateKeysMap.has("")) {
				setSelectedType(null)
			}
		} else if (selectedType !== null) {
			setSelectedType(null)
		}
	}, [selectedType, selectedStateKey, creatingNew, room])
	const handleNewEventDone = useCallback((type: string, stateKey?: string) => {
		setCreatingNew(null)
		if (stateKey !== undefined) {
			setSelectedType(type)
			setSelectedStateKey(stateKey)
		}
	}, [])

	if (creatingNew === "state") {
		return <StateEventView
			room={room}
			onBack={handleBack}
			onDone={handleNewEventDone}
		/>
	} else if (creatingNew === "message") {
		return <NewMessageEventView
			room={room}
			onBack={handleBack}
		/>
	} else if (selectedType !== null && selectedStateKey !== null) {
		return <StateEventView
			room={room}
			type={selectedType}
			stateKey={selectedStateKey}
			onBack={handleBack}
		/>
	} else if (selectedType !== null) {
		return <StateKeyList
			room={room}
			type={selectedType}
			onSelectStateKey={setSelectedStateKey}
			onBack={handleBack}
		/>
	} else {
		const loadRoomState = () => {
			setLoadingState(true)
			client.loadRoomState(room.roomID, {
				omitMembers: false,
				refetch: room.stateLoaded && room.fullMembersLoaded,
			}).then(
				() => {
					console.log("Room state loaded from devtools", room.roomID)
				},
				err => {
					console.error("Failed to fetch room state", err)
					window.alert(`Failed to fetch room state: ${err}`)
				},
			).finally(() => setLoadingState(false))
		}
		return <div className="state-explorer">
			<h3>Room State Explorer</h3>
			<div className="state-button-list">
				{Array.from(room.state?.keys().map(type => (
					<button key={type} onClick={() => handleTypeSelect(type)}>
						<code>{type}</code>
					</button>
				)) ?? [])}
			</div>
			<div className="nav-buttons">
				<button onClick={loadRoomState} disabled={loadingState}>
					{room.stateLoaded
						? room.fullMembersLoaded
							? "Resync full room state"
							: "Load room members"
						: "Load room state and members"}
				</button>
				<div className="spacer"/>
				<button onClick={() => setCreatingNew("message")}>Send new message event</button>
				<button onClick={() => setCreatingNew("state")}>Send new state event</button>
			</div>
		</div>
	}
}

export default StateExplorer
