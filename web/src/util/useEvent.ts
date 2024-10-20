import { useLayoutEffect, useRef } from "react"

type Fn<Params extends Array<unknown>> = (...args: Params) => void

function useEvent<P extends Array<unknown>>(fn: Fn<P>): (...args: P) => void {
	const ref = useRef<[Fn<P>, Fn<P>]>([fn, (...args) => ref[0](...args)]).current
	useLayoutEffect(() => {
		ref[0] = fn
	})
	return ref[1]
}

export default useEvent
