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
import { Suspense, lazy, use, useCallback, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { getRoomAvatarThumbnailURL, getRoomAvatarURL } from "@/api/media.ts"
import { RoomStateStore, usePreferences } from "@/api/statestore"
import { KeyRestoreProgress, RoomID } from "@/api/types"
import {
	Preference,
	PreferenceContext,
	PreferenceValueType,
	Preferences,
	preferenceContextToInt,
	preferences,
} from "@/api/types/preferences"
import { NonNullCachedEventDispatcher, useEventAsState } from "@/util/eventdispatcher.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext, ModalCloseContext, ModalContext } from "../modal"
import JSONView from "../util/JSONView.tsx"
import Toggle from "../util/Toggle.tsx"
import RoomStateExplorer from "./RoomStateExplorer.tsx"
import CloseIcon from "@/icons/close.svg?react"
import "./SettingsView.css"

interface PreferenceCellProps<T extends PreferenceValueType> {
	context: PreferenceContext
	name: keyof Preferences
	pref: Preference<T>
	setPref: SetPrefFunc
	value: T | undefined
	inheritedValue: T
}

const makeRemover = (
	context: PreferenceContext, setPref: SetPrefFunc, name: keyof Preferences, value: PreferenceValueType | undefined,
) => {
	if (value === undefined) {
		return null
	}
	return <button onClick={() => setPref(context, name, undefined)}><CloseIcon /></button>
}

const BooleanPreferenceCell = ({ context, name, setPref, value, inheritedValue }: PreferenceCellProps<boolean>) => {
	return <div className="preference boolean-preference">
		<Toggle checked={value ?? inheritedValue} onChange={evt => setPref(context, name, evt.target.checked)}/>
		{makeRemover(context, setPref, name, value)}
	</div>
}

const TextPreferenceCell = ({ context, name, setPref, value, inheritedValue }: PreferenceCellProps<string>) => {
	return <div className="preference string-preference">
		<input value={value ?? inheritedValue} onChange={evt => setPref(context, name, evt.target.value)}/>
		{makeRemover(context, setPref, name, value)}
	</div>
}

const SelectPreferenceCell = ({ context, name, pref, setPref, value, inheritedValue }: PreferenceCellProps<string>) => {
	if (!pref.allowedValues) {
		return null
	}
	return <div className="preference select-preference">
		<select value={value ?? inheritedValue} onChange={evt => setPref(context, name, evt.target.value)}>
			{pref.allowedValues.map(value =>
				<option key={value} value={value}>{value}</option>)}
		</select>
		{makeRemover(context, setPref, name, value)}
	</div>
}

type SetPrefFunc = (context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined) => void

interface PreferenceRowProps {
	name: keyof Preferences
	pref: Preference
	setPref: SetPrefFunc
	globalServer?: PreferenceValueType
	globalLocal?: PreferenceValueType
	roomServer?: PreferenceValueType
	roomLocal?: PreferenceValueType
}

const customUIPrefs = new Set([
	"custom_css",
	"custom_notification_sound",
] as (keyof Preferences)[])

const PreferenceRow = ({
	name, pref, setPref, globalServer, globalLocal, roomServer, roomLocal,
}: PreferenceRowProps) => {
	const prefType = typeof pref.defaultValue
	if (customUIPrefs.has(name)) {
		return null
	}
	const makeContentCell = (
		context: PreferenceContext,
		val: PreferenceValueType | undefined,
		inheritedVal: PreferenceValueType,
	) => {
		if (!pref.allowedContexts.includes(context)) {
			return null
		}
		if (prefType === "boolean") {
			return <BooleanPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<boolean>}
				value={val as boolean | undefined}
				inheritedValue={inheritedVal as boolean}
			/>
		} else if (pref.allowedValues) {
			return <SelectPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<string>}
				value={val as string | undefined}
				inheritedValue={inheritedVal as string}
			/>
		} else if (prefType === "string") {
			return <TextPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<string>}
				value={val as string | undefined}
				inheritedValue={inheritedVal as string}
			/>
		} else {
			return null
		}
	}
	let inherit: PreferenceValueType
	return <tr>
		<th title={pref.description}>{pref.displayName}</th>
		<td>{makeContentCell(PreferenceContext.Account, globalServer, inherit = pref.defaultValue)}</td>
		<td>{makeContentCell(PreferenceContext.Device, globalLocal, inherit = globalServer ?? inherit)}</td>
		<td>{makeContentCell(PreferenceContext.RoomAccount, roomServer, inherit = globalLocal ?? inherit)}</td>
		<td>{makeContentCell(PreferenceContext.RoomDevice, roomLocal, inherit = roomServer ?? inherit)}</td>
	</tr>
}

interface SettingsViewProps {
	room: RoomStateStore
}

function getActiveCSSContext(client: Client, room: RoomStateStore): PreferenceContext {
	if (room.localPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.RoomDevice
	} else if (room.serverPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.RoomAccount
	} else if (client.store.localPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.Device
	} else {
		return PreferenceContext.Account
	}
}

const Monaco = lazy(() => import("../util/monaco.tsx"))

const CustomCSSInput = ({ setPref, room }: { setPref: SetPrefFunc, room: RoomStateStore }) => {
	const client = use(ClientContext)!
	const appliedContext = getActiveCSSContext(client, room)
	const [context, setContext] = useState(appliedContext)
	const getContextText = (context: PreferenceContext) => {
		if (context === PreferenceContext.Account) {
			return client.store.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.Device) {
			return client.store.localPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomAccount) {
			return room.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomDevice) {
			return room.localPreferenceCache.custom_css
		}
	}
	const origText = getContextText(context)
	const [text, setText] = useState(origText ?? "")
	const onChangeContext = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		const newContext = evt.target.value as PreferenceContext
		setContext(newContext)
		setText(getContextText(newContext) ?? "")
	}
	const onChangeText = (evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setText(evt.target.value)
	}
	const onSave = useEvent(() => {
		if (vscodeOpen) {
			setText(vscodeContentRef.current)
			setPref(context, "custom_css", vscodeContentRef.current)
		} else {
			setPref(context, "custom_css", text)
		}
	})
	const onDelete = () => {
		setPref(context, "custom_css", undefined)
		setText("")
	}
	const [vscodeOpen, setVSCodeOpen] = useState(false)
	const vscodeContentRef = useRef("")
	const vscodeInitialContentRef = useRef("")
	const onClickVSCode = () => {
		vscodeContentRef.current = text
		vscodeInitialContentRef.current = text
		setVSCodeOpen(true)
	}
	const closeVSCode = useCallback(() => {
		setVSCodeOpen(false)
		setText(vscodeContentRef.current)
		vscodeContentRef.current = ""
	}, [])
	return <div className="custom-css-input">
		<div className="header">
			<h3>Custom CSS</h3>
			<select value={context} onChange={onChangeContext}>
				<option value={PreferenceContext.Account}>Account</option>
				<option value={PreferenceContext.Device}>Device</option>
				<option value={PreferenceContext.RoomAccount}>Room (account)</option>
				<option value={PreferenceContext.RoomDevice}>Room (device)</option>
			</select>
			{preferenceContextToInt(context) < preferenceContextToInt(appliedContext) &&
				<span className="warning">
					&#x26a0;&#xfe0f; This context will not be applied, <code>{appliedContext}</code> has content
				</span>}
		</div>
		{vscodeOpen ? <div className="vscode-wrapper">
			<Suspense fallback={
				<div className="loader"><ScaleLoader width={40} height={80} color="var(--primary-color)"/></div>
			}>
				<Monaco
					initData={vscodeInitialContentRef.current}
					onClose={closeVSCode}
					onSave={onSave}
					contentRef={vscodeContentRef}
				/>
			</Suspense>
		</div> : <textarea value={text} onChange={onChangeText}/>}
		<div className="buttons">
			<button onClick={onClickVSCode}>Open in VS Code</button>
			{origText !== undefined && <button className="delete" onClick={onDelete}>Delete</button>}
			<button className="save primary-color-button" onClick={onSave} disabled={origText === text}>Save</button>
		</div>
	</div>
}

const AppliedSettingsView = ({ room }: SettingsViewProps) => {
	const client = use(ClientContext)!

	return <div className="applied-settings">
		<h3>Raw settings data</h3>
		<details>
			<summary><h4>Applied settings in this room</h4></summary>
			<JSONView data={room.preferences}/>
		</details>
		<details open>
			<summary><h4>Global account settings</h4></summary>
			<JSONView data={client.store.serverPreferenceCache}/>
		</details>
		<details open>
			<summary><h4>Global device settings</h4></summary>
			<JSONView data={client.store.localPreferenceCache}/>
		</details>
		<details open>
			<summary><h4>Room account settings</h4></summary>
			<JSONView data={room.serverPreferenceCache}/>
		</details>
		<details open>
			<summary><h4>Room device settings</h4></summary>
			<JSONView data={room.localPreferenceCache}/>
		</details>
	</div>
}

export interface KeyRestoreStatus {
	progress: KeyRestoreProgress
	connected: boolean
	done?: "ok" | string
}

const KeyRestoreProgressModal = ({ evt }: { evt: NonNullCachedEventDispatcher<KeyRestoreStatus> }) => {
	const status = useEventAsState(evt)
	const prog = status.progress
	let statusMessage: string = "Unknown status"
	let handledCountMessage: string = ""

	const decryptedCount = prog.decrypted + prog.decryption_failed + prog.import_failed
	const statusMax = prog.total * 3 - (prog.decryption_failed * 2) - (prog.import_failed * 2)
	const statusValue = prog.stage === "fetching"
		? undefined
		: decryptedCount + prog.saved + prog.post_processed

	if (prog.stage === "fetching") {
		statusMessage = "Fetching keys from server"
	} else if (prog.stage === "decrypting") {
		statusMessage = "Decrypting keys"
		handledCountMessage = `Decrypted ${prog.decrypted} / ${prog.total} keys`
	} else if (prog.stage === "saving") {
		statusMessage = "Saving decrypted keys"
		handledCountMessage = `Saved ${prog.saved} / ${prog.decrypted} keys`
	} else if (prog.stage === "postprocessing") {
		statusMessage = "Decrypting pending messages"
		handledCountMessage = `Post-processed ${prog.post_processed} / ${prog.decrypted} keys`
	} else if (prog.stage === "done") {
		statusMessage = "Restore completed"
		handledCountMessage = `Successfully restored ${prog.post_processed} / ${prog.total} keys`
	}
	if (status.done && status.done !== "ok") {
		statusMessage = status.done
	} else if (!status.connected) {
		statusMessage = "Connecting to server"
	}
	return <>
		<div className="status">
			{statusMessage}
		</div>
		{prog.current_room_id && !status.done ? <div className="active-room-id">
			Currently processing <code>{prog.current_room_id}</code>
		</div> : null}
		<progress id="key-backup-restore-progress" value={statusValue} max={statusMax}/>

		<label htmlFor="key-backup-restore-progress">
			<div>{handledCountMessage}</div>
			{prog.decryption_failed ? <div>Failed to decrypt {prog.decryption_failed} keys</div> : null}
			{prog.import_failed ? <div>Failed to import {prog.import_failed} keys</div> : null}
		</label>
	</>
}

const KeyExportView = ({ room }: SettingsViewProps) => {
	const [passphrase, setPassphrase] = useState("")
	const [hasFile, setHasFile] = useState(false)
	const openModal = use(ModalContext)
	const importBackup = (roomID?: RoomID) => {
		let path = "_gomuks/keys/restorebackup"
		if (roomID) {
			path += `/${encodeURIComponent(roomID)}`
		}
		const evtSource = new EventSource(path)
		let progress: KeyRestoreProgress = {
			stage: "fetching",
			current_room_id: "",
			decrypted: 0,
			decryption_failed: 0,
			import_failed: 0,
			saved: 0,
			post_processed: 0,
			total: 0,
		}
		let connected = false
		const disp = new NonNullCachedEventDispatcher<KeyRestoreStatus>({
			progress,
			connected,
		})
		evtSource.addEventListener("progress", evt => {
			progress = JSON.parse(evt.data)
			connected = true
			disp.emit({ progress, connected })
		})
		evtSource.addEventListener("done", evt => {
			disp.emit({ progress, connected, done: evt.data })
			evtSource.close()
		})
		evtSource.addEventListener("error", () => {
			disp.emit({ progress, connected, done: "Failed to connect to server" })
			evtSource.close()
		})
		evtSource.addEventListener("close", () => {
			if (!disp.current.done) {
				disp.emit({ progress, connected, done: "Connection closed unexpectedly" })
			}
			evtSource.close()
		})
		openModal({
			dimmed: true,
			boxed: true,
			content: <KeyRestoreProgressModal evt={disp}/>,
			innerBoxClass: "key-restore-modal",
			boxClass: "key-restore-modal-wrapper",
		})
	}
	return <div className="key-export">
		<h3>Key export/import</h3>
		<input
			className="passphrase"
			type="password"
			value={passphrase}
			onChange={evt => setPassphrase(evt.target.value)}
			placeholder="Passphrase"
		/>
		<form
			className="import-buttons"
			action="_gomuks/keys/import"
			encType="multipart/form-data"
			method="post"
			target="_blank"
		>
			<input type="password" name="passphrase" hidden readOnly value={passphrase} />
			<input
				className="import-file"
				type="file"
				accept="text/plain"
				name="export"
				defaultValue=""
				onChange={evt => setHasFile(!!evt.target.files?.length)}
			/>
			<button type="submit" disabled={passphrase == "" || !hasFile}>Import file</button>
		</form>
		<div className="export-buttons">
			<form action="_gomuks/keys/export" method="post" target="_blank">
				<input type="password" name="passphrase" hidden readOnly value={passphrase} />
				<button type="submit" disabled={passphrase == ""}>Export all keys</button>
			</form>
			<form action={`_gomuks/keys/export/${encodeURIComponent(room.roomID)}`} method="post" target="_blank">
				<input type="password" name="passphrase" hidden readOnly value={passphrase} />
				<button type="submit" disabled={passphrase == ""}>Export room keys</button>
			</form>
		</div>
		<hr/>
		<div className="key-backup-buttons">
			<button onClick={() => importBackup(room.roomID)}>Import room backup</button>
			<button onClick={() => importBackup()}>Import entire backup</button>
		</div>
	</div>
}

const SettingsView = ({ room }: SettingsViewProps) => {
	const roomMeta = useEventAsState(room.meta)
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)
	const setPref = useCallback((
		context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined,
	) => {
		if (context === PreferenceContext.Account) {
			client.rpc.setAccountData("fi.mau.gomuks.preferences", {
				...client.store.serverPreferenceCache,
				[key]: value,
			})
		} else if (context === PreferenceContext.Device) {
			if (value === undefined) {
				delete client.store.localPreferenceCache[key]
			} else {
				(client.store.localPreferenceCache[key] as PreferenceValueType) = value
			}
		} else if (context === PreferenceContext.RoomAccount) {
			client.rpc.setAccountData("fi.mau.gomuks.preferences", {
				...room.serverPreferenceCache,
				[key]: value,
			}, room.roomID)
		} else if (context === PreferenceContext.RoomDevice) {
			if (value === undefined) {
				delete room.localPreferenceCache[key]
			} else {
				(room.localPreferenceCache[key] as PreferenceValueType) = value
			}
		}
	}, [client, room])
	const onClickLogout = () => {
		if (window.confirm("Really log out and delete all local data?")) {
			client.logout().then(
				() => console.info("Successfully logged out"),
				err => window.alert(`Failed to log out: ${err}`),
			)
		}
	}
	const onClickLeave = () => {
		if (window.confirm(`Really leave ${room.meta.current.name}?`)) {
			client.rpc.leaveRoom(room.roomID).then(
				() => {
					console.info("Successfully left", room.roomID)
					closeModal()
				},
				err => window.alert(`Failed to leave room: ${err}`),
			)
		}
	}
	const openDevtools = () => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "state-explorer-box",
			content: <RoomStateExplorer room={room} />,
		})
	}
	const onClickOpenCSSApp = () => {
		client.rpc.requestOpenIDToken().then(
			resp => window.open(
				`https://css.gomuks.app/login?token=${resp.access_token}&server_name=${resp.matrix_server_name}`,
				"_blank",
				"noreferrer noopener",
			),
			err => window.alert(`Failed to request OpenID token: ${err}`),
		)
	}
	const previousRoomID = roomMeta.creation_content?.predecessor?.room_id
	const openPredecessorRoom = () => {
		window.mainScreenContext.setActiveRoom(previousRoomID!)
		closeModal()
	}
	usePreferences(client.store, room)
	const globalServer = client.store.serverPreferenceCache
	const globalLocal = client.store.localPreferenceCache
	const roomServer = room.serverPreferenceCache
	const roomLocal = room.localPreferenceCache
	return <>
		<h2>Settings</h2>
		<div className="room-details">
			<img
				className="avatar large"
				loading="lazy"
				src={getRoomAvatarThumbnailURL(roomMeta)}
				data-full-src={getRoomAvatarURL(roomMeta)}
				onClick={use(LightboxContext)}
				alt=""
			/>
			<div>
				{roomMeta.name && <div className="room-name">{roomMeta.name}</div>}
				<code>{room.roomID}</code>
				<div>{roomMeta.topic}</div>
				<div className="room-buttons">
					<button className="leave-room" onClick={onClickLeave}>Leave room</button>
					<button className="devtools" onClick={openDevtools}>Explore room state</button>
					{previousRoomID &&
						<button className="previous-room" onClick={openPredecessorRoom}>
							Open Predecessor Room
						</button>}
				</div>
			</div>
		</div>
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Account</th>
					<th>Device</th>
					<th>Room (account)</th>
					<th>Room (device)</th>
				</tr>
			</thead>
			<tbody>
				{Object.entries(preferences).map(([key, pref]) =>
					<PreferenceRow
						key={key}
						name={key as keyof Preferences}
						pref={pref}
						setPref={setPref}
						globalServer={globalServer[key as keyof Preferences]}
						globalLocal={globalLocal[key as keyof Preferences]}
						roomServer={roomServer[key as keyof Preferences]}
						roomLocal={roomLocal[key as keyof Preferences]}
					/>)}
			</tbody>
		</table>
		<CustomCSSInput setPref={setPref} room={room} />
		<AppliedSettingsView room={room} />
		<hr/>
		<KeyExportView room={room} />
		<hr/>
		<div className="misc-buttons">
			<button onClick={onClickOpenCSSApp}>Sign into css.gomuks.app</button>
			{window.Notification && !window.gomuksAndroid && <button onClick={client.requestNotificationPermission}>
				Request notification permission
			</button>}
			{!window.gomuksAndroid &&
				<button onClick={client.registerURIHandler}>Register <code>matrix:</code> URI handler</button>
			}
			<button className="logout" onClick={onClickLogout}>Logout</button>
		</div>
	</>
}

export default SettingsView
