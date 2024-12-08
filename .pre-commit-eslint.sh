#!/usr/bin/env bash
cd web > /dev/null
if [[ -f "./node_modules/.bin/eslint" ]]; then
	ARGS=("$@")
	./node_modules/.bin/eslint --fix ${ARGS[@]/#web\// }
fi
