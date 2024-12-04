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
import { use, useEffect, useMemo, useReducer, useState } from "react"
import { PuffLoader, ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { getAvatarURL } from "@/api/media.ts"
import { RoomListEntry, RoomStateStore, useRoomState } from "@/api/statestore"
import { MemberEventContent, ProfileView, ProfileViewDevice, RoomID, TrustState, UserID } from "@/api/types"
import { getLocalpart } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import ListEntry from "../roomlist/Entry.tsx"
import { RoomContext } from "../roomview/roomcontext.ts"
import EncryptedOffIcon from "@/icons/encrypted-off.svg?react"
import EncryptedQuestionIcon from "@/icons/encrypted-question.svg?react"
import EncryptedIcon from "@/icons/encrypted.svg?react"

interface UserInfoProps {
	userID: UserID
}

const UserInfo = ({ userID }: UserInfoProps) => {
	const client = use(ClientContext)!
	const roomCtx = use(RoomContext)
	const openLightbox = use(LightboxContext)!
	const memberEvt = useRoomState(roomCtx?.store, "m.room.member", userID)
	const member = (memberEvt?.content ?? null) as MemberEventContent | null
	if (!memberEvt) {
		use(ClientContext)?.requestMemberEvent(roomCtx?.store, userID)
	}
	const [view, setView] = useState<ProfileView | null>(null)
	const [errors, setErrors] = useState<string[]>([])
	useEffect(() => {
		client.rpc.getProfileView(roomCtx?.store.roomID, userID).then(
			resp => {
				setView(resp)
				setErrors(resp.errors)
			},
			err => setErrors([`${err}`]),
		)
	}, [roomCtx, userID, client])

	const displayname = member?.displayname || view?.global_profile?.displayname || getLocalpart(userID)
	return <>
		<div className="avatar-container">
			{member === null && view === null && !errors.length ? <PuffLoader
				color="var(--primary-color)"
				size="100%"
				className="avatar-loader"
			/> : <img
				className="avatar"
				src={getAvatarURL(userID, member ?? view?.global_profile)}
				onClick={openLightbox}
				alt=""
			/>}
		</div>
		<div className="displayname" title={displayname}>{displayname}</div>
		<div className="userid" title={userID}>{userID}</div>
		<hr/>
		{renderFullInfo(client, roomCtx?.store, view, !!errors.length)}
		{renderErrors(errors)}
	</>
}

function renderErrors(errors: string[]) {
	if (!errors.length) {
		return null
	}
	return <div className="error">{errors.map((err, i) => <p key={i}>{err}</p>)}</div>
}

function renderFullInfo(
	client: Client,
	room: RoomStateStore | undefined,
	view: ProfileView | null,
	hasErrors: boolean,
) {
	if (view === null) {
		if (hasErrors) {
			return null
		}
		return <>
			<div className="full-info-loading">
				Loading full profile
				<ScaleLoader color="var(--primary-color)"/>
			</div>
			<hr/>
		</>
	}
	return <>
		{view.mutual_rooms && <MutualRooms client={client} rooms={view.mutual_rooms}/>}
		<DeviceList view={view} room={room}/>
	</>
}

interface DeviceListProps {
	view: ProfileView
	room?: RoomStateStore
}

function trustStateDescription(state: TrustState): string {
	switch (state) {
	case "blacklisted":
		return "Device has been blacklisted manually"
	case "unverified":
		return "Device has not been verified by cross-signing keys, or cross-signing keys were not found"
	case "verified":
		return "Device was verified manually"
	case "cross-signed-untrusted":
		return "Device is cross-signed, cross-signing keys are NOT trusted"
	case "cross-signed-tofu":
		return "Device is cross-signed, cross-signing keys were trusted on first use"
	case "cross-signed-verified":
		return "Device is cross-signed, cross-signing keys were verified manually"
	default:
		return "Invalid trust state"
	}
}

function renderDevice(device: ProfileViewDevice) {
	let Icon = EncryptedIcon
	if (device.trust_state === "blacklisted") {
		Icon = EncryptedOffIcon
	} else if (device.trust_state === "cross-signed-untrusted" || device.trust_state === "unverified") {
		Icon = EncryptedQuestionIcon
	}
	return <li key={device.device_id} className="device">
		<div
			className={`icon-wrapper trust-${device.trust_state}`}
			title={trustStateDescription(device.trust_state)}
		><Icon/></div>
		<div title={device.device_id}>{device.name || device.device_id}</div>
	</li>
}

const DeviceList = ({ view, room }: DeviceListProps) => {
	const isEncrypted = room?.meta.current.encryption_event?.algorithm === "m.megolm.v1.aes-sha2"
	const encryptionMessage = isEncrypted
		? "Messages in this room are end-to-end encrypted."
		: "Messages in this room are not end-to-end encrypted."
	if (!view.devices_tracked) {
		return <div className="devices not-tracked">
			<h4>Security</h4>
			<p>{encryptionMessage}</p>
			<p>This user's device list is not being tracked.</p>
		</div>
	}
	let verifiedMessage = null
	if (view.user_trusted) {
		verifiedMessage = <p className="verified-message verified" title={view.master_key}>
			<EncryptedIcon/> You have verified this user
		</p>
	} else if (view.master_key) {
		if (view.master_key === view.first_master_key) {
			verifiedMessage = <p className="verified-message tofu" title={view.master_key}>
				<EncryptedIcon/> Trusted master key on first use
			</p>
		} else {
			verifiedMessage = <p className="verified-message tofu-broken" title={view.master_key}>
				<EncryptedQuestionIcon/> Master key has changed
			</p>
		}
	}
	return <div className="devices">
		<h4>Security</h4>
		<p>{encryptionMessage}</p>
		{verifiedMessage}
		<h4>{view.devices.length} devices</h4>
		<ul>{view.devices.map(renderDevice)}</ul>
		<hr/>
	</div>
}

interface MutualRoomsProps {
	client: Client
	rooms: RoomID[]
}

const MutualRooms = ({ client, rooms }: MutualRoomsProps) => {
	const [maxCount, increaseMaxCount] = useReducer(count => count + 10, 5)
	const mappedRooms = useMemo(() => rooms.map((roomID): RoomListEntry | null => {
		const roomData = client.store.rooms.get(roomID)
		if (!roomData || roomData.hidden) {
			return null
		}
		return {
			room_id: roomID,
			dm_user_id: roomData.meta.current.lazy_load_summary?.heroes?.length === 1
				? roomData.meta.current.lazy_load_summary.heroes[0] : undefined,
			name: roomData.meta.current.name ?? "Unnamed room",
			avatar: roomData.meta.current.avatar,
			search_name: "",
			sorting_timestamp: 0,
			unread_messages: 0,
			unread_notifications: 0,
			unread_highlights: 0,
			marked_unread: false,
		}
	}).filter((data): data is RoomListEntry => !!data), [client, rooms])
	return <div className="mutual-rooms">
		<h4>Shared rooms</h4>
		{mappedRooms.slice(0, maxCount).map(room => <div key={room.room_id}>
			<ListEntry room={room} isActive={false} hidden={false}/>
		</div>)}
		{mappedRooms.length > maxCount && <button className="show-more" onClick={increaseMaxCount}>
			Show {mappedRooms.length - maxCount} more
		</button>}
		<hr/>
	</div>
}

export default UserInfo
