# This is a fork of atotto/clipboard

This fork is used for `zyedidia/micro` and has some modifications, namely: support for the primary clipboard on linux and support for an internal clipboard if the system clipboard is not available.

[![Build Status](https://travis-ci.org/atotto/clipboard.svg?branch=master)](https://travis-ci.org/atotto/clipboard) [![Build Status](https://drone.io/github.com/atotto/clipboard/status.png)](https://drone.io/github.com/atotto/clipboard/latest) 

[![GoDoc](https://godoc.org/github.com/atotto/clipboard?status.svg)](http://godoc.org/github.com/atotto/clipboard)

# Clipboard for Go

Provide copying and pasting to the Clipboard for Go.

Download shell commands at https://drone.io/github.com/atotto/clipboard/files

Build:

    $ go get github.com/atotto/clipboard

Platforms:

* OSX
* Windows 7 (probably work on other Windows)
* Linux, Unix (requires 'xclip' or 'xsel' command to be installed)


Document: 

* http://godoc.org/github.com/atotto/clipboard

Notes:

* Text string only
* UTF-8 text encoding only (no conversion)

TODO:

* Clipboard watcher(?)

## Commands:

paste shell command:

    $ go get github.com/atotto/clipboard/cmd/gopaste
    $ # example:
    $ gopaste > document.txt

copy shell command:

    $ go get github.com/atotto/clipboard/cmd/gocopy
    $ # example:
    $ cat document.txt | gocopy



