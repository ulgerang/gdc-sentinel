// Package idgen provides collision-resistant ID generation for sentinel records.
//
// IDs are 16-character lowercase hex strings drawn from crypto/rand. The
// keyspace (2^64) makes accidental collision across the lifetime of a project
// negligible without adding a dependency on a UUID library.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// New returns a 16-hex-char ID drawn from crypto/rand. It returns an error
// only when the system entropy source fails, which is treated as fatal by
// the caller (no fallback to time-based IDs, which can collide).
func New() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("read crypto/rand: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
