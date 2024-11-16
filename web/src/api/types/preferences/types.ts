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
export enum PreferenceContext {
	Config = "config",
	Account = "account",
	Device = "device",
	RoomAccount = "room_account",
	RoomDevice = "room_device",
}

export const anyContext = [
	PreferenceContext.RoomDevice,
	PreferenceContext.RoomAccount,
	PreferenceContext.Device,
	PreferenceContext.Account,
	PreferenceContext.Config,
] as const

export const anyGlobalContext = [
	PreferenceContext.Device,
	PreferenceContext.Account,
	PreferenceContext.Config,
] as const

export const deviceSpecific = [
	PreferenceContext.RoomDevice,
	PreferenceContext.Device,
] as const

export type PreferenceValueType =
	| boolean
	| number
	| string
	| number[]
	| string[]
	| Record<string, unknown>
	| Record<string, unknown>[]
	| null;

interface PreferenceFields<T extends PreferenceValueType = PreferenceValueType> {
	displayName: string
	allowedContexts: readonly PreferenceContext[]
	defaultValue: T
	description: string
	allowedValues?: readonly T[]
}

export class Preference<T extends PreferenceValueType = PreferenceValueType> {
	public readonly displayName: string
	public readonly allowedContexts: readonly PreferenceContext[]
	public readonly defaultValue: T
	public readonly description?: string
	public readonly allowedValues?: readonly T[]

	constructor(fields: PreferenceFields<T>) {
		this.displayName = fields.displayName
		this.allowedContexts = fields.allowedContexts
		this.defaultValue = fields.defaultValue
		this.description = fields.description ?? ""
		this.allowedValues = fields.allowedValues
	}
}
