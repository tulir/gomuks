# gomuks
![Languages](https://img.shields.io/github/languages/top/tulir/gomuks.svg)
[![License](https://img.shields.io/github/license/tulir/gomuks.svg)](LICENSE)
[![Release](https://img.shields.io/github/release/tulir/gomuks/all.svg)](https://github.com/tulir/gomuks/releases)
[![Build Status](https://travis-ci.org/tulir/gomuks.svg?branch=master)](https://travis-ci.org/tulir/gomuks)
[![Maintainability](https://img.shields.io/codeclimate/maintainability/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)
[![Coverage](https://img.shields.io/codeclimate/coverage/tulir/gomuks.svg)](https://codeclimate.com/github/tulir/gomuks)

![Chat Preview](chat-preview.png)

A terminal Matrix client written in Go using [gomatrix](https://github.com/matrix-org/gomatrix) and [tview](https://github.com/rivo/tview).

Basic usage is possible, but expect bugs and missing features.

## Discussion
Matrix room: [#gomuks:maunium.net](https://matrix.to/#/#gomuks:maunium.net)

## Installation
Once the client becomes actually usable, I'll start making GitHub releases with
precompiled executables and maybe even some Linux packages.

For now, you'll have to compile from source:

0. Install [Go](https://golang.org/) 1.10 or higher
1. Run `go get -u maunium.net/go/gomuks`
2. gomuks should now be in `$GOPATH/bin/gomuks`

## Usage
Switch between rooms by clicking or with ctrl + up/down arrow (alt+arrows works too).

Scroll chat with the scroll wheel (3 rows per tick), page up/down (half of height per click) or up/down arrow (1 row per click)

### Commands
* `/quit` - Close gomuks
* `/logout` - Log out, clear caches and go back to the login view
* `/clearcache` - Clear room state cache and close gomuks
* `/leave` - Leave the current room
* `/join <room>` - Join the room with the given room ID or alias
* `/panic` - Trigger a test panic
