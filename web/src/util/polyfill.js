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

if (!window.Iterator?.prototype.map) {
	const mapIterProto = (new Map([])).keys().__proto__
	const regexIterProto = "a".matchAll(/a/g).__proto__
	const identity = x => x
	for (const iterProto of [mapIterProto, regexIterProto]) {
		iterProto.map = function(callbackFn) {
			const output = []
			let i = 0
			for (const item of this) {
				output.push(callbackFn(item, i))
				i++
			}
			return output
		}
		iterProto.filter = function(callbackFn) {
			const output = []
			let i = 0
			for (const item of this) {
				if (callbackFn(item, i)) {
					output.push(item)
				}
				i++
			}
			return output
		}
		iterProto.toArray = function() {
			return this.map(identity)
		}
	}
	Array.prototype.toArray = function() {
		return this
	}
}
