package tools

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserData struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (d *UserData) ToMap() map[string]any {
	return map[string]any{
		"id":    d.ID,
		"email": d.Email,
		"role":  d.Role,
	}
}

type JWTClaims struct {
	Data UserData `json:"user_data"`
	jwt.RegisteredClaims
}

// Creates a JWT with given user details a
// token expiration time, and a symmetric secret key.
// Returns an error if the token fails to be signed.
func CreateJWT(
	// User Data
	userData UserData,
	// JWT Params
	secretKey []byte,
	expiryTime time.Duration,
) (string, error) {

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			// Standard Claims
			"iss": "online-exam-server",
			"exp": time.Now().Add(expiryTime).Unix(),
			// User Claims
			"user_data": userData.ToMap(),
		},
	)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Checks if the given token is valid.
// Will return the user data attached if valid,
// returning nothing along with an error otherwise.
func ValidateJWT(tokenString string, secretKey []byte) (*UserData, error) {

	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},

		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secretKey, nil
		},
	)
	if err != nil {

		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	userData := token.Claims.(*JWTClaims).Data
	return &userData, nil
}
