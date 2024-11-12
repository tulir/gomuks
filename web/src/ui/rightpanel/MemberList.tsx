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
import { getAvatarURL } from "@/api/media.ts"
import { useRoomMembers } from "@/api/statestore"
import { MemDBEvent, MemberEventContent } from "@/api/types"
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { RoomContext } from "../roomview/roomcontext.ts"

interface MemberRowProps {
	evt: MemDBEvent
	onClick: (evt: React.MouseEvent<HTMLDivElement>) => void
}

const MemberRow = ({ evt, onClick }: MemberRowProps) => {
	const userID = evt.state_key!
	const content = evt.content as MemberEventContent
	return <div className="member" data-target-panel="user" data-target-user={userID} onClick={onClick}>
		<img
			className="avatar"
			src={getAvatarURL(userID, content)}
			alt=""
			loading="lazy"
		/>
		<span className="displayname">{getDisplayname(userID, content)}</span>
	</div>
}

const MemberList = () => {
	const [limit, setLimit] = useState(50)
	const increaseLimit = useCallback(() => setLimit(limit => limit + 50), [])
	const roomCtx = use(RoomContext)
	if (roomCtx?.store && !roomCtx?.store.membersRequested && !roomCtx?.store.fullMembersLoaded) {
		roomCtx.store.membersRequested = true
		use(ClientContext)?.loadRoomState(roomCtx.store.roomID, { omitMembers: false, refetch: false })
	}
	const memberEvents = useRoomMembers(roomCtx?.store)
	if (!roomCtx) {
		return null
	}
	const mainScreen = use(MainScreenContext)
	const members = []
	for (const evt of memberEvents) {
		members.push(<MemberRow
			key={evt.state_key}
			evt={evt}
			onClick={mainScreen.clickRightPanelOpener}
		/>)
		if (members.length >= limit) {
			break
		}
	}
	return <>
		{members}
		{memberEvents.length > limit ? <button onClick={increaseLimit}>
			and {memberEvents.length - limit} others…
		</button> : null}
	</>
}

export default MemberList
