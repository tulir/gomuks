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

export interface AndroidRegisterPushEvent {
	type: "register_push"
	device_id: string
	token: string
	encryption: { key: string }
	expiration?: number
}

export interface AndroidAuthEvent {
	type: "auth"
	authorization: `Bearer ${string}`
}

export type GomuksAndroidMessageToWeb = AndroidRegisterPushEvent | AndroidAuthEvent
