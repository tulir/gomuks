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
import { getRoomAvatarURL } from "@/api/media.ts"
import { RoomStateStore, usePreferences } from "@/api/statestore"
import {
	Preference,
	PreferenceContext,
	PreferenceValueType,
	Preferences,
	preferenceContextToInt,
	preferences,
} from "@/api/types/preferences"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import { ModalCloseContext } from "../modal/Modal.tsx"
import JSONView from "../util/JSONView.tsx"
import Toggle from "../util/Toggle.tsx"
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

const useRemover = (
	context: PreferenceContext, setPref: SetPrefFunc, name: keyof Preferences, value: PreferenceValueType | undefined,
) => {
	const onClear = useCallback(() => {
		setPref(context, name, undefined)
	}, [setPref, context, name])
	if (value === undefined) {
		return null
	}
	return <button onClick={onClear}><CloseIcon /></button>
}

const BooleanPreferenceCell = ({ context, name, setPref, value, inheritedValue }: PreferenceCellProps<boolean>) => {
	const onChange = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setPref(context, name, evt.target.checked)
	}, [setPref, context, name])
	return <div className="preference boolean-preference">
		<Toggle checked={value ?? inheritedValue} onChange={onChange}/>
		{useRemover(context, setPref, name, value)}
	</div>
}

const TextPreferenceCell = ({ context, name, setPref, value, inheritedValue }: PreferenceCellProps<string>) => {
	const onChange = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setPref(context, name, evt.target.value)
	}, [setPref, context, name])
	return <div className="preference string-preference">
		<input value={value ?? inheritedValue} onChange={onChange}/>
		{useRemover(context, setPref, name, value)}
	</div>
}

const SelectPreferenceCell = ({ context, name, pref, setPref, value, inheritedValue }: PreferenceCellProps<string>) => {
	const onChange = useCallback((evt: React.ChangeEvent<HTMLSelectElement>) => {
		setPref(context, name, evt.target.value)
	}, [setPref, context, name])
	const remover = useRemover(context, setPref, name, value)
	if (!pref.allowedValues) {
		return null
	}
	return <div className="preference select-preference">
		<select value={value ?? inheritedValue} onChange={onChange}>
			{pref.allowedValues.map(value =>
				<option key={value} value={value}>{value}</option>)}
		</select>
		{remover}
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
	const getContextText = useCallback((context: PreferenceContext) => {
		if (context === PreferenceContext.Account) {
			return client.store.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.Device) {
			return client.store.localPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomAccount) {
			return room.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomDevice) {
			return room.localPreferenceCache.custom_css
		}
	}, [client, room])
	const origText = getContextText(context)
	const [text, setText] = useState(origText ?? "")
	const onChangeContext = useCallback((evt: React.ChangeEvent<HTMLSelectElement>) => {
		const newContext = evt.target.value as PreferenceContext
		setContext(newContext)
		setText(getContextText(newContext) ?? "")
	}, [getContextText])
	const onChangeText = useCallback((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setText(evt.target.value)
	}, [])
	const onSave = useEvent(() => {
		if (vscodeOpen) {
			setText(vscodeContentRef.current)
			setPref(context, "custom_css", vscodeContentRef.current)
		} else {
			setPref(context, "custom_css", text)
		}
	})
	const onDelete = useEvent(() => {
		setPref(context, "custom_css", undefined)
		setText("")
	})
	const [vscodeOpen, setVSCodeOpen] = useState(false)
	const vscodeContentRef = useRef("")
	const vscodeInitialContentRef = useRef("")
	const onClickVSCode = useEvent(() => {
		vscodeContentRef.current = text
		vscodeInitialContentRef.current = text
		setVSCodeOpen(true)
	})
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
			<Suspense fallback={<div className="loader"><ScaleLoader width={40} height={80} color="var(--primary-color)"/></div>}>
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

const SettingsView = ({ room }: SettingsViewProps) => {
	const roomMeta = useEventAsState(room.meta)
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const setPref = useCallback((context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined) => {
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
	const onClickLogout = useCallback(() => {
		if (window.confirm("Really log out and delete all local data?")) {
			client.logout().then(
				() => console.info("Successfully logged out"),
				err => window.alert(`Failed to log out: ${err}`),
			)
		}
	}, [client])
	const onClickLeave = useCallback(() => {
		if (window.confirm(`Really leave ${room.meta.current.name}?`)) {
			client.rpc.leaveRoom(room.roomID).then(
				() => {
					console.info("Successfully left", room.roomID)
					closeModal()
				},
				err => window.alert(`Failed to leave room: ${err}`),
			)
		}
	}, [client, room, closeModal])
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
				src={getRoomAvatarURL(roomMeta)}
				onClick={use(LightboxContext)}
				alt=""
			/>
			<div>
				{roomMeta.name && <div className="room-name">{roomMeta.name}</div>}
				<code>{room.roomID}</code>
				<div>{roomMeta.topic}</div>
				<button className="leave-room" onClick={onClickLeave}>Leave room</button>
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
		<div className="misc-buttons">
			{window.Notification && <button onClick={client.requestNotificationPermission}>
				Request notification permission
			</button>}
			<button onClick={client.registerURIHandler}>Register <code>matrix:</code> URI handler</button>
			<button className="logout" onClick={onClickLogout}>Logout</button>
		</div>
	</>
}

export default SettingsView
