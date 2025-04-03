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
import "monaco-editor/esm/vs/basic-languages/css/css.contribution.js"
import "monaco-editor/esm/vs/editor/edcore.main.js"
import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js"
import CSSWorker from "monaco-editor/esm/vs/language/css/css.worker.js?worker"
import "monaco-editor/esm/vs/language/css/monaco.contribution.js"
import { RefObject, memo, useLayoutEffect, useRef } from "react"

window.MonacoEnvironment = {
	getWorker: function() {
		return new CSSWorker()
	},
}

export interface MonacoProps {
	initData: string
	onClose: () => void
	onSave: () => void
	contentRef: RefObject<string>
}

const Monaco = ({ initData, onClose, onSave, contentRef }: MonacoProps) => {
	const container = useRef<HTMLDivElement>(null)
	const editor = useRef<monaco.editor.IStandaloneCodeEditor>(null)
	useLayoutEffect(() => {
		if (!container.current) {
			return
		}
		const newEditor = monaco.editor.create(container.current, {
			language: "css",
			value: initData,
			fontLigatures: true,
			fontFamily: `var(--monospace-font-stack)`,
			theme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "vs-dark" : "vs",
		})
		const model = newEditor.getModel()
		if (!model) {
			return
		}
		model.onDidChangeContent(() => contentRef.current = model.getValue(monaco.editor.EndOfLinePreference.LF))
		newEditor.onKeyDown(evt => {
			if (evt.keyCode === monaco.KeyCode.Escape) {
				onClose()
			} else if ((evt.ctrlKey || evt.metaKey) && evt.keyCode === monaco.KeyCode.KeyS) {
				onSave()
				evt.preventDefault()
			}
		})
		newEditor.focus()
		editor.current = newEditor
		return () => newEditor.dispose()
		// All props are intentionally immutable
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [])
	return <div style={{ width: "100%", height: "100%" }} ref={container}/>
}

export default memo(Monaco)
