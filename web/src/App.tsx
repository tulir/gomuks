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
import { useEffect, useMemo } from "react"
import { ScaleLoader } from "react-spinners"
import { useEventAsState } from "./util/eventdispatcher.ts"
import Client from "./api/client.ts"
import WSClient from "./api/wsclient.ts"
import { LoginScreen, VerificationScreen } from "./ui/login"
import MainScreen from "./ui/MainScreen.tsx"

function App() {
	const client = useMemo(() => new Client(new WSClient("/_gomuks/websocket")), [])
	const connState = useEventAsState(client.rpc.connect)
	const clientState = useEventAsState(client.state)
	useEffect(() => {
		((window as unknown) as { client: Client }).client = client

		// TODO remove this debug log
		const unlistenDebug = client.rpc.event.listen(ev => {
			console.debug("Received event:", ev)
		})
		client.rpc.start()
		return () => {
			unlistenDebug()
			client.rpc.stop()
		}
	}, [client])

	if (connState?.error) {
		return <div>
			error {`${connState.error}`} :(
		</div>
	} else if (!connState?.connected || !clientState) {
		const msg = connState?.connected ?
			"Waiting for client state..." : "Connecting to backend..."
		return <div>
			<ScaleLoader/>
			{msg}
		</div>
	} else if (!clientState.is_logged_in) {
		return <LoginScreen client={client} clientState={clientState}/>
	} else if (!clientState.is_verified) {
		return <VerificationScreen client={client} clientState={clientState}/>
	} else {
		return <MainScreen client={client} />
	}
}

export default App
