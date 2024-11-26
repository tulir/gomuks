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
const minSizeForSet = 10

export function listDiff<T>(newArr: T[], oldArr: T[]): [added: T[], removed: T[]] {
	if (oldArr.length < minSizeForSet && newArr.length < minSizeForSet) {
		return [
			newArr.filter(item => !oldArr.includes(item)),
			oldArr.filter(item => !newArr.includes(item)),
		]
	}
	const oldSet = new Set(oldArr)
	const newSet = new Set(newArr)
	return [
		newArr.filter(item => !oldSet.has(item)),
		oldArr.filter(item => !newSet.has(item)),
	]
}

export function objectDiff<T>(
	newObj: Record<string, T>,
	oldObj: Record<string, T>,
	defaultValue: T,
	prevDefaultValue?: T,
): Map<string, { old: T, new: T }>
export function objectDiff<T>(
	newObj: Record<string, T>,
	oldObj: Record<string, T>,
): Map<string, { old?: T, new?: T }>
export function objectDiff<T>(
	newObj: Record<string, T>,
	oldObj: Record<string, T>,
	defaultValue?: T,
	prevDefaultValue?: T,
): Map<string, { old?: T, new?: T }> {
	const keys = new Set(Object.keys(oldObj).concat(Object.keys(newObj)))
	const diff = new Map<string, { old?: T, new?: T }>()
	for (const key of keys) {
		const oldVal = Object.prototype.hasOwnProperty.call(oldObj, key) ? oldObj[key] :
			(prevDefaultValue ?? defaultValue)
		const newVal = Object.prototype.hasOwnProperty.call(newObj, key) ? newObj[key] : defaultValue
		if (oldVal !== newVal) {
			diff.set(key, { old: oldVal, new: newVal })
		}
	}
	return diff
}
