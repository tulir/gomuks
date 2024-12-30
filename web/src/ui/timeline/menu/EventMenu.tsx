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
import { RoomContextData } from "../../roomview/roomcontext.ts"
import { usePrimaryItems } from "./usePrimaryItems.tsx"
import { useSecondaryItems } from "./useSecondaryItems.tsx"
import CloseIcon from "@/icons/close.svg?react"

interface BaseEventMenuProps {
	evt: MemDBEvent
	roomCtx: RoomContextData
}

interface EventHoverMenuProps extends BaseEventMenuProps {
	setForceOpen: (forceOpen: boolean) => void
}

export const EventHoverMenu = ({ evt, roomCtx, setForceOpen }: EventHoverMenuProps) => {
	const elements = usePrimaryItems(use(ClientContext)!, roomCtx, evt, true, false, undefined, setForceOpen)
	return <div className="event-hover-menu">{elements}</div>
}

interface EventContextMenuProps extends BaseEventMenuProps {
	style: CSSProperties
}

export const EventExtraMenu = ({ evt, roomCtx, style }: EventContextMenuProps) => {
	const elements = useSecondaryItems(use(ClientContext)!, roomCtx, evt)
	return <div style={style} className="event-context-menu extra">{elements}</div>
}

export const EventFullMenu = ({ evt, roomCtx, style }: EventContextMenuProps) => {
	const client = use(ClientContext)!
	const primary = usePrimaryItems(client, roomCtx, evt, false, false, style, undefined)
	const secondary = useSecondaryItems(client, roomCtx, evt)
	return <div style={style} className="event-context-menu full">
		{primary}
		<hr/>
		{secondary}
	</div>
}

export const EventFixedMenu = ({ evt, roomCtx }: BaseEventMenuProps) => {
	const client = use(ClientContext)!
	const primary = usePrimaryItems(client, roomCtx, evt, false, true, undefined, undefined)
	const secondary = useSecondaryItems(client, roomCtx, evt, false)
	return <div className="event-fixed-menu">
		{primary}
		<div className="vertical-line"/>
		{secondary}
		<div className="vertical-line"/>
		<button className="close"><CloseIcon/></button>
	</div>
}
