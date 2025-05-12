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
import { CachedEventDispatcher, EventDispatcher } from "../util/eventdispatcher.ts"
import { CancellablePromise } from "../util/promise.ts"
import type {
	ClientWellKnown,
	DBPushRegistration,
	EventID,
	EventType,
	JSONValue,
	LoginFlowsResponse,
	LoginRequest,
	MembershipAction,
	Mentions,
	MessageEventContent,
	PaginationResponse,
	ProfileEncryptionInfo,
	RPCCommand,
	RPCEvent,
	RawDBEvent,
	ReceiptType,
	RelatesTo,
	RelationType,
	ReqCreateRoom,
	ResolveAliasResponse,
	RespCreateRoom,
	RespMediaConfig,
	RespOpenIDToken,
	RespRoomJoin,
	RespTurnServer,
	RoomAlias,
	RoomID,
	RoomStateGUID,
	RoomSummary,
	TimelineRowID,
	URLPreview,
	UserID,
	UserProfile,
} from "./types"

export interface ConnectionEvent {
	connected: boolean
	reconnecting: boolean
	error: string | null
	nextAttempt?: string
}

export class ErrorResponse extends Error {
	constructor(public data: unknown) {
		super(`${data}`)
	}
}

export interface SendMessageParams {
	room_id: RoomID
	base_content?: MessageEventContent
	extra?: Record<string, unknown>
	text: string
	media_path?: string
	relates_to?: RelatesTo
	mentions?: Mentions
	url_previews?: URLPreview[]
}

export default abstract class RPCClient {
	public readonly connect: CachedEventDispatcher<ConnectionEvent> = new CachedEventDispatcher()
	public readonly event: EventDispatcher<RPCEvent> = new EventDispatcher()
	protected readonly pendingRequests: Map<number, {
		resolve: (data: unknown) => void,
		reject: (err: Error) => void
	}> = new Map()
	#requestIDCounter: number = 1

	protected abstract isConnected: boolean
	protected abstract send(data: string): void
	public abstract start(): void
	public abstract stop(): void

	protected onCommand(data: RPCCommand) {
		if (data.command === "response" || data.command === "error") {
			const target = this.pendingRequests.get(data.request_id)
			if (!target) {
				console.error("Received response for unknown request:", data)
				return
			}
			this.pendingRequests.delete(data.request_id)
			if (data.command === "response") {
				target.resolve(data.data)
			} else {
				target.reject(new ErrorResponse(data.data))
			}
		} else {
			this.event.emit(data as RPCEvent)
		}
	}

	protected cancelRequest(request_id: number, reason: string) {
		if (!this.pendingRequests.has(request_id)) {
			console.debug("Tried to cancel unknown request", request_id)
			return
		}
		this.request("cancel", { request_id, reason }).then(
			() => console.debug("Cancelled request", request_id, "for", reason),
			err => console.debug("Failed to cancel request", request_id, "for", reason, err),
		)
	}

	protected get nextRequestID(): number {
		return this.#requestIDCounter++
	}

	request<Req, Resp>(command: string, data: Req): CancellablePromise<Resp> {
		if (!this.isConnected) {
			return new CancellablePromise((_resolve, reject) => {
				reject(new Error("Websocket not connected"))
			}, () => {
			})
		}
		const request_id = this.nextRequestID
		return new CancellablePromise((resolve, reject) => {
			if (!this.isConnected) {
				reject(new Error("Websocket not connected"))
				return
			}
			this.pendingRequests.set(request_id, { resolve: resolve as ((value: unknown) => void), reject })
			this.send(JSON.stringify({
				command,
				request_id,
				data,
			}))
		}, this.cancelRequest.bind(this, request_id))
	}

	logout(): Promise<boolean> {
		return this.request("logout", {})
	}

	sendMessage(params: SendMessageParams): Promise<RawDBEvent | null> {
		return this.request("send_message", params)
	}

	sendEvent(
		room_id: RoomID,
		type: EventType,
		content: unknown,
		disable_encryption: boolean = false,
		synchronous: boolean = false,
	): Promise<RawDBEvent> {
		return this.request("send_event", { room_id, type, content, disable_encryption, synchronous })
	}

	resendEvent(transaction_id: string): Promise<RawDBEvent> {
		return this.request("resend_event", { transaction_id })
	}

	reportEvent(room_id: RoomID, event_id: EventID, reason: string): Promise<boolean> {
		return this.request("report_event", { room_id, event_id, reason })
	}

	redactEvent(room_id: RoomID, event_id: EventID, reason: string): Promise<boolean> {
		return this.request("redact_event", { room_id, event_id, reason })
	}

	setState(
		room_id: RoomID, type: EventType, state_key: string, content: Record<string, unknown>,
		extra: { delay_ms?: number } = {},
	): Promise<EventID> {
		return this.request("set_state", { room_id, type, state_key, content, ...extra })
	}

	updateDelayedEvent(delay_id: string, action: string): Promise<void> {
		return this.request("update_delayed_event", { delay_id, action })
	}

	setMembership(room_id: RoomID, user_id: UserID, action: MembershipAction, reason?: string): Promise<void> {
		return this.request("set_membership", { room_id, user_id, action, reason })
	}

	setAccountData(type: EventType, content: unknown, room_id?: RoomID): Promise<boolean> {
		return this.request("set_account_data", { type, content, room_id })
	}

	markRead(room_id: RoomID, event_id: EventID, receipt_type: ReceiptType = "m.read"): Promise<boolean> {
		return this.request("mark_read", { room_id, event_id, receipt_type })
	}

	setTyping(room_id: RoomID, timeout: number): Promise<boolean> {
		return this.request("set_typing", { room_id, timeout })
	}

	getProfile(user_id: UserID): Promise<UserProfile> {
		return this.request("get_profile", { user_id })
	}

	setProfileField(field: string, value: JSONValue): Promise<boolean> {
		return this.request("set_profile_field", { field, value })
	}

	getMutualRooms(user_id: UserID): Promise<RoomID[]> {
		return this.request("get_mutual_rooms", { user_id })
	}

	getProfileEncryptionInfo(user_id: UserID): Promise<ProfileEncryptionInfo> {
		return this.request("get_profile_encryption_info", { user_id })
	}

	trackUserDevices(user_id: UserID): Promise<ProfileEncryptionInfo> {
		return this.request("track_user_devices", { user_id })
	}

	ensureGroupSessionShared(room_id: RoomID): Promise<boolean> {
		return this.request("ensure_group_session_shared", { room_id })
	}

	sendToDevice(
		event_type: EventType,
		messages: { [userId: string]: { [deviceId: string]: object } },
		encrypted: boolean = false,
	): Promise<void> {
		return this.request("send_to_device", { event_type, messages, encrypted })
	}

	getSpecificRoomState(keys: RoomStateGUID[]): Promise<RawDBEvent[]> {
		return this.request("get_specific_room_state", { keys })
	}

	getRoomState(
		room_id: RoomID, include_members = false, fetch_members = false, refetch = false,
	): Promise<RawDBEvent[]> {
		return this.request("get_room_state", { room_id, include_members, fetch_members, refetch })
	}

	getEvent(room_id: RoomID, event_id: EventID, unredact?: boolean): Promise<RawDBEvent> {
		return this.request("get_event", { room_id, event_id, unredact })
	}

	getRelatedEvents(room_id: RoomID, event_id: EventID, relation_type?: RelationType): Promise<RawDBEvent[]> {
		return this.request("get_related_events", { room_id, event_id, relation_type })
	}

	paginate(room_id: RoomID, max_timeline_id: TimelineRowID, limit: number): Promise<PaginationResponse> {
		return this.request("paginate", { room_id, max_timeline_id, limit })
	}

	paginateServer(room_id: RoomID, limit: number): Promise<PaginationResponse> {
		return this.request("paginate_server", { room_id, limit })
	}

	getRoomSummary(room_id_or_alias: RoomID | RoomAlias, via?: string[]): Promise<RoomSummary> {
		return this.request("get_room_summary", { room_id_or_alias, via })
	}

	joinRoom(room_id_or_alias: RoomID | RoomAlias, via?: string[], reason?: string): Promise<RespRoomJoin> {
		return this.request("join_room", { room_id_or_alias, via, reason })
	}

	knockRoom(room_id_or_alias: RoomID | RoomAlias, via?: string[], reason?: string): Promise<RespRoomJoin> {
		return this.request("knock_room", { room_id_or_alias, via, reason })
	}

	leaveRoom(room_id: RoomID, reason?: string): Promise<Record<string, never>> {
		return this.request("leave_room", { room_id, reason })
	}

	createRoom(request: ReqCreateRoom): Promise<RespCreateRoom> {
		return this.request("create_room", request)
	}

	muteRoom(room_id: RoomID, muted: boolean): Promise<boolean> {
		return this.request("mute_room", { room_id, muted })
	}

	resolveAlias(alias: RoomAlias): Promise<ResolveAliasResponse> {
		return this.request("resolve_alias", { alias })
	}

	discoverHomeserver(user_id: UserID): Promise<ClientWellKnown> {
		return this.request("discover_homeserver", { user_id })
	}

	getLoginFlows(homeserver_url: string): Promise<LoginFlowsResponse> {
		return this.request("get_login_flows", { homeserver_url })
	}

	login(homeserver_url: string, username: string, password: string): Promise<boolean> {
		return this.request("login", { homeserver_url, username, password })
	}

	loginCustom(homeserver_url: string, request: LoginRequest): Promise<boolean> {
		return this.request("login_custom", { homeserver_url, request })
	}

	verify(recovery_key: string): Promise<boolean> {
		return this.request("verify", { recovery_key })
	}

	requestOpenIDToken(): Promise<RespOpenIDToken> {
		return this.request("request_openid_token", {})
	}

	registerPush(reg: DBPushRegistration): Promise<boolean> {
		return this.request("register_push", reg)
	}

	getTurnServers(): Promise<RespTurnServer> {
		return this.request("get_turn_servers", {})
	}

	getMediaConfig(): Promise<RespMediaConfig> {
		return this.request("get_media_config", {})
	}

	setListenToDevice(listen: boolean): Promise<void> {
		return this.request("listen_to_device", listen)
	}
}
