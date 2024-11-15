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

const headers = {
	"Authorization": "Bearer BEEPER-PRIVATE-API-PLEASE-DONT-USE",
	"Content-Type": "application/json",
}

async function tryJSON(resp: Response): Promise<unknown> {
	try {
		return await resp.json()
	} catch (err) {
		console.error(err)
	}
}

export async function doSubmitCode(domain: string, request: string, response: string): Promise<string> {
	const resp = await fetch(`https://api.${domain}/user/login/response`, {
		method: "POST",
		body: JSON.stringify({ response, request }),
		headers,
	})
	const data = await tryJSON(resp) as { token?: string, error?: string }
	console.log("Login code submit response data:", data)
	if (!resp.ok) {
		throw new Error(data ? `HTTP ${resp.status} / ${data?.error ?? JSON.stringify(data)}` : `HTTP ${resp.status}`)
	} else if (!data || typeof data !== "object" || typeof data.token !== "string") {
		throw new Error(`No token returned`)
	}
	return data.token
}

export async function doRequestCode(domain: string, request: string, email: string) {
	const resp = await fetch(`https://api.${domain}/user/login/email`, {
		method: "POST",
		body: JSON.stringify({ email, request }),
		headers,
	})
	const data = await tryJSON(resp) as { error?: string }
	console.log("Login email submit response data:", data)
	if (!resp.ok) {
		throw new Error(data ? `HTTP ${resp.status} / ${data?.error ?? JSON.stringify(data)}` : `HTTP ${resp.status}`)
	}
}

export async function doStartLogin(domain: string): Promise<string> {
	const resp = await fetch(`https://api.${domain}/user/login`, {
		method: "POST",
		body: "{}",
		headers,
	})
	const data = await tryJSON(resp) as { request?: string, error?: string }
	console.log("Login start response data:", data)
	if (!resp.ok) {
		throw new Error(data ? `HTTP ${resp.status} / ${data?.error ?? JSON.stringify(data)}` : `HTTP ${resp.status}`)
	} else if (!data || typeof data !== "object" || typeof data.request !== "string") {
		throw new Error(`No request ID returned`)
	}
	return data.request
}
