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
import Client from "./api/client.ts"
import WSClient from "./api/wsclient.ts"
import ClientContext from "./ui/ClientContext.ts"
import MainScreen from "./ui/MainScreen.tsx"
import { LoginScreen, VerificationScreen } from "./ui/login"
import { LightboxWrapper } from "./ui/modal/Lightbox.tsx"
import { ModalWrapper } from "./ui/modal/Modal.tsx"
import { useEventAsState } from "./util/eventdispatcher.ts"

function App() {
	const client = useMemo(() => new Client(new WSClient("_gomuks/websocket")), [])
	window.client = client
	const connState = useEventAsState(client.rpc.connect)
	const clientState = useEventAsState(client.state)
	useEffect(() => client.start(), [client])

	if (connState?.error) {
		return <div>
			{`${connState.error} \u{1F63F}`}
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
		return <ClientContext value={client}>
			<LightboxWrapper>
				<ModalWrapper>
					<MainScreen/>
				</ModalWrapper>
			</LightboxWrapper>
		</ClientContext>
	}
}

export default App
