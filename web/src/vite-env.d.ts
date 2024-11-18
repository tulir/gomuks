/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

import type Client from "@/api/client.ts"
import type { GCSettings, RoomStateStore } from "@/api/statestore"
import type { MainScreenContextFields } from "@/ui/MainScreenContext.ts"

declare global {
	interface Window {
		client: Client
		activeRoom?: RoomStateStore | null
		mainScreenContext: MainScreenContextFields
		openLightbox: (params: { src: string, alt: string }) => void
		gcSettings: GCSettings
	}
}
