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
	if (content.membership === prevContent?.membership) {
		if (content.displayname !== prevContent.displayname) {
			if (content.avatar_url !== prevContent.avatar_url) {
				return "changed their displayname and avatar"
			} else if (!content.displayname) {
				return "removed their displayname"
			} else if (!prevContent.displayname) {
				return `set their displayname to ${content.displayname}`
			}
			return `changed their displayname from ${prevContent.displayname} to ${content.displayname}`
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
		return <>invited {content.avatar_url && targetAvatar} {content.displayname ?? target}</>
	} else if (content.membership === "ban") {
		return <>banned {content.avatar_url && targetAvatar} {content.displayname ?? target}</>
	} else if (content.membership === "knock") {
		return "knocked on the room"
	} else if (content.membership === "leave") {
		if (sender === target) {
			return "left the room"
		}
		return <>kicked {content.avatar_url && targetAvatar} {content.displayname ?? target}</>
	}
	return "made an unknown membership change"
}

const MemberBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as MemberEventContent
	const prevContent = event.unsigned.prev_content as MemberEventContent | undefined
	return <div className="member-body">
		{sender?.content.displayname ?? event.sender} {
			useChangeDescription(event.sender, event.state_key as UserID, content, prevContent)
		}
	</div>
}

export default MemberBody
