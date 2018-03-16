# gomuks
![Preview](https://img.mau.lu/6sgqm.png)

A terminal Matrix client written in Go using [gomatrix](https://github.com/matrix-org/gomatrix) and [tview](https://github.com/rivo/tview).

Very basic usage is possible, but many of the features you'd expect from a
Matrix client (like chat history) haven't been implemented.

## Installation
Once the client becomes actually usable, I'll start making GitHub releases with
precompiled executables and maybe even some Linux packages.

For now, you'll have to compile from source:

0. Install [Go](https://golang.org/)
1. Run `go get -u maunium.net/go/gomuks`
2. gomuks should now be in `$GOPATH/bin/gomuks`
