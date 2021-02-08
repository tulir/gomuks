// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// +build !cgo

package ui

func autocompleteDevice(cmd *CommandAutocomplete) ([]string, string) {
	return []string{}, ""
}

func autocompleteUser(cmd *CommandAutocomplete) ([]string, string) {
	return []string{}, ""
}

func cmdNoCrypto(cmd *Command) {
	cmd.Reply("This gomuks was built without encryption support")
}

var (
	cmdDevices        = cmdNoCrypto
	cmdDevice         = cmdNoCrypto
	cmdVerifyDevice   = cmdNoCrypto
	cmdVerify         = cmdNoCrypto
	cmdUnverify       = cmdNoCrypto
	cmdBlacklist      = cmdNoCrypto
	cmdResetSession   = cmdNoCrypto
	cmdImportKeys     = cmdNoCrypto
	cmdExportKeys     = cmdNoCrypto
	cmdExportRoomKeys = cmdNoCrypto
	cmdSSSS           = cmdNoCrypto
	cmdCrossSigning   = cmdNoCrypto
)
