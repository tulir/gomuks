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
import { JSX } from "react"
import type { RoomListFilter } from "@/api/statestore/space.ts"
import HomeIcon from "@/icons/home.svg?react"
import NotificationsIcon from "@/icons/notifications.svg?react"
import PersonIcon from "@/icons/person.svg?react"
import TagIcon from "@/icons/tag.svg?react"
import "./RoomList.css"

export interface FakeSpaceProps {
	space: RoomListFilter | null
	setSpace: (space: RoomListFilter | null) => void
	isActive: boolean
}

const getFakeSpaceIcon = (space: RoomListFilter | null): JSX.Element | null => {
	switch (space?.id) {
	case undefined:
		return <HomeIcon />
	case "fi.mau.gomuks.direct_chats":
		return <PersonIcon />
	case "fi.mau.gomuks.unreads":
		return <NotificationsIcon />
	case "fi.mau.gomuks.space_orphans":
		return <TagIcon />
	default:
		return null
	}
}

const FakeSpace = ({ space, setSpace, isActive }: FakeSpaceProps) => {
	return <div className={`space-entry ${isActive ? "active" : ""}`} onClick={() => setSpace(space)}>
		{getFakeSpaceIcon(space)}
	</div>

}

export default FakeSpace