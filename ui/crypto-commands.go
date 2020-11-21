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
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/ssss"
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

func autocompleteUser(cmd *CommandAutocomplete) ([]string, string) {
	if len(cmd.Args) == 1 && !unicode.IsSpace(rune(cmd.RawArgs[len(cmd.RawArgs)-1])) {
		return autocompleteDeviceUserID(cmd)
	}
	return []string{}, ""
}

func autocompleteDevice(cmd *CommandAutocomplete) ([]string, string) {
	if len(cmd.Args) == 0 {
		return []string{}, ""
	} else if len(cmd.Args) == 1 && !unicode.IsSpace(rune(cmd.RawArgs[len(cmd.RawArgs)-1])) {
		return autocompleteDeviceUserID(cmd)
	}
	return autocompleteDeviceDeviceID(cmd)
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
		trust := device.Trust.String()
		if device.Trust == crypto.TrustStateUnset && mach.IsDeviceTrusted(device) {
			trust = "verified (transitive)"
		}
		_, _ = fmt.Fprintf(&buf, "%s (%s) - %s\n    Fingerprint: %s\n", device.DeviceID, device.Name, trust, device.Fingerprint())
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
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	trustState := device.Trust.String()
	if device.Trust == crypto.TrustStateUnset && mach.IsDeviceTrusted(device) {
		trustState = "verified (transitive)"
	}
	cmd.Reply("%s %s of %s\nFingerprint: %s\nIdentity key: %s\nDevice name: %s\nTrust state: %s",
		deviceType, device.DeviceID, device.UserID,
		device.Fingerprint(), device.IdentityKey,
		device.Name, trustState)
}

func crossSignDevice(cmd *Command, device *crypto.DeviceIdentity) {
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	err := mach.SignOwnDevice(device)
	if err != nil {
		cmd.Reply("Failed to upload cross-signing signature: %v", err)
	} else {
		cmd.Reply("Successfully cross-signed %s (%s)", device.DeviceID, device.Name)
	}
}

func cmdVerifyDevice(cmd *Command) {
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
		if device.UserID == cmd.Matrix.Client().UserID {
			crossSignDevice(cmd, device)
			device.Trust = crypto.TrustStateVerified
			putDevice(cmd, device, action)
		} else {
			putDevice(cmd, device, action)
			cmd.Reply("Warning: verifying individual devices of other users is not synced with cross-signing")
		}
	}
}

func cmdVerify(cmd *Command) {
	if len(cmd.Args) < 1 {
		cmd.Reply("Usage: /%s <user ID> [--force]", cmd.OrigCommand)
		return
	}
	force := len(cmd.Args) >= 2 && strings.ToLower(cmd.Args[1]) == "--force"
	userID := id.UserID(cmd.Args[0])
	room := cmd.Room.Room
	if !room.Encrypted {
		cmd.Reply("In-room verification is only supported in encrypted rooms")
		return
	}
	if (!room.IsDirect || room.OtherUser != userID) && !force {
		cmd.Reply("This doesn't seem to be a direct chat. Either switch to a direct chat with %s, "+
			"or use `--force` to start the verification anyway.", userID)
		return
	}
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	if mach.CrossSigningKeys == nil && !force {
		cmd.Reply("Cross-signing private keys not cached. Generate or fetch cross-signing keys with `/cross-signing`, " +
			"or use `--force` to start the verification anyway")
		return
	}
	modal := NewVerificationModal(cmd.MainView, &crypto.DeviceIdentity{UserID: userID}, mach.DefaultSASTimeout)
	_, err := mach.NewInRoomSASVerificationWith(cmd.Room.Room.ID, userID, modal, 120*time.Second)
	if err != nil {
		cmd.Reply("Failed to start in-room verification: %v", err)
		return
	}
	cmd.MainView.ShowModal(modal)
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

func cmdImportKeys(cmd *Command) {
	path, err := filepath.Abs(cmd.RawArgs)
	if err != nil {
		cmd.Reply("Failed to get absolute path: %v", err)
		return
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		cmd.Reply("Failed to read %s: %v", path, err)
		return
	}
	passphrase, ok := cmd.MainView.AskPassword("Key import", "passphrase", "", false)
	if !ok {
		cmd.Reply("Passphrase entry cancelled")
		return
	}
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	imported, total, err := mach.ImportKeys(passphrase, data)
	if err != nil {
		cmd.Reply("Failed to import sessions: %v", err)
	} else {
		cmd.Reply("Successfully imported %d/%d sessions", imported, total)
	}
}

func exportKeys(cmd *Command, sessions []*crypto.InboundGroupSession) {
	path, err := filepath.Abs(cmd.RawArgs)
	if err != nil {
		cmd.Reply("Failed to get absolute path: %v", err)
		return
	}
	passphrase, ok := cmd.MainView.AskPassword("Key export", "passphrase", "", true)
	if !ok {
		cmd.Reply("Passphrase entry cancelled")
		return
	}
	export, err := crypto.ExportKeys(passphrase, sessions)
	if err != nil {
		cmd.Reply("Failed to export sessions: %v", err)
	}
	err = ioutil.WriteFile(path, export, 0400)
	if err != nil {
		cmd.Reply("Failed to write sessions to %s: %v", path, err)
	} else {
		cmd.Reply("Successfully exported %d sessions to %s", len(sessions), path)
	}
}

func cmdExportKeys(cmd *Command) {
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	sessions, err := mach.CryptoStore.GetAllGroupSessions()
	if err != nil {
		cmd.Reply("Failed to get sessions to export: %v", err)
		return
	}
	exportKeys(cmd, sessions)
}

func cmdExportRoomKeys(cmd *Command) {
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)
	sessions, err := mach.CryptoStore.GetGroupSessionsForRoom(cmd.Room.MxRoom().ID)
	if err != nil {
		cmd.Reply("Failed to get sessions to export: %v", err)
		return
	}
	exportKeys(cmd, sessions)
}

const ssssHelp = `Usage: /%s <subcommand> [...]

Subcommands:
* status [key ID] - Check the status of your SSSS.
* generate [--set-default] - Generate a SSSS key and optionally set it as the default.
* set-default <key ID> - Set a SSSS key as the default.`

func cmdSSSS(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply(ssssHelp, cmd.OrigCommand)
		return
	}

	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)

	switch strings.ToLower(cmd.Args[0]) {
	case "status":
		keyID := ""
		if len(cmd.Args) > 1 {
			keyID = cmd.Args[1]
		}
		cmdS4Status(cmd, mach, keyID)
	case "generate":
		setDefault := len(cmd.Args) > 1 && strings.ToLower(cmd.Args[1]) == "--set-default"
		cmdS4Generate(cmd, mach, setDefault)
	case "set-default":
		if len(cmd.Args) < 2 {
			cmd.Reply("Usage: /%s set-default <key ID>", cmd.OrigCommand)
			return
		}
		cmdS4SetDefault(cmd, mach, cmd.Args[1])
	default:
		cmd.Reply(ssssHelp, cmd.OrigCommand)
	}
}

func cmdS4Status(cmd *Command, mach *crypto.OlmMachine, keyID string) {
	var keyData *ssss.KeyMetadata
	var err error
	if len(keyID) == 0 {
		keyID, keyData, err = mach.SSSS.GetDefaultKeyData()
	} else {
		keyData, err = mach.SSSS.GetKeyData(keyID)
	}
	if errors.Is(err, ssss.ErrNoDefaultKeyAccountDataEvent) {
		cmd.Reply("SSSS is not set up: no default key set")
		return
	} else if err != nil {
		cmd.Reply("Failed to get key data: %v", err)
		return
	}
	hasPassphrase := "no"
	if keyData.Passphrase != nil {
		hasPassphrase = fmt.Sprintf("yes (alg=%s,bits=%d,iter=%d)", keyData.Passphrase.Algorithm, keyData.Passphrase.Bits, keyData.Passphrase.Iterations)
	}
	algorithm := keyData.Algorithm
	if algorithm != ssss.AlgorithmAESHMACSHA2 {
		algorithm += " (not supported!)"
	}
	cmd.Reply("Default key is set.\n  Key ID: %s\n  Has passphrase: %s\n  Algorithm: %s", keyID, hasPassphrase, algorithm)
}

func cmdS4Generate(cmd *Command, mach *crypto.OlmMachine, setDefault bool) {
	passphrase, ok := cmd.MainView.AskPassword("Passphrase", "", "", true)
	if !ok {
		return
	}

	key, err := ssss.NewKey(passphrase)
	if err != nil {
		cmd.Reply("Failed to generate new key: %v", err)
		return
	}

	err = mach.SSSS.SetKeyData(key.ID, key.Metadata)
	if err != nil {
		cmd.Reply("Failed to upload key metadata: %v", err)
		return
	}

	// TODO if we start persisting command replies, the recovery key needs to be moved into a popup
	cmd.Reply("Successfully generated key %s\nRecovery key: %s", key.ID, key.RecoveryKey())

	if setDefault {
		err = mach.SSSS.SetDefaultKeyID(key.ID)
		if err != nil {
			cmd.Reply("Failed to set key as default: %v", err)
		}
	} else {
		cmd.Reply("You can use `/%s set-default %s` to set it as the default", cmd.OrigCommand, key.ID)
	}
}

func cmdS4SetDefault(cmd *Command, mach *crypto.OlmMachine, keyID string) {
	_, err := mach.SSSS.GetKeyData(keyID)
	if err != nil {
		if errors.Is(err, mautrix.MNotFound) {
			cmd.Reply("Couldn't find key data on server")
		} else {
			cmd.Reply("Failed to fetch key data: %v", err)
		}
		return
	}

	err = mach.SSSS.SetDefaultKeyID(keyID)
	if err != nil {
		cmd.Reply("Failed to set key as default: %v", err)
	} else {
		cmd.Reply("Successfully set key %s as default", keyID)
	}
}

const crossSigningHelp = `Usage: /%s <subcommand> [...]

Subcommands:
* status
    Check the status of your own cross-signing keys.
* generate [--force]
    Generate and upload new cross-signing keys.
    This will prompt you to enter your account password.
    If you already have existing keys, --force is required.
* self-sign
    Sign the current device with cached cross-signing keys.
* fetch [--save-to-disk]
    Fetch your cross-signing keys from SSSS and decrypt them.
    If --save-to-disk is specified, the keys are saved to disk.
* upload
    Upload your cross-signing keys to SSSS.`

func cmdCrossSigning(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply(crossSigningHelp, cmd.OrigCommand)
		return
	}

	client := cmd.Matrix.Client()
	mach := cmd.Matrix.Crypto().(*crypto.OlmMachine)

	switch strings.ToLower(cmd.Args[0]) {
	case "status":
		cmdCrossSigningStatus(cmd, mach)
	case "generate":
		force := len(cmd.Args) > 1 && strings.ToLower(cmd.Args[1]) == "--force"
		cmdCrossSigningGenerate(cmd, cmd.Matrix, mach, client, force)
	case "fetch":
		saveToDisk := len(cmd.Args) > 1 && strings.ToLower(cmd.Args[1]) == "--save-to-disk"
		cmdCrossSigningFetch(cmd, mach, saveToDisk)
	case "upload":
		cmdCrossSigningUpload(cmd, mach)
	case "self-sign":
		cmdCrossSigningSelfSign(cmd, mach)
	default:
		cmd.Reply(crossSigningHelp, cmd.OrigCommand)
	}
}

func cmdCrossSigningStatus(cmd *Command, mach *crypto.OlmMachine) {
	keys := mach.GetOwnCrossSigningPublicKeys()
	if keys == nil {
		if mach.CrossSigningKeys != nil {
			cmd.Reply("Cross-signing keys are cached, but not published")
		} else {
			cmd.Reply("Didn't find published cross-signing keys")
		}
		return
	}
	if mach.CrossSigningKeys != nil {
		cmd.Reply("Cross-signing keys are published and private keys are cached")
	} else {
		cmd.Reply("Cross-signing keys are published, but private keys are not cached")
	}
	cmd.Reply("Master key: %s", keys.MasterKey)
	cmd.Reply("User signing key: %s", keys.UserSigningKey)
	cmd.Reply("Self-signing key: %s", keys.SelfSigningKey)
}

func cmdCrossSigningFetch(cmd *Command, mach *crypto.OlmMachine, saveToDisk bool) {
	key := getSSSS(cmd, mach)
	if key == nil {
		return
	}

	err := mach.FetchCrossSigningKeysFromSSSS(key)
	if err != nil {
		cmd.Reply("Error fetching cross-signing keys: %v", err)
		return
	}
	if saveToDisk {
		cmd.Reply("Saving keys to disk is not yet implemented")
	}
	cmd.Reply("Successfully unlocked cross-signing keys")
}

func cmdCrossSigningGenerate(cmd *Command, container ifc.MatrixContainer, mach *crypto.OlmMachine, client *mautrix.Client, force bool) {
	if !force {
		existingKeys := mach.GetOwnCrossSigningPublicKeys()
		if existingKeys != nil {
			cmd.Reply("Found existing cross-signing keys. Use `--force` if you want to overwrite them.")
			return
		}
	}

	keys, err := mach.GenerateCrossSigningKeys()
	if err != nil {
		cmd.Reply("Failed to generate cross-signing keys: %v", err)
		return
	}

	err = mach.PublishCrossSigningKeys(keys, func(uia *mautrix.RespUserInteractive) interface{} {
		if !uia.HasSingleStageFlow(mautrix.AuthTypePassword) {
			for _, flow := range uia.Flows {
				if len(flow.Stages) != 1 {
					return nil
				}
				cmd.Reply("Opening browser for authentication")
				err := container.UIAFallback(flow.Stages[0], uia.Session)
				if err != nil {
					cmd.Reply("Authentication failed: %v", err)
					return nil
				}
				return &mautrix.ReqUIAuthFallback{
					Session: uia.Session,
					User:    mach.Client.UserID.String(),
				}
			}
			cmd.Reply("No supported authentication mechanisms found")
			return nil
		}
		password, ok := cmd.MainView.AskPassword("Account password", "", "correct horse battery staple", false)
		if !ok {
			return nil
		}
		return &mautrix.ReqUIAuthLogin{
			BaseAuthData: mautrix.BaseAuthData{
				Type:    mautrix.AuthTypePassword,
				Session: uia.Session,
			},
			User:     mach.Client.UserID.String(),
			Password: password,
		}
	})
	if err != nil {
		cmd.Reply("Failed to publish cross-signing keys: %v", err)
		return
	}
	cmd.Reply("Successfully generated and published cross-signing keys")

	err = mach.SignOwnMasterKey()
	if err != nil {
		cmd.Reply("Failed to sign master key with device key: %v", err)
	}
}

func getSSSS(cmd *Command, mach *crypto.OlmMachine) *ssss.Key {
	_, keyData, err := mach.SSSS.GetDefaultKeyData()
	if err != nil {
		if errors.Is(err, mautrix.MNotFound) {
			cmd.Reply("SSSS not set up, use `!ssss generate --set-default` first")
		} else {
			cmd.Reply("Failed to fetch default SSSS key data: %v", err)
		}
		return nil
	}

	var key *ssss.Key
	if keyData.Passphrase != nil && keyData.Passphrase.Algorithm == ssss.PassphraseAlgorithmPBKDF2 {
		passphrase, ok := cmd.MainView.AskPassword("Passphrase", "", "correct horse battery staple", false)
		if !ok {
			return nil
		}
		key, err = keyData.VerifyPassphrase(passphrase)
		if errors.Is(err, ssss.ErrIncorrectSSSSKey) {
			cmd.Reply("Incorrect passphrase")
			return nil
		}
	} else {
		recoveryKey, ok := cmd.MainView.AskPassword("Recovery key", "", "tDAK LMRH PiYE bdzi maCe xLX5 wV6P Nmfd c5mC wLef 15Fs VVSc", false)
		if !ok {
			return nil
		}
		key, err = keyData.VerifyRecoveryKey(recoveryKey)
		if errors.Is(err, ssss.ErrInvalidRecoveryKey) {
			cmd.Reply("Malformed recovery key")
			return nil
		} else if errors.Is(err, ssss.ErrIncorrectSSSSKey) {
			cmd.Reply("Incorrect recovery key")
			return nil
		}
	}
	// All the errors should already be handled above, this is just for backup
	if err != nil {
		cmd.Reply("Failed to get SSSS key: %v", err)
		return nil
	}
	return key
}

func cmdCrossSigningUpload(cmd *Command, mach *crypto.OlmMachine) {
	if mach.CrossSigningKeys == nil {
		cmd.Reply("Cross-signing keys not cached, use `!%s generate` first", cmd.OrigCommand)
		return
	}

	key := getSSSS(cmd, mach)
	if key == nil {
		return
	}

	err := mach.UploadCrossSigningKeysToSSSS(key, mach.CrossSigningKeys)
	if err != nil {
		cmd.Reply("Failed to upload keys to SSSS: %v", err)
	} else {
		cmd.Reply("Successfully uploaded cross-signing keys to SSSS")
	}
}

func cmdCrossSigningSelfSign(cmd *Command, mach *crypto.OlmMachine) {
	if mach.CrossSigningKeys == nil {
		cmd.Reply("Cross-signing keys not cached")
		return
	}

	err := mach.SignOwnDevice(mach.OwnIdentity())
	if err != nil {
		cmd.Reply("Failed to self-sign: %v", err)
	} else {
		cmd.Reply("Successfully self-signed. This device is now trusted by other devices")
	}
}
