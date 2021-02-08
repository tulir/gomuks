// This contains no-op stubs of the methods in crypto.go for non-cgo builds with crypto disabled.

// +build !cgo

package matrix

func isBadEncryptError(err error) bool {
	return false
}

func (c *Container) initCrypto() error {
	return nil
}

func (c *Container) cryptoOnLogin() {}
