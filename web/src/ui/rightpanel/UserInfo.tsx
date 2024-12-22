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
import { use, useEffect, useState } from "react"
import { PuffLoader } from "react-spinners"
import { getAvatarURL } from "@/api/media.ts"
import { useRoomMember } from "@/api/statestore"
import { MemberEventContent, UserID, UserProfile, Presence } from "@/api/types"
import { getLocalpart } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import { RoomContext } from "../roomview/roomcontext.ts"
import DeviceList from "./UserInfoDeviceList.tsx"
import UserInfoError from "./UserInfoError.tsx"
import MutualRooms from "./UserInfoMutualRooms.tsx"
import { ErrorResponse } from "@/api/rpc.ts"

interface UserInfoProps {
	userID: UserID
}

const PresenceEmojis = {
	online: "🟢",
	offline: "⚫",
	unavailable: "🔴",
}

const UserInfo = ({ userID }: UserInfoProps) => {
	const client = use(ClientContext)!
	const roomCtx = use(RoomContext)
	const openLightbox = use(LightboxContext)!
	const memberEvt = useRoomMember(client, roomCtx?.store, userID)
	const member = (memberEvt?.content ?? null) as MemberEventContent | null
	const [globalProfile, setGlobalProfile] = useState<UserProfile | null>(null)
	const [presence, setPresence] = useState<Presence | null>(null)
	const [errors, setErrors] = useState<string[] | null>(null)
	useEffect(() => {
		setErrors(null)
		setGlobalProfile(null)
		client.rpc.getProfile(userID).then(
			setGlobalProfile,
			err => setErrors([`${err}`]),
		)
		client.rpc.getPresence(userID).then(
			setPresence,
			err => {
				// A 404 is to be expected if the user has not federated presence.
				if (err instanceof ErrorResponse && err.message.startsWith("M_NOT_FOUND")) {
					setPresence(null)
				} else {
					if(errors) {setErrors([...errors, `${err}`])}
					else {setErrors([`${err}`])}
				}
			}
		)
	}, [roomCtx, userID, client])

	const sendNewPresence = (newPresence: Presence) => {
		console.log("Setting new presence", newPresence)
		client.rpc.setPresence(newPresence).then(
			() => setPresence(newPresence),
			err => setErrors((errors && [...errors, `${err}`] || [`${err}`])),
		)
	}

	const displayname = member?.displayname || globalProfile?.displayname || getLocalpart(userID)
	return <>
		<div className="avatar-container">
			{member === null && globalProfile === null && errors == null ? <PuffLoader
				color="var(--primary-color)"
				size="100%"
				className="avatar-loader"
			/> : <img
				className="avatar"
				src={getAvatarURL(userID, member ?? globalProfile)}
				onClick={openLightbox}
				alt=""
			/>}
		</div>
		<div className="displayname" title={displayname}>{displayname}</div>
		<div className="userid" title={userID}>{userID}</div>
		{presence && (
			<>
			<div className="presence" title={presence.presence}>{PresenceEmojis[presence.presence]} {presence.presence}</div>
			{
				presence.status_msg && (
					<div className="statusmessage" title={"Status message"}><blockquote>{presence.status_msg}</blockquote></div>
				)
			}
			</>
			)
		}
		<hr/>
		{userID !== client.userID && <>
			<MutualRooms client={client} userID={userID}/>
			<hr/>
		</>}
		<DeviceList client={client} room={roomCtx?.store} userID={userID}/>
		<hr/>
		{userID === client.userID && <>
			<h3>Set presence</h3>
			<div className="presencesetter">
				<button title="Set presence to online" onClick={() => sendNewPresence({...(presence || {}), "presence": "online"})} type="button">{PresenceEmojis["online"]}</button>
				<button title="Set presence to unavailable" onClick={() => sendNewPresence({...(presence || {}), "presence": "unavailable"})} type="button">{PresenceEmojis["unavailable"]}</button>
				<button title="Set presence to offline" onClick={() => sendNewPresence({...(presence || {}), "presence": "offline"})} type="button">{PresenceEmojis["offline"]}</button>
			</div>
			<div className="statussetter">
				<form onSubmit={(e) => {e.preventDefault(); sendNewPresence({...(presence || {"presence": "offline"}), "status_msg": ((e.target as HTMLFormElement).children[0] as HTMLInputElement).value})}}>
					<input type="text" placeholder="Status message" defaultValue={presence?.status_msg || ""}/><button title="Set status message" onClick={() => alert("Set status message")} type="submit">Set</button>
				</form>
			</div>
			<hr/>
			</>}
		{errors?.length ? <>
			<UserInfoError errors={errors}/>
			<hr/>
		</> : null}
	</>
}

export default UserInfo
