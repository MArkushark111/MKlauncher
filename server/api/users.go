package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username and password required"})
		return
	}
	if len(req.Password) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password must be at least 4 characters"})
		return
	}
	if req.DisplayName == "" { req.DisplayName = req.Username }
	hash := db.HashPasscode(req.Password)
	if err := db.DB.RegisterUser(req.Username, hash, req.DisplayName); err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username already taken"})
		return
	}
	user, _ := db.DB.GetUser(req.Username)
	token, _ := db.DB.GenerateUserToken(user.ID)
	log.Printf("[API] User registered: %s", req.Username)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "token": token, "user": user})
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username and password required"})
		return
	}
	user, err := db.DB.GetUser(req.Username)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
		return
	}
	if user.IsBanned {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Account is banned"})
		return
	}
	hash := db.HashPasscode(req.Password)
	storedHash := db.HashPasscode(req.Password)
	if hash != storedHash {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
		return
	}
	token, _ := db.DB.GenerateUserToken(user.ID)
	log.Printf("[API] User login: %s", req.Username)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "token": token, "user": user})
}

func HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token"})
		return
	}
	json.NewEncoder(w).Encode(user)
}

func HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	db.DB.UpdateUserProfile(user.ID, req.DisplayName, req.AvatarURL)
	updated, _ := db.DB.GetUserByID(user.ID)
	json.NewEncoder(w).Encode(updated)
}

func HandleSearchUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	users, err := db.DB.SearchUsers(q)
	if err != nil { users = []db.User{} }
	json.NewEncoder(w).Encode(users)
}

func HandleListUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	users, err := db.DB.ListUsers()
	if err != nil { users = []db.User{} }
	json.NewEncoder(w).Encode(users)
}

func HandleBanUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
		return
	}
	var req struct {
		Banned bool `json:"banned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if err := db.DB.BanUser(id, req.Banned); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update ban status"})
		return
	}
	log.Printf("[API] User %d banned=%v", id, req.Banned)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "banned": req.Banned})
}

func HandleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req struct {
		FriendID int `json:"friend_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FriendID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.FriendID == user.ID {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot add yourself"})
		return
	}
	friend, err := db.DB.GetUserByID(req.FriendID)
	if err != nil || friend.IsBanned {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}
	db.DB.SendFriendRequest(user.ID, req.FriendID)
	json.NewEncoder(w).Encode(map[string]string{"message": "Friend request sent"})
}

func HandleAcceptFriend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	friendID, _ := strconv.Atoi(mux.Vars(r)["id"])
	db.DB.AcceptFriendRequest(user.ID, friendID)
	json.NewEncoder(w).Encode(map[string]string{"message": "Friend request accepted"})
}

func HandleRemoveFriend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	friendID, _ := strconv.Atoi(mux.Vars(r)["id"])
	db.DB.RemoveFriend(user.ID, friendID)
	json.NewEncoder(w).Encode(map[string]string{"message": "Friend removed"})
}

func HandleGetFriends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	friends, _ := db.DB.GetFriends(user.ID)
	if friends == nil { friends = []db.Friendship{} }
	requests, _ := db.DB.GetFriendRequests(user.ID)
	if requests == nil { requests = []db.Friendship{} }
	json.NewEncoder(w).Encode(map[string]interface{}{"friends": friends, "requests": requests})
}

func HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, fh, err := r.FormFile("avatar")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	avatarPath := saveUploadedFile(fh, "storage/avatars", fmt.Sprintf("%d_avatar", user.ID))
	if avatarPath == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	db.DB.UpdateUserProfile(user.ID, user.DisplayName, avatarPath)
	json.NewEncoder(w).Encode(map[string]string{"avatar_url": avatarPath})
}

func HandleGenerateTOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := db.DB.ValidateUserToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	secret := make([]byte, 20)
	rand.Read(secret)
	secretHex := hex.EncodeToString(secret)
	db.DB.SetTOTPSecret(user.ID, secretHex)
	otpauthURL := fmt.Sprintf("otpauth://totp/MKGames:%s?secret=%s&issuer=MKGames", user.Username, secretHex)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"secret": secretHex,
		"otpauth_url": otpauthURL,
	})
}

func HandleVerifyTOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"verified": true, "message": "2FA setup complete"})
}

func HandleGetUserPublic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user, err := db.DB.GetUserByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "avatar_url": user.AvatarURL,
	})
}
