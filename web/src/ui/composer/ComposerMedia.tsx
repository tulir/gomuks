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
import Client from "@/api/client.ts"
import { RoomStateStore, usePreference } from "@/api/statestore"
import type { MediaMessageEventContent } from "@/api/types"
import { LeafletPicker } from "../maps/async.tsx"
import { useMediaContent } from "../timeline/content/useMediaContent.tsx"
import CloseIcon from "@/icons/close.svg?react"
import "./MessageComposer.css"

export interface ComposerMediaProps {
	content: MediaMessageEventContent
	clearMedia: false | (() => void)
}

export const ComposerMedia = ({ content, clearMedia }: ComposerMediaProps) => {
	const [mediaContent, containerClass, containerStyle] = useMediaContent(
		content, "m.room.message", { height: 120, width: 360 },
	)
	return <div className="composer-media">
		<div className={`media-container ${containerClass}`} style={containerStyle}>
			{mediaContent}
		</div>
		{clearMedia && <button onClick={clearMedia}><CloseIcon/></button>}
	</div>
}

export interface ComposerLocationValue {
	lat: number
	long: number
	prec?: number
}

export interface ComposerLocationProps {
	room: RoomStateStore
	client: Client
	location: ComposerLocationValue
	onChange: (location: ComposerLocationValue) => void
	clearLocation: () => void
}

export const ComposerLocation = ({ client, room, location, onChange, clearLocation }: ComposerLocationProps) => {
	const tileTemplate = usePreference(client.store, room, "leaflet_tile_template")
	return <div className="composer-location">
		<div className="location-container">
			<LeafletPicker tileTemplate={tileTemplate} onChange={onChange} initialLocation={location}/>
		</div>
		<button onClick={clearLocation}><CloseIcon/></button>
	</div>
}
