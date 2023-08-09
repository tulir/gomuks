package headless

import "fmt"

type ExportDirSetMsg struct{ dir string }

func (msg ExportDirSetMsg) String() string {
	return fmt.Sprintf("Set gomuks root directory to %s…", msg.dir)
}

type InitializedGomuksMsg struct{}

func (msg InitializedGomuksMsg) String() string {
	return "Initialized gomuks…"
}

type LoggedInMsg struct{ account fmt.Stringer }

func (msg LoggedInMsg) String() string {
	return fmt.Sprintf("Logged in to %s…", msg.account)
}

type ImportedKeysMsg struct{ imported, total int }

func (msg ImportedKeysMsg) String() string {
	return fmt.Sprintf("Successfully imported %d/%d sessions", msg.imported, msg.total)
}

type FetchedVerificationKeysMsg struct{}

func (msg FetchedVerificationKeysMsg) String() string {
	return "Successfully unlocked cross-signing keys…"
}

type SuccessfullyVerifiedMsg struct{}

func (msg SuccessfullyVerifiedMsg) String() string {
	return "Successfully self-signed. This device is now trusted by other devices…"
}

type ConfiguredDisplayModeMsg struct{}

func (msg ConfiguredDisplayModeMsg) String() string {
	return "Configured display mode…"
}

type BeginningSyncMsg struct{}

func (msg BeginningSyncMsg) String() string {
	return "Beginning the sync process…"
}

type FetchedSyncDataMsg struct{}

func (msg FetchedSyncDataMsg) String() string {
	return "Fetched sync data…"
}

type ProcessingSyncMsg struct{}

func (msg ProcessingSyncMsg) String() string {
	return "Processing sync response…"
}
