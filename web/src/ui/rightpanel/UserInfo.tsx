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
import { PuffLoader, ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { getAvatarURL } from "@/api/media.ts"
import { RoomStateStore, useRoomState } from "@/api/statestore"
import { MemberEventContent, ProfileView, UserID } from "@/api/types"
import { getLocalpart } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import { RoomContext } from "../roomview/roomcontext.ts"
import DeviceList from "./UserInfoDeviceList.tsx"
import MutualRooms from "./UserInfoMutualRooms.tsx"
import ErrorIcon from "@/icons/error.svg?react"

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
		setErrors([])
		setView(null)
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
	return <div className="errors">{errors.map((err, i) => <div className="error" key={i}>
		<div className="icon"><ErrorIcon /></div>
		<p>{err}</p>
	</div>)}</div>
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



export default UserInfo
