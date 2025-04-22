/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

import type Client from "@/api/client.ts"
import type { GCSettings, RoomStateStore } from "@/api/statestore"
import type { MainScreenContextFields } from "@/ui/MainScreenContext.ts"
import type { openModal } from "@/ui/modal/contexts.ts"
import type { RoomContextData } from "@/ui/roomview/roomcontext.ts"

declare global {
	interface Window {
		client: Client
		activeRoom?: RoomStateStore | null
		activeRoomContext?: RoomContextData
		mainScreenContext: MainScreenContextFields
		openLightbox: (params: { src: string, alt: string }) => void
		gcSettings: GCSettings
		hackyOpenEventContextMenu?: string
		closeModal: () => void
		closeNestableModal: () => void
		openModal: openModal
		openNestableModal: openModal
		gomuksAndroid?: true
	}
}
