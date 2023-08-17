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

type configuredDisplayMode struct{}

func (msg configuredDisplayMode) String() string {
	return "Configured display mode…"
}

type creatingSyncer struct{}

func (msg creatingSyncer) String() string {
	return "Initializing sync utilities…"
}

type synchronizing struct{}

func (msg synchronizing) String() string {
	return "Synchronizing…"
}

type syncFinished struct{}

func (msg syncFinished) String() string {
	return "Sync completed…"
}

type fetchedVerificationKeys struct{}

func (msg fetchedVerificationKeys) String() string {
	return "Successfully unlocked cross-signing keys…"
}

type successfullyVerified struct{}

func (msg successfullyVerified) String() string {
	return "Successfully self-signed. This device is now trusted by other devices…"
}
