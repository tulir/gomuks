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
import { HTMLAttributes, ReactNode } from "react"
import "./TooltipButton.css"

export interface TooltipButtonProps extends HTMLAttributes<HTMLButtonElement> {
	tooltipDirection?: "top" | "bottom" | "left" | "right"
	tooltipText: string
	tooltipProps?: HTMLAttributes<HTMLDivElement>
	children: ReactNode
}

const TooltipButton = ({
	tooltipDirection, tooltipText, children, className, tooltipProps, ...attrs
}: TooltipButtonProps) => {
	if (!tooltipDirection) {
		tooltipDirection = "top"
	}
	className = className ? `with-tooltip ${className}` : "with-tooltip"
	const tooltipClassName = `button-tooltip button-tooltip-${tooltipDirection} ${tooltipProps?.className ?? ""}`
	return <button {...attrs} className={className}>
		{children}
		<div {...(tooltipProps ?? {})} className={tooltipClassName}>
			{tooltipText}
		</div>
	</button>
}

export default TooltipButton
