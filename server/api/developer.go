package api

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func DevAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		user, err := db.DB.ValidateUserToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}
		r.Header.Set("X-User-ID", strconv.Itoa(user.ID))
		next(w, r)
	}
}

func HandleDevLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	if req.Username == "" || req.Password == "" {
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
	if user.PasswordHash != req.Password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
		return
	}
	token, err := db.DB.GenerateUserToken(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    map[string]interface{}{"id": user.ID, "username": user.Username, "display_name": user.DisplayName},
	})
}

func HandleDevRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username and password required"})
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}
	user, err := db.DB.RegisterUser(req.Username, req.Password, req.DisplayName)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	token, _ := db.DB.GenerateUserToken(user.ID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
		"user":    map[string]interface{}{"id": user.ID, "username": user.Username, "display_name": user.DisplayName},
	})
}

func HandleDevGetGames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	rows, err := db.DB.Conn.Query(`SELECT id, name, version, exe_path, description, is_public, downloads, created_at
		FROM games WHERE developer_id=? ORDER BY created_at DESC`, userID)
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	defer rows.Close()
	var games []map[string]interface{}
	for rows.Next() {
		var id, downloads int
		var name, version, exePath, description, createdAt string
		var isPublic bool
		if err := rows.Scan(&id, &name, &version, &exePath, &description, &isPublic, &downloads, &createdAt); err != nil {
			continue
		}
		games = append(games, map[string]interface{}{
			"id": id, "name": name, "version": version, "exe_path": exePath,
			"description": description, "is_public": isPublic, "downloads": downloads, "created_at": createdAt,
		})
	}
	json.NewEncoder(w).Encode(games)
}

func HandleDevGetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var totalGames, totalDownloads, publicGames int
	db.DB.Conn.QueryRow(`SELECT COUNT(*), COALESCE(SUM(downloads),0), SUM(CASE WHEN is_public THEN 1 ELSE 0 END)
		FROM games WHERE developer_id=?`, userID).Scan(&totalGames, &totalDownloads, &publicGames)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_games": totalGames, "total_downloads": totalDownloads, "public_games": publicGames,
	})
}

func HandleDevGetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	user, err := db.DB.GetUserByID(userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username": user.Username, "display_name": user.DisplayName,
		"avatar_url": user.AvatarURL, "is_banned": user.IsBanned,
	})
}

func HandleDevGetApps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	apps, err := db.DB.ListDeveloperApps(userID)
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	json.NewEncoder(w).Encode(apps)
}

func HandleDevCreateApp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var req struct {
		AppName string `json:"app_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AppName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "App name required"})
		return
	}
	apiKey, err := db.DB.CreateDeveloperApp(userID, req.AppName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create app"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "api_key": apiKey})
}

func HandleDevDeleteApp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)
	if err := db.DB.DeleteDeveloperApp(id, userID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleDevUploadArchive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	gameIDStr := mux.Vars(r)["id"]
	gameID, _ := strconv.Atoi(gameIDStr)

	var game db.Game
	err := db.DB.Conn.QueryRow("SELECT id, name, developer_id FROM games WHERE id=?", gameID).Scan(&game.ID, &game.Name, &game.DeveloperID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game not found"})
		return
	}
	if game.DeveloperID != userID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not your game"})
		return
	}

	if err := r.ParseMultipartForm(4096 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid form data"})
		return
	}

	file, fh, err := r.FormFile("archive")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	archivePath, written, err := saveFolderArchive(game.Name, r.FormValue("version"), []*multipart.FileHeader{fh})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	config, _ := db.DB.GetConfig()
	if config != nil && config.WANHost != "" && config.WANPort != "" {
		archivePath = fmt.Sprintf("http://%s:%s/%s", config.WANHost, config.WANPort, archivePath)
	}

	db.DB.Conn.Exec("UPDATE games SET archive_path=?, file_size=?, version=COALESCE(NULLIF(?,''), version) WHERE id=?",
		archivePath, written, r.FormValue("version"), gameID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "archive_path": archivePath, "file_size": written,
	})
}
