/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

import type Client from "@/api/client.ts"

declare global {
	interface Window {
		client: Client
		openLightbox: (params: { src: string, alt: string }) => void
	}
}
