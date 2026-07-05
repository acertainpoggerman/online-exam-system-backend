package jwt

import (
	"context"
	"fmt"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/go-viper/mapstructure/v2"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserKey contextKey = "user"

func GetUserDataFromContext(ctx context.Context) (store.User, error) {
	var user store.User
	if err := mapstructure.Decode(ctx.Value(UserKey), &user); err != nil {
		return store.User{}, err
	}
	return user, nil
}

func toMap(user store.User) map[string]any {
	return map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
}

type JWTClaims struct {
	User store.User `json:"user"`
	jwt.RegisteredClaims
}

// Creates a JWT with given user details a
// token expiration time, and a symmetric secret key.
// Returns an error if the token fails to be signed.
func CreateJWT(user store.User, secretKey []byte, expiryTime time.Duration) (string, error) {

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			// Standard Claims
			"iss": "online-exam-server",
			"exp": time.Now().Add(expiryTime).Unix(),
			// User Claims
			"user": toMap(user),
		},
	)

	return token.SignedString(secretKey)
}

// Checks if the given token is valid.
// Will return the user data attached if valid,
// returning nothing along with an error otherwise.
func ValidateJWT(tokenString string, secretKey []byte) (*store.User, error) {

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

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("Failed to convert claims")
	}

	return &claims.User, nil
}
