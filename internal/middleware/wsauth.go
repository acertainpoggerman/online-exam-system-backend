package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
)

func WebsocketJWTAuth(secretKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			auth := r.Header.Get("Sec-WebSocket-Protocol")
			if auth == "" {
				jwtAuthFailed(w, "No Sec-WebSocket-Protocol value found")
				return
			}

			const prefix = "base64url.bearer.authorization.oes."
			const suffix = ", base64.binary.oes"
			token64, found := strings.CutPrefix(auth, prefix)
			if !found {
				jwtAuthFailed(w, "Invalid Authorization header value format")
				return
			}

			token64, found = strings.CutSuffix(token64, suffix)
			if !found {
				jwtAuthFailed(w, "Invalid Authorization header value format")
				return
			}

			tokenBytes, err := base64.RawStdEncoding.DecodeString(token64)
			if err != nil {
				jwtAuthFailed(w, err.Error())
				return
			}

			userData, err := jwt.ValidateJWT(string(tokenBytes), []byte(secretKey))
			if err != nil {
				jwtAuthFailed(w, "Failed to validate JWT token")
				return
			}

			r.Header.Set("Sec-Websocket-Protocol", "base64.binary.oes")

			ctx := context.WithValue(r.Context(), jwt.UserKey, userData)
			finalReq := r.WithContext(ctx)

			next.ServeHTTP(w, finalReq)
		})
	}
}
