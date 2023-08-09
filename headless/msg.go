package headless

import "fmt"

type exportDirSet struct{ dir string }

func (msg exportDirSet) String() string {
	return fmt.Sprintf("Set gomuks root directory to %s…", msg.dir)
}

type initializedGomuks struct{}

func (msg initializedGomuks) String() string {
	return "Initialized gomuks…"
}

type loggedIn struct{ account fmt.Stringer }

func (msg loggedIn) String() string {
	return fmt.Sprintf("Logged in to %s…", msg.account)
}

type importedKeys struct{ imported, total int }

func (msg importedKeys) String() string {
	return fmt.Sprintf("Successfully imported %d/%d sessions", msg.imported, msg.total)
}

type fetchedVerificationKeys struct{}

func (msg fetchedVerificationKeys) String() string {
	return "Successfully unlocked cross-signing keys…"
}

type successfullyVerified struct{}

func (msg successfullyVerified) String() string {
	return "Successfully self-signed. This device is now trusted by other devices…"
}

type configuredDisplayMode struct{}

func (msg configuredDisplayMode) String() string {
	return "Configured display mode…"
}

type beginningSync struct{}

func (msg beginningSync) String() string {
	return "Beginning the sync process…"
}

type fetchedSyncData struct{}

func (msg fetchedSyncData) String() string {
	return "Fetched sync data…"
}

type processingSync struct{}

func (msg processingSync) String() string {
	return "Processing sync response…"
}
