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
import { MemDBEvent } from "@/api/types"
import JSONView from "../../util/JSONView.tsx"
import CopyIcon from "@/icons/copy.svg?react"

interface ViewSourceModalProps {
	evt: MemDBEvent
}

// TODO: change the copy button's text on copy, without having typescript scream at me.
// will i need to make a component for the copy button and change its state? hmm
const copyButtonOnClick = (evt: MemDBEvent) => {
	navigator.clipboard.writeText(JSON.stringify(evt, null, 4))
}

// TODO check with tulir that he in fact uses material design icons. i got the copy icon from google's site
const ViewSourceModal = ({ evt }: ViewSourceModalProps) => {
	return <div className="view-source-modal">
		<button onClick={() => {copyButtonOnClick(evt)}}><CopyIcon/> Copy</button>
		<JSONView data={evt} />
	</div>
}

export default ViewSourceModal
