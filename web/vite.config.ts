import react from "@vitejs/plugin-react-swc"
import { defineConfig } from "vite"
import svgr from "vite-plugin-svgr"

export default defineConfig({
	plugins: [
		react(),
		svgr({
			svgrOptions: {
				replaceAttrValues: {
					"#5f6368": "currentColor",
				},
			},
		}),
	],
	server: {
		proxy: {
			"/_gomuks/websocket": {
				target: "http://localhost:29325",
				ws: true,
			},
			"/_gomuks": {
				target: "http://localhost:29325",
			},
		},
	},
})
