// Copyright 2013 @atotto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clipboard read/write on clipboard
package clipboard

import ()

// ReadAll read string from clipboard
func ReadAll(register string) (string, error) {
	return readAll(register)
}

// WriteAll write string to clipboard
func WriteAll(text string, register string) error {
	return writeAll(text, register)
}

// Unsupported might be set true during clipboard init, to help callers decide
// whether or not to offer clipboard options.
var Unsupported bool
