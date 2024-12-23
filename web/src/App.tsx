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
import RPCClient from "./api/rpc.ts"
import WailsClient from "./api/wailsclient.ts"
import WSClient from "./api/wsclient.ts"
import ClientContext from "./ui/ClientContext.ts"
import MainScreen from "./ui/MainScreen.tsx"
import { LoginScreen, VerificationScreen } from "./ui/login"
import { LightboxWrapper } from "./ui/modal"
import { useEventAsState } from "./util/eventdispatcher.ts"

function makeRPCClient(): RPCClient {
	if (window.wails || window._wails || navigator.userAgent.includes("wails.io")) {
		return new WailsClient()
	}
	return new WSClient("_gomuks/websocket")
}

function App() {
	const client = useMemo(() => new Client(makeRPCClient()), [])
	const connState = useEventAsState(client.rpc.connect)
	const clientState = useEventAsState(client.state)
	useEffect(() => {
		window.client = client
		return client.start()
	}, [client])

	const afterConnectError = Boolean(connState?.error && connState.reconnecting && clientState?.is_verified)
	useEffect(() => {
		if (afterConnectError) {
			const cancelKeys = (evt: KeyboardEvent | MouseEvent) => evt.stopPropagation()
			document.body.addEventListener("keydown", cancelKeys, { capture: true })
			document.body.addEventListener("keyup", cancelKeys, { capture: true })
			document.body.addEventListener("click", cancelKeys, { capture: true })
			return () => {
				document.body.removeEventListener("keydown", cancelKeys, { capture: true })
				document.body.removeEventListener("keyup", cancelKeys, { capture: true })
				document.body.removeEventListener("click", cancelKeys, { capture: true })
			}
		}
	}, [afterConnectError])
	const errorOverlay = connState?.error ? <div
		className={`connection-error-wrapper ${afterConnectError ? "post-connect" : ""}`}
		tabIndex={-1}
	>
		<div className="connection-error-inner">
			<div>{connState.error} &#x1F63F;</div>
			{connState.reconnecting && <div>
				<ScaleLoader width="2rem" height="2rem" color="var(--primary-color)"/>
				Reconnecting to backend...
				{connState.nextAttempt ? <div><small>(next attempt at {connState.nextAttempt})</small></div> : null}
			</div>}
		</div>
	</div> : null

	if (connState?.error && !afterConnectError) {
		return errorOverlay
	} else if ((!connState?.connected && !afterConnectError) || !clientState) {
		const msg = connState?.connected ?
			"Waiting for client state..." : "Connecting to backend..."
		return <div className="pre-connect">
			<ScaleLoader width="2rem" height="2rem" color="var(--primary-color)"/>
			{msg}
		</div>
	} else if (!clientState.is_logged_in) {
		return <LoginScreen client={client} clientState={clientState}/>
	} else if (!clientState.is_verified) {
		return <VerificationScreen client={client} clientState={clientState}/>
	} else {
		return <ClientContext value={client}>
			<LightboxWrapper>
				<MainScreen/>
			</LightboxWrapper>
			{errorOverlay}
		</ClientContext>
	}
}

export default App
