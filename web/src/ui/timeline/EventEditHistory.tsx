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
import { use, useEffect, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { MemDBEvent } from "@/api/types"
import ClientContext from "../ClientContext.ts"
import { RoomContext, RoomContextData } from "../roomview/roomcontext.ts"
import TimelineEvent from "./TimelineEvent.tsx"
import "./EventEditHistory.css"

interface EventEditHistoryProps {
	evt: MemDBEvent
	roomCtx: RoomContextData
}

const EventEditHistory = ({ evt, roomCtx }: EventEditHistoryProps) => {
	const client = use(ClientContext)!
	const [revisions, setRevisions] = useState<MemDBEvent[]>([])
	const [error, setError] = useState("")
	const [loading, setLoading] = useState(true)
	useEffect(() => {
		setLoading(true)
		setError("")
		setRevisions([])
		client.getRelatedEvents(roomCtx.store, evt.event_id, "m.replace").then(
			edits => {
				setRevisions([{
					...evt,
					content: evt.orig_content ?? evt.content,
					local_content: evt.orig_local_content ?? evt.local_content,
					last_edit: undefined,
					reactions: undefined,
					orig_content: undefined,
					orig_local_content: undefined,
				}, ...edits.map(editEvt => ({
					...editEvt,
					content: editEvt.content["m.new_content"] ?? editEvt.content,
					orig_content: editEvt.content,
					relation_type: undefined,
					reactions: undefined,
				}))])
			},
			err => {
				console.error("Failed to get event edit history", err)
				setError(`${err}`)
			},
		).finally(() => setLoading(false))
	}, [client, roomCtx, evt])

	if (loading) {
		return <ScaleLoader color="var(--primary-color)"/>
	} else if (error) {
		return <div>Failed to load :( {error}</div>
	}
	return <>
		<RoomContext value={roomCtx}>
			<p>Event has {revisions.length} revisions</p>
			{revisions.map((rev, i) => <TimelineEvent
				key={rev.rowid}
				evt={rev}
				prevEvt={revisions[i-1] ?? null}
				editHistoryView={true}
			/>)}
		</RoomContext>
	</>
}

export default EventEditHistory
