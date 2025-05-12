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
import { use, useCallback, useEffect, useState } from "react"
import { PuffLoader } from "react-spinners"
import { getAvatarURL } from "@/api/media.ts"
import { maybeRedactMemberEvent, useRoomMember } from "@/api/statestore"
import { UserID, UserProfile } from "@/api/types"
import { ensureString, getLocalpart } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext } from "../modal"
import { RoomContext } from "../roomview/roomcontext.ts"
import UserExtendedProfile from "./UserExtendedProfile.tsx"
import DeviceList from "./UserInfoDeviceList.tsx"
import UserInfoError from "./UserInfoError.tsx"
import MutualRooms from "./UserInfoMutualRooms.tsx"
import UserModeration from "./UserModeration.tsx"

interface UserInfoProps {
	userID: UserID
}

const UserInfo = ({ userID }: UserInfoProps) => {
	const client = use(ClientContext)!
	const roomCtx = use(RoomContext)
	const openLightbox = use(LightboxContext)!
	const memberEvt = useRoomMember(client, roomCtx?.store, userID)
	const member = maybeRedactMemberEvent(memberEvt)
	const [globalProfile, setGlobalProfile] = useState<UserProfile | null>(null)
	const [errors, setErrors] = useState<string[] | null>(null)
	const refreshProfile = useCallback((clearState = false) => {
		if (clearState) {
			setErrors(null)
			setGlobalProfile(null)
		}
		client.rpc.getProfile(userID).then(
			setGlobalProfile,
			err => setErrors([`${err}`]),
		)
	}, [userID, client])
	useEffect(() => refreshProfile(true), [refreshProfile])
	const displayname = ensureString(member?.displayname)
		|| ensureString(globalProfile?.displayname)
		|| getLocalpart(userID)
	return <>
		<div className="avatar-container">
			{member === null && globalProfile === null && errors == null ? <PuffLoader
				color="var(--primary-color)"
				size="100%"
				className="avatar-loader"
			/> : <img
				className="avatar"
				// this is a big avatar (236px by default), use full resolution
				src={getAvatarURL(userID, member ?? globalProfile)}
				onClick={openLightbox}
				alt=""
			/>}
		</div>
		<div className="displayname" title={displayname}>{displayname}</div>
		<div className="userid" title={userID}>{userID}</div>
		<UserExtendedProfile profile={globalProfile} refreshProfile={refreshProfile} client={client} userID={userID}/>
		<DeviceList client={client} room={roomCtx?.store} userID={userID}/>
		{userID !== client.userID && <>
			<MutualRooms client={client} userID={userID}/>
			<UserModeration client={client} room={roomCtx?.store} member={memberEvt} userID={userID}/>
		</>}
		<UserInfoError errors={errors}/>
	</>
}

export default UserInfo
