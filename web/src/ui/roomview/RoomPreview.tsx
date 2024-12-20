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
import { ScaleLoader } from "react-spinners"
import { getAvatarURL, getRoomAvatarURL } from "@/api/media.ts"
import { InvitedRoomStore } from "@/api/statestore/invitedroom.ts"
import { RoomID, RoomSummary } from "@/api/types"
import { getDisplayname, getServerName } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { LightboxContext } from "../modal/Lightbox.tsx"
import MutualRooms from "../rightpanel/UserInfoMutualRooms.tsx"
import ErrorIcon from "@/icons/error.svg?react"
import GroupIcon from "@/icons/group.svg?react"
import "./RoomPreview.css"

export interface RoomPreviewProps {
	roomID: RoomID
	via?: string[]
	alias?: string
	invite?: InvitedRoomStore
}

const RoomPreview = ({ roomID, via, alias, invite }: RoomPreviewProps) => {
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)
	const [summary, setSummary] = useState<RoomSummary | null>(null)
	const [loading, setLoading] = useState(false)
	const [buttonClicked, setButtonClicked] = useState(false)
	const [error, setError] = useState<string | null>(null)
	const doJoinRoom = useCallback(() => {
		let realVia = via
		if (!via?.length && invite?.invited_by) {
			realVia = [getServerName(invite.invited_by)]
		}
		setButtonClicked(true)
		client.rpc.joinRoom(alias || roomID, alias ? undefined : realVia).then(
			() => console.info("Successfully joined", roomID),
			err => {
				setError(`Failed to join room: ${err}`)
				setButtonClicked(false)
			},
		)
	}, [client, roomID, via, alias, invite])
	const doRejectInvite = useCallback(() => {
		setButtonClicked(true)
		client.rpc.leaveRoom(roomID).then(
			() => {
				console.info("Successfully rejected invite to", roomID)
				mainScreen.clearActiveRoom()
			},
			err => {
				setError(`Failed to reject invite: ${err}`)
				setButtonClicked(false)
			},
		)
	}, [client, mainScreen, roomID])
	useEffect(() => {
		setSummary(null)
		setError(null)
		setLoading(true)
		let realVia = via
		if (!via?.length && invite?.invited_by) {
			realVia = [getServerName(invite.invited_by)]
		}
		client.rpc.getRoomSummary(alias || roomID, realVia).then(
			setSummary,
			err => !invite && setError(`Failed to load room info: ${err}`),
		).finally(() => setLoading(false))
	}, [client, roomID, via, alias, invite])
	const name = summary?.name ?? summary?.canonical_alias ?? invite?.name ?? invite?.canonical_alias ?? alias ?? roomID
	const memberCount = summary?.num_joined_members || null
	const topic = summary?.topic ?? invite?.topic ?? ""
	return <div className="room-view preview">
		<div className="preview-inner">
			{invite?.invited_by && !invite.dm_user_id ? <div className="inviter-info">
				<img
					className="small avatar"
					onClick={use(LightboxContext)}
					src={getAvatarURL(invite.invited_by, invite.inviter_profile)}
					alt=""
				/>
				{getDisplayname(invite.invited_by, invite.inviter_profile)} invited you to
			</div> : null}
			<h2 className="room-name">{name}</h2>
			<img
				src={getRoomAvatarURL(invite ?? summary ?? { room_id: roomID })}
				className="large avatar"
				onClick={use(LightboxContext)}
				alt=""
			/>
			{loading && <ScaleLoader color="var(--primary-color)"/>}
			{memberCount && <div className="member-count"><GroupIcon/> {memberCount} members</div>}
			<div className="room-topic">{topic}</div>
			{invite?.invited_by && <MutualRooms client={client} userID={invite.invited_by} />}
			<div className="buttons">
				{invite && <button
					disabled={buttonClicked}
					className="reject"
					onClick={doRejectInvite}
				>Reject</button>}
				<button
					disabled={buttonClicked}
					className="primary-color-button"
					onClick={doJoinRoom}
				>{invite ? "Accept" : "Join room"}</button>
			</div>
			{error && <div className="error">
				<ErrorIcon color="var(--error-color)"/>
				{error}
			</div>}
		</div>
	</div>
}

export default RoomPreview
