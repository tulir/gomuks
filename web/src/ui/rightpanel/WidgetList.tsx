// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { use } from "react"
import MainScreenContext from "../MainScreenContext.ts"
import { RoomContext } from "../roomview/roomcontext.ts"

const WidgetList = () => {
	const roomCtx = use(RoomContext)
	const mainScreen = use(MainScreenContext)
	const widgets = roomCtx?.store.state.get("im.vector.modular.widgets") ?? new Map()
	const widgetElements = []
	for (const [stateKey, rowid] of widgets.entries()) {
		const evt = roomCtx?.store.eventsByRowID.get(rowid)
		if (!evt || !evt.content.url) {
			continue
		}
		const onClick = () => mainScreen.setRightPanel({
			type: "widget",
			info: {
				id: stateKey,
				creatorUserId: evt.sender,
				...evt.content,
			} as IWidget,
		})
		widgetElements.push(<button key={rowid} onClick={onClick}>
			{evt.content.name || stateKey}
		</button>)
	}

	const openElementCall = () => {
		mainScreen.setRightPanel({ type: "element-call" })
	}

	return <>
		{widgetElements}
		<div className="separator" />
		<button onClick={openElementCall}>Element Call</button>
	</>
}

export default WidgetList
