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
import { Fragment, use, useState } from "react"
import { CreateRoomInitialState, RoomPreset, RoomVersion, UserID } from "@/api/types"
import { getServerName } from "@/util/validation"
import ClientContext from "../ClientContext"
import MainScreenContext from "../MainScreenContext"
import { ModalCloseContext } from "../modal"
import AddIcon from "@/icons/add.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import InviteIcon from "@/icons/person-add.svg?react"
import "./CreateRoomView.css"

interface initialStateEntry {
	type: string
	stateKey: string
	content: string
}

const CreateRoomView = () => {
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const mainScreen = use(MainScreenContext)
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState("")
	const [preset, setPreset] = useState<RoomPreset>("private_chat")
	const [name, setName] = useState("")
	const [topic, setTopic] = useState("")
	const [aliasLocalpart, setAliasLocalpart] = useState("")
	const [invite, setInvite] = useState<UserID[]>([])
	const [isDirect, setIsDirect] = useState(false)
	const [isEncrypted, setIsEncrypted] = useState(true)
	const [initialState, setInitialState] = useState<initialStateEntry[]>([])
	const [roomVersion, setRoomVersion] = useState<RoomVersion | "">("")
	const [roomID, setRoomID] = useState("")
	const [creationContent, setCreationContent] = useState<string>("{\n\n}")
	const [powerLevelContentOverride, setPowerLevelContentOverride] = useState<string>(() => `{
  "users": {
    ${JSON.stringify(client.store.userID)}: 9001
  }
}`)

	const onSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
		let creation_content, power_level_content_override
		try {
			creation_content = JSON.parse(creationContent)
		} catch (err) {
			setError(`Failed to parse creation content: ${err}`)
			return
		}
		try {
			power_level_content_override = JSON.parse(powerLevelContentOverride)
		} catch (err) {
			setError(`Failed to parse power level content override: ${err}`)
			return
		}
		let reqInitialState: CreateRoomInitialState[]
		try {
			reqInitialState = initialState.filter(state => state.type && state.content).map(state => ({
				type: state.type,
				state_key: state.stateKey,
				content: JSON.parse(state.content),
			}))
		} catch (err) {
			setError(`Failed to parse initial state: ${err}`)
			return
		}
		evt.preventDefault()
		setLoading(true)
		setError("")
		if (isEncrypted) {
			reqInitialState.push({
				type: "m.room.encryption",
				content: {
					algorithm: "m.megolm.v1.aes-sha2",
				},
			})
		}
		client.rpc.createRoom({
			name: name || undefined,
			topic: topic || undefined,
			room_alias_name: aliasLocalpart || undefined,
			preset,
			is_direct: isDirect,
			invite: invite.filter(id => !!id),
			initial_state: reqInitialState,
			room_version: roomVersion || undefined,
			creation_content,
			power_level_content_override,
			"fi.mau.room_id": roomID || undefined,
		}).then(resp => {
			closeModal()
			console.log("Created room:", resp.room_id)

			// FIXME this is a hacky way to work around the room taking time to come down /sync
			setTimeout(() => {
				mainScreen.setActiveRoom(resp.room_id)
			}, 1000)
		}, err => {
			setError(`${err}`.replace(/^Error: /, ""))
			setLoading(false)
		})
	}

	const serverName = getServerName(client.store.userID)

	return <form className="create-room-view" onSubmit={onSubmit}>
		<h2>Create a new room</h2>

		<div className="form-fields">
			<label htmlFor="room-create-name" title="The name of the room">Name</label>
			<input
				id="room-create-name"
				type="text"
				placeholder="Meow room"
				value={name}
				onChange={e => setName(e.target.value)}
			/>
			<label htmlFor="room-create-topic" title="A short description of the room">Topic</label>
			<input
				id="room-create-topic"
				type="text"
				placeholder="A room for meowing"
				value={topic}
				onChange={e => setTopic(e.target.value)}
			/>
			<label htmlFor="room-create-alias" title="The alias for the room">Alias</label>
			<label className="room-alias-container">
				#
				<input
					id="room-create-alias"
					type="text"
					placeholder="meow"
					value={aliasLocalpart}
					onChange={e => setAliasLocalpart(e.target.value)}
				/>
				:{serverName}
			</label>
			<label htmlFor="room-create-encrypted" title="Whether the room is encrypted">
				Encrypted
			</label>
			<input
				id="room-create-encrypted"
				type="checkbox"
				checked={isEncrypted}
				onChange={e => setIsEncrypted(e.target.checked)}
			/>

			<label htmlFor="room-create-preset" title="Preset for join rules and history visibility">Preset</label>
			<select id="room-create-preset" value={preset} onChange={e => setPreset(e.target.value as RoomPreset)}>
				<option value="public_chat">Public chat</option>
				<option value="private_chat">Private chat</option>
				<option value="trusted_private_chat">Trusted private chat</option>
			</select>
		</div>
		<div className="form-fields item-list" id="room-create-invite">
			<div className="item-list-header">
				Users to invite
				<button
					className="item-list-add"
					type="button"
					onClick={() => setInvite([...invite, ""])}
				><InviteIcon /></button>
			</div>
			{invite.map((id, index) => {
				const onChange = (e: React.ChangeEvent<HTMLInputElement>) =>
					setInvite([...invite.slice(0, index), e.target.value, ...invite.slice(index + 1)])
				const onRemove = () => setInvite([...invite.slice(0, index), ...invite.slice(index + 1)])
				return <Fragment key={index}>
					<input
						className="item-list-item"
						type="text"
						placeholder={`@user:${serverName}`}
						value={id}
						onChange={onChange}
					/>
					<button
						className="item-list-remove"
						type="button"
						onClick={onRemove}
					><CloseIcon /></button>
				</Fragment>
			})}
		</div>
		<details>
			<summary>Advanced options</summary>
			<div className="form-fields">
				<label htmlFor="room-create-is-direct" title="Whether the room is a direct chat">
					Direct chat
				</label>
				<input
					id="room-create-is-direct"
					type="checkbox"
					checked={isDirect}
					onChange={e => setIsDirect(e.target.checked)}
				/>
				<label
					htmlFor="room-create-version"
					title="The version of the room to create. If unset, the server will decide"
				>
					Room version
				</label>
				<input
					id="room-create-version"
					type="text"
					placeholder="11"
					value={roomVersion}
					onChange={e => setRoomVersion(e.target.value as RoomVersion)}
				/>
				<label htmlFor="room-create-id" title="Custom room ID. Only works if supported by the server.">
					Room ID
				</label>
				<input
					id="room-create-id"
					type="text"
					placeholder={`!meow:${serverName}`}
					value={roomID}
					onChange={e => setRoomID(e.target.value)}
				/>
				<label htmlFor="room-create-power-level-override" title="Override power levels in the room">
					Power level override
				</label>
				<textarea
					id="room-create-power-level-override"
					value={powerLevelContentOverride}
					onChange={e => setPowerLevelContentOverride(e.target.value)}
					rows={5}
				/>
				<label htmlFor="room-create-creation-content" title="Override the creation content of the room">
					Creation content
				</label>
				<textarea
					id="room-create-creation-content"
					value={creationContent}
					onChange={e => setCreationContent(e.target.value)}
					rows={3}
				/>
			</div>
			<div className="form-fields item-list state-event-list" id="room-create-initial-state">
				<div className="item-list-header">
					Initial state
					<button
						className="item-list-state-add"
						type="button"
						onClick={() => setInitialState([
							...initialState,
							{ type: "", stateKey: "", content: "{\n\n}" },
						])}
					><AddIcon /></button>
				</div>
				{initialState.map((state, index) => {
					const onChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setInitialState([
						...initialState.slice(0, index),
						{
							...initialState[index],
							[e.target.dataset.field!]: e.target.value,
						},
						...initialState.slice(index + 1),
					])
					const onRemove = () => setInitialState([
						...initialState.slice(0, index),
						...initialState.slice(index + 1),
					])
					return <Fragment key={index}>
						<div className="item-list-item state-event-form">
							<input
								className="state-event-type"
								type="text"
								data-field="type"
								placeholder="Event type"
								value={state.type}
								onChange={onChange}
							/>
							<input
								className="state-event-key"
								type="text"
								data-field="stateKey"
								placeholder="State key"
								value={state.stateKey}
								onChange={onChange}
							/>
							<textarea
								className="state-event-content"
								data-field="content"
								placeholder="Event content"
								value={state.content}
								onChange={onChange}
								rows={3}
							/>
						</div>
						<button
							className="item-list-remove"
							type="button"
							onClick={onRemove}
						><CloseIcon /></button>
					</Fragment>
				})}
			</div>
		</details>

		<button type="submit" disabled={loading}>Create</button>
		{error && <div className="error">{error}</div>}
	</form>
}

export default CreateRoomView
