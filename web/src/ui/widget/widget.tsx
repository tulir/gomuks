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
import { ClientWidgetApi, IWidget, Widget as WrappedWidget } from "matrix-widget-api"
import { memo } from "react"
import type Client from "@/api/client"
import type { RoomStateStore, WidgetListener } from "@/api/statestore"
import type { MemDBEvent, RoomID, SyncToDevice } from "@/api/types"
import { getDisplayname } from "@/util/validation"
import PermissionPrompt from "./PermissionPrompt"
import { memDBEventToIRoomEvent } from "./util"
import GomuksWidgetDriver from "./widgetDriver"
import "./Widget.css"

export interface WidgetProps {
	info: IWidget
	room: RoomStateStore
	client: Client
	onClose?: () => void
}

// TODO remove this after widgets start using a parameter for it
const addLegacyParams = (url: string, widgetID: string) => {
	const urlObj = new URL(url)
	urlObj.searchParams.set("parentUrl", window.location.href)
	urlObj.searchParams.set("widgetId", widgetID)
	return urlObj.toString()
}

class WidgetListenerImpl implements WidgetListener {
	constructor(private api: ClientWidgetApi) {}

	onTimelineEvent = (evt: MemDBEvent) => {
		this.api.feedEvent(memDBEventToIRoomEvent(evt))
			.catch(err => console.error("Failed to feed event", memDBEventToIRoomEvent(evt), err))
	}

	onStateEvent = (evt: MemDBEvent) => {
		this.api.feedStateUpdate(memDBEventToIRoomEvent(evt))
			.catch(err => console.error("Failed to feed state update", memDBEventToIRoomEvent(evt), err))
	}

	onRoomChange = (roomID: RoomID | null) => {
		this.api.setViewedRoomId(roomID)
	}

	onToDeviceEvent = (evt: SyncToDevice) => {
		this.api.feedToDevice({
			sender: evt.sender,
			type: evt.type,
			content: evt.content,
			// Why does this use the IRoomEvent interface??
			event_id: "",
			room_id: "",
			origin_server_ts: 0,
			unsigned: {},
		}, evt.encrypted).catch(err => console.error("Failed to feed to-device event", evt, err))
	}
}

const openPermissionPrompt = (requested: Set<string>): Promise<Set<string>> => {
	return new Promise(resolve => {
		window.openModal({
			content: <PermissionPrompt
				capabilities={requested}
				onConfirm={resolve}
			/>,
			dimmed: true,
			boxed: true,
			noDismiss: true,
			innerBoxClass: "permission-prompt",
		})
	})
}

const ReactWidget = ({ room, info, client, onClose }: WidgetProps) => {
	const wrappedWidget = new WrappedWidget(info)
	const driver = new GomuksWidgetDriver(client, room, openPermissionPrompt)
	const widgetURL = addLegacyParams(wrappedWidget.getCompleteUrl({
		widgetRoomId: room.roomID,
		currentUserId: client.userID,
		deviceId: client.state.current?.is_logged_in ? client.state.current.device_id : "",
		userDisplayName: getDisplayname(client.userID, room.getStateEvent("m.room.member", client.userID)?.content),
		clientId: "fi.mau.gomuks",
		clientTheme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
		clientLanguage: navigator.language,
	}), wrappedWidget.id)

	const handleIframe = (iframe: HTMLIFrameElement) => {
		console.info("Setting up widget API for", iframe)
		const clientAPI = new ClientWidgetApi(wrappedWidget, iframe, driver)
		clientAPI.setViewedRoomId(room.roomID)

		clientAPI.on("ready", () => console.info("Widget is ready"))
		// Suppress unnecessary events to avoid errors
		const noopReply = (evt: CustomEvent) => {
			evt.preventDefault()
			clientAPI.transport.reply(evt.detail, {})
		}
		const closeWidget = (evt: CustomEvent) => {
			noopReply(evt)
			onClose?.()
		}
		clientAPI.on("action:io.element.join", noopReply)
		clientAPI.on("action:im.vector.hangup", noopReply)
		clientAPI.on("action:io.element.device_mute", noopReply)
		clientAPI.on("action:io.element.tile_layout", noopReply)
		clientAPI.on("action:io.element.spotlight_layout", noopReply)
		clientAPI.on("action:io.element.close", closeWidget)
		clientAPI.on("action:set_always_on_screen", noopReply)
		const removeListener = client.addWidgetListener(new WidgetListenerImpl(clientAPI))

		return () => {
			console.info("Removing widget API")
			removeListener()
			clientAPI.stop()
			clientAPI.removeAllListeners()
		}
	}

	return <iframe
		key={crypto.randomUUID()}
		ref={handleIframe}
		src={widgetURL}
		className="widget-iframe"
		allow="microphone; camera; fullscreen; encrypted-media; display-capture; screen-wake-lock;"
	/>
}

export default memo(ReactWidget)
