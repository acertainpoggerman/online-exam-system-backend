package passwords

import "golang.org/x/crypto/bcrypt"

// Generates and returns a hash given as password as input
// using bcrypt algorithm
func GeneratePasswordHash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// Verifies a password given the password and a hash
func VerifyPassword(password string, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	return err == nil
}
