package headless

import (
	"errors"
	"fmt"
	"os"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/ssss"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/initialize"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/ui"
)

type Config struct {
	OutputDir, Session, Code, KeyPath, KeyPassword, RecoveryCode string
}

func Init(conf Config, updates chan fmt.Stringer) error {
	defer close(updates)

	// setup package dir
	os.Setenv("GOMUKS_ROOT", conf.OutputDir)
	updates <- exportDirSet{dir: conf.OutputDir}

	// init boilerplate
	configDir, dataDir, cacheDir, downloadDir, err := initDirs()
	if err != nil {
		return err
	}

	gmx := initialize.NewGomuks(ui.NewGomuksUI, configDir, dataDir, cacheDir, downloadDir)
	gmx.Matrix().(*matrix.Container).SetHeadless()
	err = gmx.StartHeadless()
	if err != nil {
		return err
	}
	updates <- initializedGomuks{}

	// login section
	gmx.Config().HS = "https://matrix.beeper.com"
	if err := gmx.Matrix().InitClient(false); err != nil {
		return err
	} else if err = gmx.Matrix().BeeperLogin(conf.Session, conf.Code); err != nil {
		return err
	}
	updates <- loggedIn{}

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
	updates <- importedKeys{imported: imported, total: total}

	// display mode
	gmx.Config().Preferences.DisplayMode = config.DisplayModeModern
	updates <- configuredDisplayMode{}

	// sync
	updates <- creatingSyncer{}
	gmx.Matrix().(*matrix.Container).InitSyncer()
	updates <- synchronizing{}
	err = gmx.Matrix().Client().Sync()
	if err != nil {
		return err
	}
	updates <- syncFinished{}

	// verify (fetch)
	key, err := getSSSS(mach, conf.RecoveryCode)
	if err != nil {
		return err
	}

	err = mach.FetchCrossSigningKeysFromSSSS(key)
	if err != nil {
		return fmt.Errorf("Error fetching cross-signing keys: %v", err)
	}
	updates <- fetchedVerificationKeys{}

	// verify (sign)
	if mach.CrossSigningKeys == nil {
		return fmt.Errorf("Cross-signing keys not cached")
	}

	err = mach.SignOwnDevice(mach.OwnIdentity())
	if err != nil {
		return fmt.Errorf("Failed to self-sign: %v", err)
	}
	updates <- successfullyVerified{}

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

func getSSSS(mach *crypto.OlmMachine, recoveryCode string) (*ssss.Key, error) {
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
		key, err = keyData.VerifyPassphrase(recoveryCode)
		if errors.Is(err, ssss.ErrIncorrectSSSSKey) {
			return nil, fmt.Errorf("Incorrect passphrase")
		}
	} else {
		key, err = keyData.VerifyRecoveryKey(recoveryCode)
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
