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
import { MouseEvent, useEffect, useState } from "react"
import { UserID, Presence, RPCEvent } from "@/api/types"
import { ErrorResponse } from "@/api/rpc.ts"
import Client from "@/api/client"

interface UserPresenceProps {
    client: Client,
    userID: UserID
}
interface EditUserPresenceProps {
    client: Client,
    presence: Presence,
    setter: (presence: Presence) => void
}
const PresenceEmojis = {
    "online": <svg className="presence-indicator presence-online" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg"><circle cx="16" cy="16" r="50" /></svg>,
    "offline": <svg className="presence-indicator presence-offline" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg"><circle cx="16" cy="16" r="50" /></svg>,
    "unavailable": <svg className="presence-indicator presence-unavailable" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg"><circle cx="16" cy="16" r="50" /></svg>,
}

export const UserPresence = ({ client, userID }: UserPresenceProps) => {
    const [presence, setPresence] = useState<Presence | null>(null)
    const [errors, setErrors] = useState<string[] | null>(null)

    client.rpc.event.listen((event: RPCEvent) => {
        if (event.command === "update_presence" && event.data.user_id === userID) {
            setPresence({
                "presence": event.data.presence,
                "status_msg": event.data.status_msg || null
            })
        }
    })

    useEffect(() => {
        client.rpc.getPresence(userID).then(
			setPresence,
			err => {
				// A 404 is to be expected if the user has not federated presence.
				if (err instanceof ErrorResponse && err.message.startsWith("M_NOT_FOUND")) {
					setPresence(null)
				} else {
					errors?.length ? setErrors([...errors, `${err}`]) : setErrors([`${err}`])
				}
			}
		)
    }, [client, userID])

    if(!presence) return null;

    return (
        <>
        <DisplayUserPresence presence={presence} />
        {
            userID === client.userID && <EditUserPresence client={client} presence={presence} setter={setPresence}/>
        }
        </>
    )
}

export const DisplayUserPresence = ({ presence }: { presence: Presence | null }) => {
    if(!presence) return null
    return (
        <>
        <div className="presence" title={presence.presence}>{PresenceEmojis[presence.presence]} {presence.presence}</div>
        {
            presence.status_msg && (
                <div className="statusmessage" title={"Status message"}><blockquote>{presence.status_msg}</blockquote></div>
            )
        }
        </>
    )
}

export const EditUserPresence = ({ client, presence, setter }: EditUserPresenceProps) => {
    const sendNewPresence = (newPresence: Presence) => {
		client.rpc.setPresence(newPresence).then(
			() => setter(newPresence),
			err => console.error(err),
		)
	}
    const clearStatusMessage = (e: MouseEvent<HTMLButtonElement>) => {
        let p = presence || {"presence": "offline"}
        if(p.status_msg) {
            delete p.status_msg
        }
        const textInputElement = e.currentTarget.parentElement?.querySelector("input[type=text]") as HTMLInputElement
        client.rpc.setPresence(p).then(
			() => {setter(p); if(textInputElement) textInputElement.value = ""},
			err => console.error(err),
		)
    }
    return (
        <>
        <h4>Set presence</h4>
			<div className="presencesetter">
				<button title="Set presence to online" onClick={() => sendNewPresence({...(presence || {}), "presence": "online"})} type="button">{PresenceEmojis["online"]} Online</button>
				<button title="Set presence to unavailable" onClick={() => sendNewPresence({...(presence || {}), "presence": "unavailable"})} type="button">{PresenceEmojis["unavailable"]} Unavailable</button>
				<button title="Set presence to offline" onClick={() => sendNewPresence({...(presence || {}), "presence": "offline"})} type="button">{PresenceEmojis["offline"]} Offline</button>
			</div>
			<p></p>
			<div className="statussetter">
				<form className={presence?.status_msg ? "canclear" : "cannotclear"} onSubmit={(e) => {e.preventDefault(); sendNewPresence({...(presence || {}), "status_msg": (e.currentTarget[0] as HTMLInputElement).value})}}>
					<input type="text" placeholder="Status message" defaultValue={presence?.status_msg || ""}/>
					{presence?.status_msg && <button title="Clear status" type="button" onClick={(e) => clearStatusMessage(e)}>Clear</button>}
					<button title="Set status message" type="submit">Set</button>
				</form>
			</div>
        </>
    )
}
