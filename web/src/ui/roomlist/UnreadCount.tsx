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
import { SpaceUnreadCounts } from "@/api/statestore/space.ts"

interface UnreadCounts extends SpaceUnreadCounts {
	marked_unread?: boolean
}

interface UnreadCountProps {
	counts: UnreadCounts | null
	space?: true
}

const UnreadCount = ({ counts, space }: UnreadCountProps) => {
	if (!counts) {
		return null
	}
	const unreadCount = space
		? counts.unread_highlights || counts.unread_notifications || counts.unread_messages
		: counts.unread_messages || counts.unread_notifications || counts.unread_highlights
	if (!unreadCount && !counts.marked_unread) {
		return null
	}
	const countIsBig = !space
		&& Boolean(counts.unread_notifications || counts.unread_highlights || counts.marked_unread)
	let unreadCountDisplay = unreadCount.toString()
	if (unreadCount > 999 && countIsBig) {
		unreadCountDisplay = "99+"
	} else if (unreadCount > 9999 && countIsBig) {
		unreadCountDisplay = "999+"
	}
	const classNames = ["unread-count"]
	if (countIsBig) {
		classNames.push("big")
	}
	let unreadCountTitle = unreadCount.toString()
	if (space) {
		classNames.push("space")
		unreadCountTitle = [
			counts.unread_highlights && `${counts.unread_highlights} highlights`,
			counts.unread_notifications && `${counts.unread_notifications} notifications`,
			counts.unread_messages && `${counts.unread_messages} messages`,
		].filter(x => !!x).join("\n")
	}
	if (counts.marked_unread) {
		classNames.push("marked-unread")
	}
	if (counts.unread_notifications) {
		classNames.push("notified")
	}
	if (counts.unread_highlights) {
		classNames.push("highlighted")
	}
	return <div className="room-entry-unreads">
		<div title={unreadCountTitle} className={classNames.join(" ")}>
			{unreadCountDisplay}
		</div>
	</div>
}

export default UnreadCount
