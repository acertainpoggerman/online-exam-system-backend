package passwords

import (
	"testing"
)

// TestVerifyPasswordHash generates a hash for a
// password, then attempts to correctly verify the
// password against the hash using the VerifyPassword
// function
func TestVerifyPasswordHash(t *testing.T) {
	originalPassword := "This is the password"
	passwordHash, err := GeneratePasswordHash(originalPassword)
	if err != nil {
		t.Errorf(`GeneratePasswordHash("%v") failed: %v`, originalPassword, err)
	}

	if !VerifyPassword(originalPassword, passwordHash) {
		t.Errorf(`VerifyPassword("%v", "%v") did not match password with hash`, originalPassword, passwordHash)
	}
}
