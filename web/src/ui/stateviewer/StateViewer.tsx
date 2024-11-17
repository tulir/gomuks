import { RoomStateStore } from "@/api/statestore"
import { useState } from "react"
import JSONView from "../util/JSONView"

interface StateViewerProps {
    room: RoomStateStore
}

// 1. you go in the state viewer, it shows buttons with each state event type
// 2. you click a button, it shows the state keys for that type
// 3. you click a state key, it shows a JSONView of the event

const StateViewer = ({ room }: StateViewerProps) => {
    // state
    const [page, setPage] = useState("state-type")
    const [stateType, setStateType] = useState("")
    const [stateKey, setStateKey] = useState("")
    // button actions
    // 1
    const stateTypeClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
        const type = evt.currentTarget.getAttribute("data-state-type")
        if (!type) {
            return
        }
        console.log(room.state.get(type))
        // progress to 2
        setStateType(type)
        setPage("state-key")
    }
    // 2
    const stateKeyClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
        const key = evt.currentTarget.getAttribute("data-state-key")
        if (!key) {
            return
        }
        // progress to 3
        setStateKey(key)
        setPage("state-event")
    }

    // render 1, 2 or 3
    switch (page) {
    case "state-type":
        const types: string[] = []
        for (const [type] of room.state) {
            types.push(type)
        }
        return types.map(type => <button data-state-type={type} onClick={stateTypeClick}>{type}</button>)
    case "state-key":
        const keys: string[] = []
        for (const [key] of room.state.get(stateType) ?? []) { // ?? [] makes the rest pointless, TODO short circuit lol
            keys.push(key)
        }
        return keys.map(type => <button data-state-key={type} onClick={stateKeyClick}>{type}</button>)
    case "state-event":
        const content = room.getStateEvent(stateType, stateKey)
        return <JSONView data={content}/>
    }
}

export default StateViewer