package mw

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AuthCookieName is the name of the authentication cookie
const AuthCookieName = "dabbi_auth"

// BearerAuth returns middleware that validates authentication via:
// 1. Cookie (preferred for browser/WebSocket)
// 2. Authorization: Bearer header (for API clients)
func BearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check cookie first (works for both regular requests and WebSocket)
			if cookie, err := r.Cookie(AuthCookieName); err == nil {
				if cookie.Value == token {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Fall back to Authorization header for API clients
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error": "Invalid Authorization header format"}`, http.StatusUnauthorized)
				return
			}

			if parts[1] != token {
				http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoginHandler returns a handler that validates token and sets auth cookie.
// This endpoint is NOT protected by auth middleware.
func LoginHandler(token string, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Token != token {
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Set HttpOnly cookie - not accessible via JavaScript
		http.SetCookie(w, &http.Cookie{
			Name:     AuthCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   secureCookie, // true when using HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400 * 30, // 30 days
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// LogoutHandler returns a handler that clears the auth cookie.
func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Clear cookie by setting MaxAge to -1
		http.SetCookie(w, &http.Cookie{
			Name:     AuthCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
