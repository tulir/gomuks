package headless

import (
	"context"
	"errors"
	"fmt"
	"os"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/ssss"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/initialize"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
)

type HeadlessConfig struct {
	OutputDir, MxPassword, KeyPath, KeyPassword, RecoveryPhrase string
	MxID                                                        id.UserID
}

func HeadlessInit(conf HeadlessConfig, updates chan fmt.Stringer) error {
	// setup package dir
	os.Setenv("GOMUKS_ROOT", conf.OutputDir)
	updates <- ExportDirSetMsg{dir: conf.OutputDir}

	// init boilerplate
	configDir, dataDir, cacheDir, downloadDir, err := initDirs()
	if err != nil {
		return err
	}

	gmx := initialize.NewGomuks(ui.NewGomuksUI, configDir, dataDir, cacheDir, downloadDir)
	err = gmx.StartHeadless()
	if err != nil {
		return err
	}
	updates <- InitializedGomuksMsg{}

	// login section
	_, hs, err := conf.MxID.Parse()
	if err != nil {
		return err
	}

	gmx.Config().HS = hs
	if err := gmx.Matrix().InitClient(false); err != nil {
		return err
	} else if err = gmx.Matrix().Login(conf.MxID.String(), conf.MxPassword); err != nil {
		return err
	}
	updates <- LoggedInMsg{account: conf.MxID}

	// key import
	data, err := os.ReadFile(conf.KeyPath)
	if err != nil {
		return err
	}
	mach := gmx.Matrix().Crypto().(*crypto.OlmMachine)
	imported, total, err := mach.ImportKeys(conf.KeyPassword, data)
	if err != nil {
		return fmt.Errorf("Failed to import sessions: %v", err)
	}
	updates <- ImportedKeysMsg{imported: imported, total: total}

	// verify (fetch)
	key, err := getSSSS(mach, conf.RecoveryPhrase)
	if err != nil {
		return err
	}

	err = mach.FetchCrossSigningKeysFromSSSS(key)
	if err != nil {
		return fmt.Errorf("Error fetching cross-signing keys: %v", err)
	}
	updates <- FetchedVerificationKeysMsg{}

	// verify (sign)
	if mach.CrossSigningKeys == nil {
		return fmt.Errorf("Cross-signing keys not cached")
	}

	err = mach.SignOwnDevice(mach.OwnIdentity())
	if err != nil {
		return fmt.Errorf("Failed to self-sign: %v", err)
	}
	updates <- SuccessfullyVerifiedMsg{}

	// display mode
	gmx.Config().Preferences.DisplayMode = config.DisplayModeModern
	updates <- ConfiguredDisplayModeMsg{}

	// sync
	updates <- BeginningSyncMsg{}
	resp, err := gmx.Matrix().Client().FullSyncRequest(mautrix.ReqSync{
		Timeout:        30000,
		Since:          "",
		FilterID:       "",
		FullState:      true,
		SetPresence:    gmx.Matrix().Client().SyncPresence,
		Context:        context.Background(),
		StreamResponse: true,
	})
	if err != nil {
		return err
	}
	updates <- FetchedSyncDataMsg{}

	gmx.Matrix().(*matrix.Container).InitSyncer()
	updates <- ProcessingSyncMsg{}
	err = gmx.Matrix().(*matrix.Container).ProcessSyncResponse(resp, "")

	return err
}

func initDirs() (string, string, string, string, error) {
	config, err := initialize.UserConfigDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("Failed to get config directory: %v", err)
	}

	data, err := initialize.UserDataDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("Failed to get data directory: %v", err)
	}

	cache, err := initialize.UserCacheDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("Failed to get cache directory: %v", err)
	}

	download, err := initialize.UserDownloadDir()
	if err != nil {
		return "", "", "", "", fmt.Errorf("Failed to get download directory: %v", err)
	}

	return config, data, cache, download, nil
}

func getSSSS(mach *crypto.OlmMachine, recoveryPhrase string) (*ssss.Key, error) {
	_, keyData, err := mach.SSSS.GetDefaultKeyData()
	if err != nil {
		if errors.Is(err, mautrix.MNotFound) {
			return nil, fmt.Errorf("SSSS not set up, use `!ssss generate --set-default` first")
		} else {
			return nil, fmt.Errorf("Failed to fetch default SSSS key data: %v", err)
		}
	}

	var key *ssss.Key
	if keyData.Passphrase != nil && keyData.Passphrase.Algorithm == ssss.PassphraseAlgorithmPBKDF2 {
		key, err = keyData.VerifyPassphrase(recoveryPhrase)
		if errors.Is(err, ssss.ErrIncorrectSSSSKey) {
			return nil, fmt.Errorf("Incorrect passphrase")
		}
	} else {
		key, err = keyData.VerifyRecoveryKey(recoveryPhrase)
		if errors.Is(err, ssss.ErrInvalidRecoveryKey) {
			return nil, fmt.Errorf("Malformed recovery key")
		} else if errors.Is(err, ssss.ErrIncorrectSSSSKey) {
			return nil, fmt.Errorf("Incorrect recovery key")
		}
	}
	// All the errors should already be handled above, this is just for backup
	if err != nil {
		return nil, fmt.Errorf("Failed to get SSSS key: %v", err)
	}
	return key, nil
}
