#!/usr/bin/env bash
cd web > /dev/null
if [[ -f "./node_modules/.bin/tsc" ]]; then
	./node_modules/.bin/tsc --build --noEmit
fi
