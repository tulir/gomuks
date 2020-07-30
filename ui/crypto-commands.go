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

// +build cgo

package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/id"
)

func autocompleteDeviceUserID(cmd *CommandAutocomplete) (completions []string, newText string) {
	userCompletions := cmd.Room.AutocompleteUser(cmd.Args[0])
	if len(userCompletions) == 1 {
		newText = fmt.Sprintf("/%s %s ", cmd.OrigCommand, userCompletions[0].id)
	} else {
		completions = make([]string, len(userCompletions))
		for i, completion := range userCompletions {
			completions[i] = completion.id
		}
	}
	return
}

func autocompleteDeviceDeviceID(cmd *CommandAutocomplete) (completions []string, newText string) {
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	devices, err := mach.CryptoStore.GetDevices(id.UserID(cmd.Args[0]))
	if len(devices) == 0 || err != nil {
		return
	}
	var completedDeviceID id.DeviceID
	if len(cmd.Args) > 1 {
		existingID := strings.ToUpper(cmd.Args[1])
		for _, device := range devices {
			deviceIDStr := string(device.DeviceID)
			if deviceIDStr == existingID {
				// We don't want to do any autocompletion if there's already a full device ID there.
				return []string{}, ""
			} else if strings.HasPrefix(strings.ToUpper(device.Name), existingID) || strings.HasPrefix(deviceIDStr, existingID) {
				completedDeviceID = device.DeviceID
				completions = append(completions, fmt.Sprintf("%s (%s)", device.DeviceID, device.Name))
			}
		}
	} else {
		completions = make([]string, len(devices))
		i := 0
		for _, device := range devices {
			completedDeviceID = device.DeviceID
			completions[i] = fmt.Sprintf("%s (%s)", device.DeviceID, device.Name)
			i++
		}
	}
	if len(completions) == 1 {
		newText = fmt.Sprintf("/%s %s %s ", cmd.OrigCommand, cmd.Args[0], completedDeviceID)
	}
	return
}

func autocompleteDevice(cmd *CommandAutocomplete) ([]string, string) {
	if len(cmd.Args) == 0 {
		return []string{}, ""
	} else if len(cmd.Args) == 1 && !unicode.IsSpace(rune(cmd.RawArgs[len(cmd.RawArgs)-1])) {
		return autocompleteDeviceUserID(cmd)
	} else if cmd.Command != "devices" {
		return autocompleteDeviceDeviceID(cmd)
	}
	return []string{}, ""
}

func getDevice(cmd *Command) *crypto.DeviceIdentity {
	if len(cmd.Args) < 2 {
		cmd.Reply("Usage: /%s <user id> <device id> [fingerprint]", cmd.Command)
		return nil
	}
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	device, err := mach.GetOrFetchDevice(id.UserID(cmd.Args[0]), id.DeviceID(cmd.Args[1]))
	if err != nil {
		cmd.Reply("Failed to get device: %v", err)
		return nil
	}
	return device
}

func putDevice(cmd *Command, device *crypto.DeviceIdentity, action string) {
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	err := mach.CryptoStore.PutDevice(device.UserID, device)
	if err != nil {
		cmd.Reply("Failed to save device: %v", err)
	} else {
		cmd.Reply("Successfully %s %s/%s (%s)", action, device.UserID, device.DeviceID, device.Name)
	}
	mach.OnDevicesChanged(device.UserID)
}

func cmdDevices(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /devices <user id>")
		return
	}
	userID := id.UserID(cmd.Args[0])
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	devices, err := mach.CryptoStore.GetDevices(userID)
	if err != nil {
		cmd.Reply("Failed to get device list: %v", err)
	}
	if len(devices) == 0 {
		cmd.Reply("Fetching device list from server...")
		devices = mach.LoadDevices(userID)
	}
	if len(devices) == 0 {
		cmd.Reply("No devices found for %s", userID)
		return
	}
	var buf strings.Builder
	for _, device := range devices {
		_, _ = fmt.Fprintf(&buf, "%s (%s) - %s\n    Fingerprint: %s\n", device.DeviceID, device.Name, device.Trust.String(), device.Fingerprint())
	}
	resp := buf.String()
	cmd.Reply("%s", resp[:len(resp)-1])
}

func cmdDevice(cmd *Command) {
	device := getDevice(cmd)
	if device == nil {
		return
	}
	deviceType := "Device"
	if device.Deleted {
		deviceType = "Deleted device"
	}
	cmd.Reply("%s %s of %s\nFingerprint: %s\nIdentity key: %s\nDevice name: %s\nTrust state: %s",
		deviceType, device.DeviceID, device.UserID,
		device.Fingerprint(), device.IdentityKey,
		device.Name, device.Trust.String())
}

func cmdVerify(cmd *Command) {
	device := getDevice(cmd)
	if device == nil {
		return
	}
	if device.Trust == crypto.TrustStateVerified {
		cmd.Reply("That device is already verified")
		return
	}
	if len(cmd.Args) == 2 {
		mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
		mach.DefaultSASTimeout = 120 * time.Second
		modal := NewVerificationModal(cmd.MainView, device, mach.DefaultSASTimeout)
		cmd.MainView.ShowModal(modal)
		_, err := mach.NewSimpleSASVerificationWith(device, modal)
		if err != nil {
			cmd.Reply("Failed to start interactive verification: %v", err)
			return
		}
	} else {
		fingerprint := strings.Join(cmd.Args[2:], "")
		if string(device.SigningKey) != fingerprint {
			cmd.Reply("Mismatching fingerprint")
			return
		}
		action := "verified"
		if device.Trust == crypto.TrustStateBlacklisted {
			action = "unblacklisted and verified"
		}
		device.Trust = crypto.TrustStateVerified
		putDevice(cmd, device, action)
	}
}

func cmdUnverify(cmd *Command) {
	device := getDevice(cmd)
	if device == nil {
		return
	}
	if device.Trust == crypto.TrustStateUnset {
		cmd.Reply("That device is already not verified")
		return
	}
	action := "unverified"
	if device.Trust == crypto.TrustStateBlacklisted {
		action = "unblacklisted"
	}
	device.Trust = crypto.TrustStateUnset
	putDevice(cmd, device, action)
}

func cmdBlacklist(cmd *Command) {
	device := getDevice(cmd)
	if device == nil {
		return
	}
	if device.Trust == crypto.TrustStateBlacklisted {
		cmd.Reply("That device is already blacklisted")
		return
	}
	action := "blacklisted"
	if device.Trust == crypto.TrustStateVerified {
		action = "unverified and blacklisted"
	}
	device.Trust = crypto.TrustStateBlacklisted
	putDevice(cmd, device, action)
}

func cmdResetSession(cmd *Command) {
	err := cmd.Matrix.Crypto().(*crypto.OlmMachine).CryptoStore.RemoveOutboundGroupSession(cmd.Room.Room.ID)
	if err != nil {
		cmd.Reply("Failed to remove outbound group session: %v", err)
	} else {
		cmd.Reply("Removed outbound group session for this room")
	}
}
