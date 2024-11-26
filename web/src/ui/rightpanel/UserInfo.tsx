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
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext, OpenLightboxType } from "../modal/Lightbox.tsx"
import { RoomContext } from "../roomview/roomcontext.ts"

interface UserInfoProps {
	userID: UserID
}

const UserInfo = ({ userID }: UserInfoProps) => {
	const roomCtx = use(RoomContext)
	const openLightbox = use(LightboxContext)!
	const memberEvt = useRoomState(roomCtx?.store, "m.room.member", userID)
	if (!memberEvt) {
		use(ClientContext)?.requestMemberEvent(roomCtx?.store, userID)
	}
	const memberEvtContent = memberEvt?.content as MemberEventContent
	if (!memberEvtContent) {
		return <NonMemberInfo userID={userID}/>
	}
	return renderUserInfo({ userID, profile: memberEvtContent, error: null, openLightbox })
}

const NonMemberInfo = ({ userID }: UserInfoProps) => {
	const openLightbox = use(LightboxContext)!
	const client = use(ClientContext)!
	const [profile, setProfile] = useState<UserProfile | null>(null)
	const [error, setError] = useState<unknown>(null)
	useEffect(() => {
		client.rpc.getProfile(userID).then(setProfile, setError)
	}, [userID, client])
	return renderUserInfo({ userID, profile, error, openLightbox })
}

interface RenderUserInfoParams {
	userID: UserID
	profile: UserProfile | null
	error: unknown
	openLightbox: OpenLightboxType
}

function renderUserInfo({ userID, profile, error, openLightbox }: RenderUserInfoParams) {
	const displayname = getDisplayname(userID, profile)
	return <>
		<div className="avatar-container">
			{profile === null && error === null ? <PuffLoader
				color="var(--primary-color)"
				size="100%"
				className="avatar-loader"
			/> : <img
				className="avatar"
				src={getAvatarURL(userID, profile)}
				onClick={openLightbox}
				alt=""
			/>}
		</div>
		<div className="displayname" title={displayname}>{displayname}</div>
		<div className="userid" title={userID}>{userID}</div>
		{error ? <div className="error">{`${error}`}</div> : null}
	</>
}

export default UserInfo
