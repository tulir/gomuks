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
import { ScaleLoader } from "react-spinners"
import { getAvatarThumbnailURL, getAvatarURL, getRoomAvatarURL } from "@/api/media.ts"
import { usePreference } from "@/api/statestore/hooks.ts"
import { InvitedRoomStore } from "@/api/statestore/invitedroom.ts"
import { RoomID, RoomSummary } from "@/api/types"
import { getDisplayname, getServerName } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { LightboxContext } from "../modal"
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
	const [knockRequest, setKnockRequest] = useState<string>("")
	const doKnockRoom = () => {
		setButtonClicked(true)
		client.rpc.knockRoom(alias || roomID, alias ? undefined : via, knockRequest || undefined).then(
			() => {
				setButtonClicked(false)
				mainScreen.clearActiveRoom()
				window.alert("Successfully knocked on room")
			},
			err => {
				setError(`Failed to knock: ${err}`)
				setButtonClicked(false)
			},
		)
	}
	const doJoinRoom = () => {
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
	}
	const doRejectInvite = () => {
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
	}
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
	const showInviteAvatars = usePreference(client.store, null, "show_invite_avatars")
	const noAvatarPreview = invite && !showInviteAvatars
	const joinRule = summary?.join_rule ?? invite?.join_rule ?? "invite"
	const allowKnock = ["knock", "knock_restricted"].includes(joinRule) && !invite
	const requiresKnock = joinRule === "knock_restricted" && !invite
		&& (summary?.allowed_room_ids ?? []).findIndex(roomID => client.store.rooms.has(roomID)) !== -1
	const acceptAction = invite ? "Accept" : "Join room"

	return <div className="room-view preview">
		<div className="preview-inner">
			{invite?.invited_by && !invite.dm_user_id ? <div className="inviter-info">
				<img
					className="small avatar"
					onClick={use(LightboxContext)}
					src={getAvatarThumbnailURL(invite.invited_by, invite.inviter_profile, noAvatarPreview)}
					data-full-src={getAvatarURL(invite.invited_by, invite.inviter_profile)}
					alt=""
				/>
				<span className="inviter-name" title={invite.invited_by}>
					{getDisplayname(invite.invited_by, invite.inviter_profile)}
				</span>
				invited you to
			</div> : null}
			<h2 className="room-name">{name}</h2>
			<img
				// this is a big avatar (120px), use full resolution
				src={getRoomAvatarURL(invite ?? summary ?? { room_id: roomID }, undefined, false, noAvatarPreview)}
				data-full-src={getRoomAvatarURL(invite ?? summary ?? { room_id: roomID })}
				className="large avatar"
				onClick={use(LightboxContext)}
				alt=""
			/>
			{loading && <ScaleLoader color="var(--primary-color)"/>}
			{memberCount && <div className="member-count"><GroupIcon/> {memberCount} members</div>}
			<div className="room-topic">{topic}</div>
			{invite && <details className="room-invite-meta">
				<summary>Invite metadata</summary>
				<table>
					<tbody>
						<tr>
							<td>Invited by</td>
							<td>{invite.invited_by}</td>
						</tr>
						<tr>
							<td>Room ID</td>
							<td>{roomID}</td>
						</tr>
						<tr>
							<td>Room alias</td>
							<td>{invite.canonical_alias ?? summary?.canonical_alias}</td>
						</tr>
						<tr>
							<td>Is direct</td>
							<td>{invite.is_direct.toString()}</td>
						</tr>
						<tr>
							<td>Encryption</td>
							<td>
								{invite.encryption ?? summary?.encryption ?? summary?.["im.nheko.summary.encryption"]}
							</td>
						</tr>
						<tr>
							<td>Join rule</td>
							<td>{invite.join_rule ?? summary?.join_rule}</td>
						</tr>
						<tr>
							<td>Timestamp</td>
							<td>{invite.date}</td>
						</tr>
					</tbody>
				</table>
			</details>}
			{invite?.invited_by && <MutualRooms client={client} userID={invite.invited_by}/>}
			{allowKnock && <input
				className="knock-reason"
				onChange={event => setKnockRequest(event.currentTarget.value)}
				placeholder="Why do you want to join this room?"
				value={knockRequest}
			/>}
			<div className="buttons">
				{invite && <button
					disabled={buttonClicked}
					className="reject"
					onClick={doRejectInvite}
				>Reject</button>}
				{!requiresKnock && <button
					disabled={buttonClicked}
					className="primary-color-button"
					onClick={doJoinRoom}
				>{acceptAction}</button>}
				{allowKnock && <button
					disabled={buttonClicked}
					className="primary-color-button"
					onClick={doKnockRoom}
				>Ask to join</button>}
			</div>
			{error && <div className="error">
				<ErrorIcon color="var(--error-color)"/>
				{error}
			</div>}
		</div>
	</div>
}

export default RoomPreview
