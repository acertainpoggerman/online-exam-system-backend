package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
)

func IsAuthenticated(secretKey string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authParts := strings.Split(r.Header.Get("Authorization"), "Bearer ")
			if len(authParts) != 2 {
				slog.Error("Wrong Format")
				api.UnauthorizedResponse(w, r)
				return
			}

			tokenString := authParts[1]
			userData, err := tools.ValidateJWT(tokenString, []byte(secretKey))
			if err != nil {
				slog.Error("Error Validating JWT", "error", err)
				api.UnauthorizedResponse(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), api.AuthUserData, userData)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
