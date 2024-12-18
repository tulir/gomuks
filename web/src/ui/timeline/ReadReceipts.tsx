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
import { use } from "react"
import { getAvatarURL } from "@/api/media.ts"
import { RoomStateStore, useReadReceipts } from "@/api/statestore"
import { EventID, MemberEventContent } from "@/api/types"
import { humanJoin } from "@/util/join.ts"
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import "./ReadReceipts.css"

const ReadReceipts = ({ room, eventID }: { room: RoomStateStore, eventID: EventID }) => {
	const receipts = useReadReceipts(room, eventID)
	if (receipts.length === 0) {
		return null
	}
	// Hacky hack for mobile clients. Would be nicer to get the number based on the CSS variable defining the size
	const maxAvatarCount = window.innerWidth > 720 ? 4 : 2
	const memberEvts = receipts.map(receipt => {
		const memberEvt = room.getStateEvent("m.room.member", receipt.user_id)
		const member = (memberEvt?.content ?? null) as MemberEventContent | null
		if (!memberEvt) {
			use(ClientContext)?.requestMemberEvent(room, receipt.user_id)
		}
		return [receipt.user_id, member] as const
	})
	const avatarMembers = receipts.length > maxAvatarCount ? memberEvts.slice(-maxAvatarCount+1) : memberEvts
	const avatars = avatarMembers.map(([userID, member]) => {
		return <img
			key={userID}
			className="small avatar"
			loading="lazy"
			src={getAvatarURL(userID, member)}
			alt=""
		/>
	})
	const names = memberEvts.map(([userID, member]) => getDisplayname(userID, member))
	return <div className="read-receipts" title={`Read by ${humanJoin(names)}`}>
		{avatars.length < receipts.length && <div className="overflow-count">
			+{receipts.length - avatars.length}
		</div>}
		<div className="avatars">
			{avatars}
		</div>
	</div>
}

export default ReadReceipts
