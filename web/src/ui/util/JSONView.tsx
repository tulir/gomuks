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
import { useReducer } from "react"
import "./JSONView.css"

interface JSONViewProps {
	data: unknown
}

interface JSONViewPropsWithKey extends JSONViewProps {
	objectKey?: string
	trailingComma?: boolean
	noCollapse?: boolean
}

function renderJSONString(data: string, styleClass: string = "s2") {
	return <span className={`json-string ${styleClass}`}>{JSON.stringify(data)}</span>
}

function renderJSONValue(data: unknown, collapsed: boolean) {
	switch (typeof data) {
	case "object":
		if (data === null) {
			return <span className="json-null kc">null</span>
		} else if (Array.isArray(data)) {
			if (data.length === 0) {
				return null
			} else if (collapsed) {
				return <span className="json-collapsed">…</span>
			}
			return <ol className="json-array-children">
				{data.map((item, index, arr) =>
					<li key={index} className="json-array-entry">
						<JSONValueWithKey data={item} trailingComma={index < arr.length - 1}/>
					</li>)}
			</ol>
		} else {
			const entries = Object.entries(data)
			if (entries.length === 0) {
				return null
			} else if (collapsed) {
				return <span className="json-collapsed">…</span>
			}
			return <ul className="json-object-children">
				{entries.map(([key, value], index, arr) =>
					value !== undefined ? <li key={key} className="json-object-entry">
						<JSONValueWithKey data={value} objectKey={key} trailingComma={index < arr.length - 1}/>
					</li> : null)}
			</ul>
		}
	case "string":
		return renderJSONString(data)
	case "number":
		return <span className="json-number mf">{data}</span>
	case "boolean":
		return <span className="json-boolean kc">{data ? "true" : "false"}</span>
	default:
		return <span className="json-unknown">undefined</span>
	}
}

function JSONValueWithKey({ data, objectKey, trailingComma, noCollapse }: JSONViewPropsWithKey) {
	const [collapsed, toggleCollapsed] = useReducer(collapsed => !collapsed, false)
	const renderedKey = objectKey
		? <span className="json-object-key">
			{renderJSONString(objectKey, "nt")}
			<span className="json-object-entry-colon p">: </span>
		</span>
		: null
	const renderedSuffix = trailingComma
		? <span className="json-object-comma p">,</span>
		: null
	const collapseButton = noCollapse ? null :
		<span
			className="button"
			data-symbol={collapsed ? "+" : "-"}
			onClick={toggleCollapsed}
			title={collapsed ? "Expand" : "Collapse"}
		/>
	if (Array.isArray(data)) {
		return <>
			{renderedKey}
			{collapseButton}
			<span className="json-array-bracket p">[</span>
			{renderJSONValue(data, collapsed)}
			<span className="json-array-bracket p">]</span>
			{renderedSuffix}
		</>
	} else if (data !== null && typeof data === "object") {
		return <>
			{renderedKey}
			{collapseButton}
			<span className="json-object-brace p">{"{"}</span>
			{renderJSONValue(data, collapsed)}
			<span className="json-object-brace p">{"}"}</span>
			{renderedSuffix}
		</>
	}
	return <>
		{renderedKey}
		<span className="json-comma-container">
			{renderJSONValue(data, collapsed)}
			{renderedSuffix}
		</span>
	</>
}

export default function JSONView({ data }: JSONViewProps) {
	return <pre className="json-view chroma">
		<JSONValueWithKey data={data} noCollapse={true} />
	</pre>
}
