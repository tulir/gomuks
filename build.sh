#!/usr/bin/env bash
go generate ./web
export MAUTRIX_VERSION=$(cat go.mod | grep 'maunium.net/go/mautrix ' | head -n1 | awk '{ print $2 }')
go build -ldflags "-X go.mau.fi/gomuks/version.Tag=$(git describe --exact-match --tags 2>/dev/null) -X go.mau.fi/gomuks/version.Commit=$(git rev-parse HEAD) -X 'go.mau.fi/gomuks/version.BuildTime=`date -Iseconds`' -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION'" ./cmd/gomuks "$@" || exit 2
