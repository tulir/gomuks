import {defineConfig} from "vite"
import react from "@vitejs/plugin-react-swc"

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
			}
		},
	},
})
