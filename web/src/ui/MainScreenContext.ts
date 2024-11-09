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
import { createContext } from "react"
import type { RoomID } from "@/api/types"
import type { RightPanelProps } from "./rightpanel/RightPanel.tsx"

export interface MainScreenContextFields {
	setActiveRoom: (roomID: RoomID | null) => void
	clickRoom: (evt: React.MouseEvent) => void
	clearActiveRoom: () => void

	setRightPanel: (props: RightPanelProps) => void
	closeRightPanel: () => void
	clickRightPanelOpener: (evt: React.MouseEvent) => void
}

const stubContext = {
	get setActiveRoom(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
	get clickRoom(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
	get clearActiveRoom(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
	get setRightPanel(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
	get closeRightPanel(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
	get clickRightPanelOpener(): never {
		throw new Error("MainScreenContext used outside main screen")
	},
}

const MainScreenContext = createContext<MainScreenContextFields>(stubContext)
window.mainScreenContext = stubContext

export default MainScreenContext
