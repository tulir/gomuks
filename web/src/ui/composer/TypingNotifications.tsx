// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Sumner Evans
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
import { JSX, use } from "react"
import { PulseLoader } from "react-spinners"
import { getAvatarURL } from "@/api/media.ts"
import { useRoomTyping } from "@/api/statestore"
import { MemberEventContent } from "@/api/types/mxtypes.ts"
import { humanJoinReact } from "@/util/reactjoin.tsx"
import ClientContext from "../ClientContext.ts"
import { useRoomContext } from "../roomview/roomcontext.ts"
import "./TypingNotifications.css"

const TypingNotifications = () => {
	const roomCtx = useRoomContext()
	const client = use(ClientContext)!
	const room = roomCtx.store
	const typing = useRoomTyping(room).filter(u => u !== client.userID)
	let loader: JSX.Element | null = null
	if (typing.length > 0) {
		loader = <PulseLoader speedMultiplier={0.5} size={5} color="var(--primary-color)" />
	}
	const avatars: JSX.Element[] = []
	const memberNames: string[] = []
	for (let i = 0; i < 5 && i < typing.length; i++) {
		const sender = typing[i]
		const memberEvt = room.getStateEvent("m.room.member", sender)
		const member = (memberEvt?.content ?? null) as MemberEventContent | null
		if (!memberEvt) {
			use(ClientContext)?.requestMemberEvent(room, sender)
		}
		avatars.push(<img
			className="small avatar"
			loading="lazy"
			src={getAvatarURL(sender, member)}
			alt=""
		/>)
		memberNames.push(member?.displayname ?? sender)
	}

	let description: JSX.Element | null = null
	if (typing.length > 4) {
		description = <div>{typing.length} users are typing</div>
	} else if (typing.length > 0) {
		description = <div>
			{humanJoinReact(memberNames)}
			{typing.length === 1 ? " is " : " are "}
			typing
		</div>
	}

	return <div className={typing.length ? "typing-notifications" : "typing-notifications empty"}>
		<div className="avatars">{avatars}</div>
		{description}
		{loader}
	</div>
}

export default TypingNotifications
