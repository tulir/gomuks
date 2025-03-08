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
import type { IWidget } from "matrix-widget-api"
import { JSX, use } from "react"
import type { UserID } from "@/api/types"
import MainScreenContext, { MainScreenContextFields } from "../MainScreenContext.ts"
import ErrorBoundary from "../util/ErrorBoundary.tsx"
import ElementCall from "../widget/ElementCall.tsx"
import LazyWidget from "../widget/LazyWidget.tsx"
import MemberList from "./MemberList.tsx"
import PinnedMessages from "./PinnedMessages.tsx"
import UserInfo from "./UserInfo.tsx"
import WidgetList from "./WidgetList.tsx"
import BackIcon from "@/icons/back.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import "./RightPanel.css"

export type RightPanelType = "pinned-messages" | "members" | "widgets" | "widget" | "user" | "element-call"

interface RightPanelSimpleProps {
	type: "pinned-messages" | "members" | "widgets" | "element-call"
}

interface RightPanelWidgetProps {
	type: "widget"
	info: IWidget
}

interface RightPanelUserProps {
	type: "user"
	userID: UserID
}

export type RightPanelProps = RightPanelUserProps | RightPanelWidgetProps | RightPanelSimpleProps

function getTitle(props: RightPanelProps): string {
	switch (props.type) {
	case "pinned-messages":
		return "Pinned Messages"
	case "members":
		return "Room Members"
	case "widgets":
		return "Widgets in room"
	case "widget":
		return props.info.name || "Widget"
	case "element-call":
		return "Element Call"
	case "user":
		return "User Info"
	}
}

function renderRightPanelContent(props: RightPanelProps, mainScreen: MainScreenContextFields): JSX.Element | null {
	switch (props.type) {
	case "pinned-messages":
		return <PinnedMessages />
	case "members":
		return <MemberList />
	case "widgets":
		return <WidgetList />
	case "element-call":
		return <ElementCall onClose={mainScreen.closeRightPanel} />
	case "widget":
		return <LazyWidget info={props.info} onClose={mainScreen.closeRightPanel} />
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
	} else if (props.type === "element-call" || props.type === "widget") {
		backButton = <button
			data-target-panel="widgets"
			onClick={mainScreen.clickRightPanelOpener}
		><BackIcon/></button>
	}
	return <div className="right-panel">
		<div className="right-panel-header">
			<div className="left-side">
				{backButton}
				<div className="panel-name">{getTitle(props)}</div>
			</div>
			<button onClick={mainScreen.closeRightPanel}><CloseIcon/></button>
		</div>
		<div className={`right-panel-content ${props.type}`}>
			<ErrorBoundary thing="right panel content">
				{renderRightPanelContent(props, mainScreen)}
			</ErrorBoundary>
		</div>
	</div>
}

export default RightPanel
