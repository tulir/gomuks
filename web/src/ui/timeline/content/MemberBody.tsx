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
import { getAvatarThumbnailURL, getAvatarURL } from "@/api/media.ts"
import { MemberEventContent, UserID } from "@/api/types"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import MainScreenContext from "../../MainScreenContext.ts"
import { LightboxContext } from "../../modal"
import EventContentProps from "./props.ts"

function useChangeDescription(
	sender: UserID, target: UserID, content: MemberEventContent, prevContent?: MemberEventContent,
): string | React.ReactElement {
	const openLightbox = use(LightboxContext)!
	const mainScreen = use(MainScreenContext)
	const makeTargetAvatar = () => <img
		className="small avatar"
		loading="lazy"
		src={getAvatarThumbnailURL(target, content)}
		data-full-src={getAvatarURL(target, content)}
		onClick={openLightbox}
		alt=""
	/>
	const makeTargetElem = () => {
		return <>
			<img
				className="small avatar"
				loading="lazy"
				src={getAvatarThumbnailURL(target, content)}
				data-full-src={getAvatarURL(target, content)}
				data-target-panel="user"
				data-target-user={target}
				onClick={mainScreen.clickRightPanelOpener}
				alt=""
			/> <span className="name">
				{getDisplayname(target, content)}
			</span>
		</>
	}
	if (content.membership === prevContent?.membership) {
		if (sender !== target) {
			return <>made no change to {makeTargetElem()}</>
		} else if (content.displayname !== prevContent.displayname) {
			if (content.avatar_url !== prevContent.avatar_url) {
				return <>changed their displayname and avatar</>
			} else if (!content.displayname) {
				return <>removed their displayname</>
			} else if (!prevContent.displayname) {
				return <>set their displayname to <span className="name">{ensureString(content.displayname)}</span></>
			}
			return <>
				changed their displayname from <span className="name">
					{ensureString(prevContent.displayname)}
				</span> to <span className="name">{ensureString(content.displayname)}</span>
			</>
		} else if (content.avatar_url !== prevContent.avatar_url) {
			if (!content.avatar_url) {
				return "removed their avatar"
			} else if (!prevContent.avatar_url) {
				return <>set their avatar to {makeTargetAvatar()}</>
			}
			return <>
				changed their avatar from <img
					className="small avatar"
					loading="lazy"
					height={16}
					src={getAvatarThumbnailURL(target, prevContent)}
					data-full-src={getAvatarURL(target, prevContent)}
					onClick={use(LightboxContext)!}
					alt=""
				/> to {makeTargetAvatar()}
			</>
		}
		return "made no change"
	} else if (content.membership === "join") {
		return "joined the room"
	} else if (content.membership === "invite") {
		if (prevContent?.membership === "knock") {
			return <>accepted {makeTargetElem()}'s join request</>
		}
		return <>invited {makeTargetElem()}</>
	} else if (content.membership === "ban") {
		return <>banned {makeTargetElem()}</>
	} else if (content.membership === "knock") {
		return "requested to join the room"
	} else if (content.membership === "leave") {
		if (sender === target) {
			if (prevContent?.membership === "knock") {
				return "cancelled their join request"
			} else if (prevContent?.membership === "invite") {
				return "rejected the invite"
			}
			return "left the room"
		}
		if (prevContent?.membership === "ban") {
			return <>unbanned {makeTargetElem()}</>
		} else if (prevContent?.membership === "invite") {
			return <>disinvited {makeTargetElem()}</>
		} else if (prevContent?.membership === "knock") {
			return <>rejected {makeTargetElem()}'s join request</>
		}
		return <>kicked {makeTargetElem()}</>
	}
	return "made an unknown membership change"
}

const MemberBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as MemberEventContent
	const prevContent = event.unsigned.prev_content as MemberEventContent | undefined
	return <div className="member-body">
		<span className="name sender-name">
			{getDisplayname(event.sender, sender?.content)}
		</span> <span className="change-description">
			{useChangeDescription(event.sender, event.state_key as UserID, content, prevContent)}
		</span>
		{content.reason ? <span className="reason"> for {ensureString(content.reason)}</span> : null}
	</div>
}

export default MemberBody
