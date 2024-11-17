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
import { use, useCallback, useState } from "react"
import { RoomStateStore, usePreferences } from "@/api/statestore"
import { Preference, PreferenceContext, PreferenceValueType, Preferences, preferences } from "@/api/types/preferences"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
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

const PreferenceRow = ({
	name, pref, setPref, globalServer, globalLocal, roomServer, roomLocal,
}: PreferenceRowProps) => {
	const prefType = typeof pref.defaultValue
	if (prefType !== "boolean" && !pref.allowedValues) {
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
		} else if (typeof prefType === "string" && pref.allowedValues) {
			return <SelectPreferenceCell
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

const CustomCSSInput = ({ setPref, room }: { setPref: SetPrefFunc, room: RoomStateStore }) => {
	const client = use(ClientContext)!
	const [context, setContext] = useState(PreferenceContext.Account)
	const [text, setText] = useState("")
	const onChangeContext = useCallback((evt: React.ChangeEvent<HTMLSelectElement>) => {
		const newContext = evt.target.value as PreferenceContext
		setContext(newContext)
		if (newContext === PreferenceContext.Account) {
			setText(client.store.serverPreferenceCache.custom_css ?? "")
		} else if (newContext === PreferenceContext.Device) {
			setText(client.store.localPreferenceCache.custom_css ?? "")
		} else if (newContext === PreferenceContext.RoomAccount) {
			setText(room.serverPreferenceCache.custom_css ?? "")
		} else if (newContext === PreferenceContext.RoomDevice) {
			setText(room.localPreferenceCache.custom_css ?? "")
		}
	}, [client, room])
	const onChangeText = useCallback((evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setText(evt.target.value)
	}, [])
	const onSave = useEvent(() => {
		setPref(context, "custom_css", text)
	})
	const onDelete = useEvent(() => {
		setPref(context, "custom_css", undefined)
		setText("")
	})
	return <div className="custom-css-input">
		<div className="header">
			<h3>Custom CSS</h3>
			<select value={context} onChange={onChangeContext}>
				<option value={PreferenceContext.Account}>Account</option>
				<option value={PreferenceContext.Device}>Device</option>
				<option value={PreferenceContext.RoomAccount}>Room (account)</option>
				<option value={PreferenceContext.RoomDevice}>Room (device)</option>
			</select>
		</div>
		<textarea value={text} onChange={onChangeText}/>
		<div className="buttons">
			<button className="delete" onClick={onDelete}>Delete</button>
			<button className="save" onClick={onSave}>Save</button>
		</div>
	</div>
}

const SettingsView = ({ room }: SettingsViewProps) => {
	const client = use(ClientContext)!
	const setPref = useCallback((context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined)=>  {
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
	usePreferences(client.store, room)
	const globalServer = client.store.serverPreferenceCache
	const globalLocal = client.store.localPreferenceCache
	const roomServer = room.serverPreferenceCache
	const roomLocal = room.localPreferenceCache
	return <>
		<h2>Settings</h2>
		<code>{room.roomID}</code>
		<table>
			<thead>
				<tr>
					<th>name</th>
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
		<JSONView data={room.preferences} />
	</>
}

export default SettingsView
