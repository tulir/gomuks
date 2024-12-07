import { RoomStateStore } from "@/api/statestore"
import { useState, use } from "react"
import { EventType } from "@/api/types"
import JSONView from "../util/JSONView"
import ClientContext from "../ClientContext"
import useEvent from "@/util/useEvent"

interface StateViewerProps {
    room: RoomStateStore
}

interface StatePageProps {
    room: RoomStateStore,
    onClick?: (evt: React.MouseEvent<HTMLButtonElement>) => void,
    eventType?: EventType,
    stateKey?: string
}

interface StateState {
    page: "all" | "type" | "event",
    eventType?: EventType,
    stateKey?: string
}

const StateAll = ({ room, onClick }: StatePageProps) => {
    const types: string[] = []
    for (const [type] of room.state) {
        types.push(type)
    }
    return types.map(type => <button data-event-type={type} onClick={onClick}>{type}</button>)
}

const StateType = ({ room, onClick, eventType }: StatePageProps) => {
    if (eventType == null) {
        return
    }
    const keysMap = room.state.get(eventType)
    const keys: string[] = []
    for (const [key] of keysMap ?? []) {
        keys.push(key)
    }
    return keys.map(key => <button data-state-key={key} onClick={onClick}>{key.length == 0 ? "<empty>" : key}</button>)
}

const StateEvent = ({ room, eventType, stateKey }: StatePageProps) => {
    if (eventType == undefined || stateKey == undefined) {
        return
    }
    const content = room.getStateEvent(eventType, stateKey)
    return <JSONView data={content}/>
}

const StateViewer = ({ room }: StateViewerProps) => {
    const [state, setState] = useState({page: "all"} as StateState)
    const client = use(ClientContext)
    if (!room.stateLoaded) {
        client?.loadRoomState(room.roomID, { omitMembers: false, refetch: true })
    }
    const onClickAll = useEvent((evt: React.MouseEvent<HTMLButtonElement>) => {
        const type = evt.currentTarget.getAttribute("data-event-type")
        if (type == null) {
            return
        }
        setState({
            page: "type",
            eventType: type
        })
    })

    const onClickType = useEvent((evt: React.MouseEvent<HTMLButtonElement>) => {
        const key = evt.currentTarget.getAttribute("data-state-key")
        if (key == null) {
            return
        }
        setState({
            page: "event",
            eventType: state.eventType,
            stateKey: key
        })
    })

    const onClickBack = useEvent(() => {
        switch (state.page) {
        case "type":
            setState({
                page: "all"
            })
            return
        case "event":
            setState({
                page: "type",
                eventType: state.eventType
            })
            return
        }
    })

    let content = <></>
    switch (state.page) {
    case "all":
        content = <StateAll room={room} onClick={onClickAll}/>
        break
    case "type":
        content = <StateType room={room} onClick={onClickType} eventType={state.eventType}/>
        break
    case "event":
        content = <StateEvent room={room} eventType={state.eventType} stateKey={state.stateKey}/>
        break
    }
    return <>
        <h3>Explore room state</h3>
        {content}
        {state.page != "all" && <button onClick={onClickBack}>Back</button>}
    </>
}

export default StateViewer
