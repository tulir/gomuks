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
import { useRoomState } from "@/api/statestore"
import { MemberEventContent, UserID, UserProfile } from "@/api/types"
import { getLocalpart } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import { RoomContext } from "../roomview/roomcontext.ts"
import DeviceList from "./UserInfoDeviceList.tsx"
import UserInfoError from "./UserInfoError.tsx"
import MutualRooms from "./UserInfoMutualRooms.tsx"

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
	const [globalProfile, setGlobalProfile] = useState<UserProfile | null>(null)
	const [errors, setErrors] = useState<string[] | null>(null)
	useEffect(() => {
		setErrors(null)
		setGlobalProfile(null)
		client.rpc.getProfile(userID).then(
			setGlobalProfile,
			err => setErrors([`${err}`]),
		)
	}, [roomCtx, userID, client])

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
		<hr/>
		{userID !== client.userID && <>
			<MutualRooms client={client} userID={userID}/>
			<hr/>
		</>}
		<DeviceList client={client} room={roomCtx?.store} userID={userID}/>
		<hr/>
		{errors?.length ? <>
			<UserInfoError errors={errors}/>
			<hr/>
		</> : null}
	</>
}

export default UserInfo
