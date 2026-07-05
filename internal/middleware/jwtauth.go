package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
)

func JWTAuth(secretKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var tokenString string

			// ----------------------------------------------------
			// --- [Choice 1] Check Header ------------------------

			auth := r.Header.Get("Authorization")

			if auth != "" {

				const prefix string = "Bearer "

				var ok bool
				tokenString, ok = strings.CutPrefix(auth, prefix)
				if !ok {
					jwtAuthFailed(w, "Invalid Authorization header value format")
					return
				}

			} else {

				// ----------------------------------------------------
				// --- [Choice 2] Check Cookie ------------------------

				cookie, err := r.Cookie("token")
				if err != nil {
					jwtAuthFailed(w, "No cookie \"token\" found")
					return
				}

				tokenString = cookie.Value
			}

			userData, err := jwt.ValidateJWT(tokenString, []byte(secretKey))
			if err != nil {
				jwtAuthFailed(w, "Failed to validate JWT token")
				return
			}

			ctx := context.WithValue(r.Context(), jwt.UserKey, userData)
			finalReq := r.WithContext(ctx)

			next.ServeHTTP(w, finalReq)
		})
	}
}

func jwtAuthFailed(w http.ResponseWriter, msg string) {
	log.Println(msg)
	json.WriteJSON(w, http.StatusUnauthorized, msg, nil)
}
