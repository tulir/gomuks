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
import { CSSProperties, use } from "react"
import { MemDBEvent } from "@/api/types"
import ClientContext from "../../ClientContext.ts"
import { RoomContextData, useRoomContext } from "../../roomview/roomcontext.ts"
import { usePrimaryItems } from "./usePrimaryItems.tsx"
import { useSecondaryItems } from "./useSecondaryItems.tsx"

interface EventHoverMenuProps {
	evt: MemDBEvent
	setForceOpen: (forceOpen: boolean) => void
}

export const EventHoverMenu = ({ evt, setForceOpen }: EventHoverMenuProps) => {
	const elements = usePrimaryItems(use(ClientContext)!, useRoomContext(), evt, true, undefined, setForceOpen)
	return <div className="event-hover-menu">{elements}</div>
}

interface EventContextMenuProps {
	evt: MemDBEvent
	roomCtx: RoomContextData
	style: CSSProperties
}

export const EventExtraMenu = ({ evt, roomCtx, style }: EventContextMenuProps) => {
	const elements = useSecondaryItems(use(ClientContext)!, roomCtx, evt)
	return <div style={style} className="event-context-menu extra">{elements}</div>
}

export const EventFullMenu = ({ evt, roomCtx, style }: EventContextMenuProps) => {
	const client = use(ClientContext)!
	const primary = usePrimaryItems(client, roomCtx, evt, false, style, undefined)
	const secondary = useSecondaryItems(client, roomCtx, evt)
	return <div style={style} className="event-context-menu full">
		{primary}
		<hr/>
		{secondary}
	</div>
}
