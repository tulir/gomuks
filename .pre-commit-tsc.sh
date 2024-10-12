#!/bin/bash
cd web > /dev/null
if [[ -f "./node_modules/.bin/tsc" ]]; then
	tsc --build --noEmit
fi
