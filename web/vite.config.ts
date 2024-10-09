import react from "@vitejs/plugin-react-swc"
import { defineConfig } from "vite"

export default defineConfig({
	plugins: [react()],
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
