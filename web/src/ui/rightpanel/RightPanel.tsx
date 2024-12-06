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
import { JSX, use } from "react"
import type { UserID } from "@/api/types"
import MainScreenContext from "../MainScreenContext.ts"
import MemberList from "./MemberList.tsx"
import PinnedMessages from "./PinnedMessages.tsx"
import UserInfo from "./UserInfo.tsx"
import BackIcon from "@/icons/back.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import "./RightPanel.css"

export type RightPanelType = "pinned-messages" | "members" | "user"

interface RightPanelSimpleProps {
	type: "pinned-messages" | "members"
}

interface RightPanelUserProps {
	type: "user"
	userID: UserID
}

export type RightPanelProps = RightPanelUserProps | RightPanelSimpleProps

function getTitle(type: RightPanelType): string {
	switch (type) {
	case "pinned-messages":
		return "Pinned Messages"
	case "members":
		return "Room Members"
	case "user":
		return "User Info"
	}
}

function renderRightPanelContent(props: RightPanelProps): JSX.Element | null {
	switch (props.type) {
	case "pinned-messages":
		return <PinnedMessages />
	case "members":
		return <MemberList />
	case "user":
		return <UserInfo userID={props.userID} />
	}
}

const RightPanel = (props: RightPanelProps) => {
	const mainScreen = use(MainScreenContext)
	let backButton: JSX.Element | null = null
	if (props.type === "user") {
		backButton = <button
			data-target-panel="members"
			onClick={mainScreen.clickRightPanelOpener}
		><BackIcon/></button>
	}
	return <div className="right-panel">
		<div className="right-panel-header">
			<div className="left-side">
				{backButton}
				<div className="panel-name">{getTitle(props.type)}</div>
			</div>
			<button onClick={mainScreen.closeRightPanel}><CloseIcon/></button>
		</div>
		<div className={`right-panel-content ${props.type}`}>
			{renderRightPanelContent(props)}
		</div>
	</div>
}

export default RightPanel
