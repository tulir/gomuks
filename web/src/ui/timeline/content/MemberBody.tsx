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
import React, { use } from "react"
import { getAvatarURL } from "@/api/media.ts"
import { MemberEventContent, UserID } from "@/api/types"
import { LightboxContext } from "../../modal/Lightbox.tsx"
import EventContentProps from "./props.ts"

function useChangeDescription(
	sender: UserID, target: UserID, content: MemberEventContent, prevContent?: MemberEventContent,
): string | React.ReactElement {
	const targetAvatar = <img
		className="small avatar"
		loading="lazy"
		src={getAvatarURL(target, content)}
		onClick={use(LightboxContext)!}
		alt=""
	/>
	const targetElem = <>
		{content.avatar_url && targetAvatar} <span className="name">
			{content.displayname ?? target}
		</span>
	</>
	if (content.membership === prevContent?.membership) {
		if (content.displayname !== prevContent.displayname) {
			if (content.avatar_url !== prevContent.avatar_url) {
				return <>changed their displayname and avatar</>
			} else if (!content.displayname) {
				return <>removed their displayname</>
			} else if (!prevContent.displayname) {
				return <>set their displayname to <span className="name">{content.displayname}</span></>
			}
			return <>
				changed their displayname from <span className="name">
					{prevContent.displayname}
				</span> to <span className="name">{content.displayname}</span>
			</>
		} else if (content.avatar_url !== prevContent.avatar_url) {
			if (!content.avatar_url) {
				return "removed their avatar"
			} else if (!prevContent.avatar_url) {
				return <>set their avatar to {targetAvatar}</>
			}
			return <>
				changed their avatar from <img
					className="small avatar"
					loading="lazy"
					height={16}
					src={getAvatarURL(target, prevContent)}
					onClick={use(LightboxContext)!}
					alt=""
				/> to {targetAvatar}
			</>
		}
		return "made no change"
	} else if (content.membership === "join") {
		return "joined the room"
	} else if (content.membership === "invite") {
		return <>invited {targetElem}</>
	} else if (content.membership === "ban") {
		return <>banned {targetElem}</>
	} else if (content.membership === "knock") {
		return "knocked on the room"
	} else if (content.membership === "leave") {
		if (sender === target) {
			if (prevContent?.membership === "knock") {
				return "cancelled their knock"
			}
			return "left the room"
		}
		if (prevContent?.membership === "ban") {
			return <>unbanned {targetElem}</>
		} else if (prevContent?.membership === "invite") {
			return <>disinvited {targetElem}</>
		}
		return <>kicked {targetElem}</>
	}
	return "made an unknown membership change"
}

const MemberBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as MemberEventContent
	const prevContent = event.unsigned.prev_content as MemberEventContent | undefined
	return <div className="member-body">
		<span className="name sender-name">
			{sender?.content.displayname ?? event.sender}
		</span> <span className="change-description">
			{useChangeDescription(event.sender, event.state_key as UserID, content, prevContent)}
		</span>
		{content.reason ? <span className="reason"> for {content.reason}</span> : null}
	</div>
}

export default MemberBody
