# gomuks
![Languages](https://img.shields.io/github/languages/top/tulir/gomuks.svg)
[![License](https://img.shields.io/github/license/tulir/gomuks.svg)](LICENSE)
[![Release](https://img.shields.io/github/release/tulir/gomuks/all.svg)](https://github.com/tulir/gomuks/releases)
[![Build Status](https://travis-ci.org/tulir/gomuks.svg?branch=master)](https://travis-ci.org/tulir/gomuks)
[![Maintainability](https://img.shields.io/codeclimate/maintainability/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)
[![Coverage](https://img.shields.io/codeclimate/coverage/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)

![Chat Preview](chat-preview.png)

A terminal Matrix client written in Go using [mautrix](https://github.com/matrix-org/mautrix) and [tview](https://github.com/rivo/tview).

Basic usage is possible, but expect bugs and missing features.

## Discussion
Matrix room: [#gomuks:maunium.net](https://matrix.to/#/#gomuks:maunium.net)

## Installation
Once the client becomes actually usable, I'll start making GitHub releases with
precompiled executables. For now, you can either download
a CI build from [dl.maunium.net/programs/gomuks](https://dl.maunium.net/programs/gomuks)
or compile from source:

0. Install [Go](https://golang.org/) 1.11 or higher
1. Clone the repo: `git clone https://github.com/tulir/gomuks.git && cd gomuks`
2. Build: `go build`

Simply pull changes (`git pull`) and run `go build` again to update.

## Developing
For debugging, use `tail -f /tmp/gomuks-debug.log` and write to it using the methods in the `maunium.net/go/gomuks/debug` package:
```go
import (
	"maunium.net/go/gomuks/debug"
)
...
func Foo() {
	debug.Print("WHY ISN'T IT WORKING?!?!?")
}
```

## Usage
- switch rooms - `Ctrl + ↑` `Ctrl + ↓` `Alt + ↑` `Alt + ↓`
- scroll chat (line) - `↑` `↓`
- scroll chat (page) - `PgUp` `PgDown`
- jump to room - `Alt + Enter`, then `Tab` and `Enter` to navigate and select room

### Commands
* `/help` - Is a known command
* `/me <text>` - Send an emote
* `/quit` - Close gomuks
* `/clearcache` - Clear room state and close gomuks
* `/leave` - Leave the current room
* `/join <room>` - Join the room with the given room ID or alias
* `/toggle <rooms/users/baremessages/images/typingnotif>` - Change user preferences
* `/logout` - Log out, clear caches and go back to the login view
* `/send <room id> <event type> <content>` - Send a custom event
* `/setstate <room id> <event type> <state key/-> <content>` - Change room state
