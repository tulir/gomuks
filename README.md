# gomuks
![Languages](https://img.shields.io/github/languages/top/tulir/gomuks.svg)
[![License](https://img.shields.io/github/license/tulir/gomuks.svg)](LICENSE)
[![Release](https://img.shields.io/github/release/tulir/gomuks/all.svg)](https://github.com/tulir/gomuks/releases)
[![Build Status](https://travis-ci.org/tulir/gomuks.svg?branch=master)](https://travis-ci.org/tulir/gomuks)
[![GitLab CI](https://mau.dev/tulir/gomuks/badges/master/pipeline.svg)](https://mau.dev/tulir/gomuks/pipelines)
[![Maintainability](https://img.shields.io/codeclimate/maintainability/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)
[![Coverage](https://img.shields.io/codeclimate/coverage/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)

![Chat Preview](chat-preview.png)

A terminal Matrix client written in Go using [mautrix](https://github.com/tulir/mautrix-go) and [mauview](https://github.com/tulir/mauview).

Basic usage is possible, but expect bugs and missing features.

## Discussion
Matrix room: [#gomuks:maunium.net](https://matrix.to/#/#gomuks:maunium.net)

## Installation
Once the client becomes actually usable, I'll start making GitHub releases with
precompiled executables. For now, you can either download
a CI build from [GitLab CI](https://mau.dev/tulir/gomuks/pipelines)
or compile from source:

0. Install [Go](https://golang.org/) 1.12 or higher
1. Clone the repo: `git clone https://github.com/tulir/gomuks.git && cd gomuks`
2. Build: `go build`

Simply pull changes (`git pull`) and run `go build` again to update.

## Developing
Set `DEBUG=1` to enable partial deadlock detection and to write panics to stdout instead of a file.

To build and run with [race detection](https://golang.org/doc/articles/race_detector.html),
use `go install -race` and set `GORACE='history_size=7 log_path=/tmp/gomuks/race.log'`
when starting gomuks, then check `/tmp/gomuks/race.log.<pid>`. Note that race detection
will use a lot of extra resources.

For debugging, use `tail -f /tmp/gomuks/debug.log` and write to it using the
methods in the `maunium.net/go/gomuks/debug` package:
```go
package foo

import (
	"maunium.net/go/gomuks/debug"
)

func Foo() {
	debug.Print("WHY ISN'T IT WORKING?!?!?")
	debug.PrintStack()
}
```

## Usage
- switch rooms - `Ctrl + ↑` `Ctrl + ↓` `Alt + ↑` `Alt + ↓`
- ~~scroll chat (line) - `↑` `↓`~~
- scroll chat (page) - `PgUp` `PgDown`
- jump to room - `Alt + Enter`, then `Tab` and `Enter` to navigate and select room

### Commands
* `/help` - View command list
* `/me <text>` - Send an emote
* `/quit` - Close gomuks
* `/clearcache` - Clear room state and close gomuks
* `/leave` - Leave the current room
* `/create <room name>` - Create a new Matrix room.
* `/join <room>` - Join the room with the given room ID or alias
* `/toggle <rooms/users/baremessages/images/typingnotif>` - Change user preferences
* `/logout` - Log out, clear caches and go back to the login view
* `/send <room id> <event type> <content>` - Send a custom event
* `/setstate <room id> <event type> <state key/-> <content>` - Change room state
* `/msend <event type> <content>` - Send a custom event to the current room
* `/msetstate <event type> <state key/-> <content>` - Change room state in the current room
