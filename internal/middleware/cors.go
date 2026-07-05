package middleware

import (
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			json.WriteJSON(w, http.StatusNoContent, nil, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
