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
import MainScreenContext from "../MainScreenContext.ts"
import PinnedMessages from "./PinnedMessages.tsx"
import CloseButton from "@/icons/close.svg?react"
import "./RightPanel.css"

export type RightPanelType = "pinned-messages" | "members"

export interface RightPanelProps {
	type: RightPanelType
}

function getTitle(type: RightPanelType): string {
	switch (type) {
	case "pinned-messages":
		return "Pinned Messages"
	case "members":
		return "Room Members"
	}
}

function renderRightPanelContent({ type }: RightPanelProps): JSX.Element | null {
	switch (type) {
	case "pinned-messages":
		return <PinnedMessages />
	case "members":
		return <>Member list is not yet implemented</>
	}
}

const RightPanel = ({ type, ...rest }: RightPanelProps) => {
	return <div className="right-panel">
		<div className="right-panel-header">
			<div className="panel-name">{getTitle(type)}</div>
			<button onClick={use(MainScreenContext).closeRightPanel}><CloseButton/></button>
		</div>
		<div className={`right-panel-content ${type}`}>
			{renderRightPanelContent({ type, ...rest })}
		</div>
	</div>
}

export default RightPanel
