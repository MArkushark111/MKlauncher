package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"mkgames-server/db"
)

var (
	authTokens   = make(map[string]time.Time)
	authMutex    sync.RWMutex
	TokenExpiry  = 24 * time.Hour
	authAttempts = make(map[string][]time.Time)
	attemptMutex sync.Mutex
)

type AuthRequest struct {
	Passcode string `json:"passcode"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Error   string `json:"error,omitempty"`
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b)
}

func authClientKey(r *http.Request) string {
	client, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return client
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func allowAuthAttempt(key string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	attemptMutex.Lock()
	defer attemptMutex.Unlock()

	attempts := authAttempts[key][:0]
	for _, attempt := range authAttempts[key] {
		if attempt.After(cutoff) {
			attempts = append(attempts, attempt)
		}
	}
	if len(attempts) >= 5 {
		authAttempts[key] = attempts
		return false
	}
	authAttempts[key] = append(attempts, now)
	return true
}

func HandleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(AuthResponse{Error: "Method not allowed"})
		return
	}
	if !allowAuthAttempt(authClientKey(r)) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(AuthResponse{Error: "Too many attempts; try again later"})
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(AuthResponse{Error: "Invalid request body"})
		return
	}

	if req.Passcode == "" {
		json.NewEncoder(w).Encode(AuthResponse{Error: "Passcode required"})
		return
	}

	if !db.DB.VerifyPasscode(req.Passcode) {
		json.NewEncoder(w).Encode(AuthResponse{Error: "Invalid passcode"})
		return
	}

	token := generateToken()
	if token == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AuthResponse{Error: "Failed to create session"})
		return
	}
	authMutex.Lock()
	authTokens[token] = time.Now().Add(TokenExpiry)
	authMutex.Unlock()

	json.NewEncoder(w).Encode(AuthResponse{Success: true, Token: token})
}

func ValidateToken(token string) bool {
	if token == "" {
		return false
	}
	authMutex.Lock()
	defer authMutex.Unlock()

	expiry, exists := authTokens[token]
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		delete(authTokens, token)
		return false
	}
	return true
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		if ValidateToken(token) {
			var adminID int
			db.DB.Conn.QueryRow("SELECT user_id FROM user_tokens WHERE token=?", token).Scan(&adminID)
			if adminID > 0 {
				r.Header.Set("X-User-ID", fmt.Sprintf("%d", adminID))
			}
			next(w, r)
			return
		}

		user, err := db.DB.ValidateUserToken(token)
		if err == nil && user != nil {
			r.Header.Set("X-User-ID", fmt.Sprintf("%d", user.ID))
			next(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
	}
}

func CleanupTokens() {
	for {
		time.Sleep(1 * time.Hour)
		authMutex.Lock()
		now := time.Now()
		for token, expiry := range authTokens {
			if now.After(expiry) {
				delete(authTokens, token)
			}
		}
		authMutex.Unlock()
	}
}
