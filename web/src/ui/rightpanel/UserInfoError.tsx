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
import ErrorIcon from "@/icons/error.svg?react"

const UserInfoError = ({ errors }: { errors: string[] | null }) => {
	if (!errors?.length) {
		return null
	}
	return <div className="errors">{errors.map((err, i) => <div className="error" key={i}>
		<div className="icon"><ErrorIcon/></div>
		<p>{err}</p>
	</div>)}</div>
}

export default UserInfoError
