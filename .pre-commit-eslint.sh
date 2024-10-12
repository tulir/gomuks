#!/bin/bash
cd web > /dev/null
if [[ -f "./node_modules/.bin/eslint" ]]; then
	ARGS=("$@")
	eslint ${ARGS[@]/#web\// }
fi
