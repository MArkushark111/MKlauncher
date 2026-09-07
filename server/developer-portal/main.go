package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var sqlDB *sql.DB

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" { w.WriteHeader(200); return }
		next(w, r)
	}
}

func auth(next http.HandlerFunc) http.HandlerFunc {
	return cors(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " { token = token[7:] }
		var userID int
		err := sqlDB.QueryRow("SELECT user_id FROM user_tokens WHERE token=?", token).Scan(&userID)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized"}`, 401)
			return
		}
		r.Header.Set("X-User-ID", strconv.Itoa(userID))
		next(w, r)
	})
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Password string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"Username and password required"}`, 400)
		return
	}
	var id int
	var passHash, displayName string
	err := sqlDB.QueryRow("SELECT id, password_hash, display_name FROM users WHERE username=?", req.Username).Scan(&id, &passHash, &displayName)
	if err != nil {
		http.Error(w, `{"error":"Invalid credentials"}`, 401)
		return
	}
	if passHash != req.Password {
		http.Error(w, `{"error":"Invalid credentials"}`, 401)
		return
	}
	var token string
	err = sqlDB.QueryRow("SELECT token FROM user_tokens WHERE user_id=? ORDER BY id DESC LIMIT 1", id).Scan(&token)
	if err != nil {
		t := fmt.Sprintf("%x", id*1000000)
		sqlDB.Exec("INSERT INTO user_tokens (user_id, token) VALUES (?, ?)", id, t)
		token = t
	}
	jsonResp(w, map[string]interface{}{
		"success": true, "token": token,
		"user": map[string]interface{}{"id": id, "username": req.Username, "display_name": displayName},
	})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Password, DisplayName string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"Username and password required"}`, 400)
		return
	}
	if req.DisplayName == "" { req.DisplayName = req.Username }
	_, err := sqlDB.Exec("INSERT INTO users (username, password_hash, display_name) VALUES (?, ?, ?)",
		req.Username, req.Password, req.DisplayName)
	if err != nil {
		http.Error(w, `{"error":"Username already taken"}`, 400)
		return
	}
	var id int
	sqlDB.QueryRow("SELECT id FROM users WHERE username=?", req.Username).Scan(&id)
	token := fmt.Sprintf("%x", id*1000000+1)
	sqlDB.Exec("INSERT INTO user_tokens (user_id, token) VALUES (?, ?)", id, token)
	jsonResp(w, map[string]interface{}{
		"success": true, "token": token,
		"user": map[string]interface{}{"id": id, "username": req.Username, "display_name": req.DisplayName},
	})
}

func handleGetGames(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	rows, err := sqlDB.Query(`SELECT g.id, g.name, g.version, g.exe_path, g.description, g.is_public, g.downloads, g.created_at
		FROM games g WHERE g.developer_id=? ORDER BY g.created_at DESC`, userID)
	if err != nil {
		jsonResp(w, []interface{}{})
		return
	}
	defer rows.Close()
	var games []map[string]interface{}
	for rows.Next() {
		var id, downloads int
		var name, version, exePath, description, createdAt string
		var isPublic bool
		if err := rows.Scan(&id, &name, &version, &exePath, &description, &isPublic, &downloads, &createdAt); err != nil { continue }
		games = append(games, map[string]interface{}{
			"id": id, "name": name, "version": version, "exe_path": exePath,
			"description": description, "is_public": isPublic, "downloads": downloads, "created_at": createdAt,
		})
	}
	jsonResp(w, games)
}

func handleGetDevApps(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	rows, err := sqlDB.Query("SELECT id, app_name, api_key, created_at FROM developer_apps WHERE user_id=?", userID)
	if err != nil {
		jsonResp(w, []interface{}{})
		return
	}
	defer rows.Close()
	var apps []map[string]interface{}
	for rows.Next() {
		var id int
		var appName, apiKey, createdAt string
		if err := rows.Scan(&id, &appName, &apiKey, &createdAt); err != nil { continue }
		apps = append(apps, map[string]interface{}{
			"id": id, "app_name": appName, "api_key": apiKey, "created_at": createdAt,
		})
	}
	jsonResp(w, apps)
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var totalGames, totalDownloads, publicGames int
	sqlDB.QueryRow("SELECT COUNT(*), COALESCE(SUM(downloads),0), SUM(CASE WHEN is_public THEN 1 ELSE 0 END) FROM games WHERE developer_id=?", userID).Scan(&totalGames, &totalDownloads, &publicGames)
	jsonResp(w, map[string]interface{}{
		"total_games": totalGames, "total_downloads": totalDownloads, "public_games": publicGames,
	})
}

func handleCreateApp(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var req struct{ AppName string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.AppName == "" {
		http.Error(w, `{"error":"App name required"}`, 400)
		return
	}
	apiKey, err := sqlDB.Exec("INSERT INTO developer_apps (user_id, app_name, api_key) VALUES (?, ?, ?)",
		userID, req.AppName, "")
	if err != nil {
		http.Error(w, `{"error":"Failed to create app"}`, 500)
		return
	}
	_ = apiKey
	jsonResp(w, map[string]interface{}{"success": true})
}

func handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	parts := strings.Split(r.URL.Path, "/")
	id, _ := strconv.Atoi(parts[len(parts)-1])
	_, err := sqlDB.Exec("DELETE FROM developer_apps WHERE id=? AND user_id=?", id, userID)
	if err != nil {
		http.Error(w, `{"error":"Not found"}`, 404)
		return
	}
	jsonResp(w, map[string]interface{}{"success": true})
}

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var username, displayName, avatarURL string
	var isBanned bool
	err := sqlDB.QueryRow("SELECT username, display_name, COALESCE(avatar_url,''), is_banned FROM users WHERE id=?", userID).Scan(&username, &displayName, &avatarURL, &isBanned)
	if err != nil {
		http.Error(w, `{"error":"Not found"}`, 404)
		return
	}
	jsonResp(w, map[string]interface{}{
		"username": username, "display_name": displayName, "avatar_url": avatarURL, "is_banned": isBanned,
	})
}

func main() {
	dbPath := "/var/lib/mkgames/mkgames.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatal("Database not found. Run main server first.")
	}
	var err error
	sqlDB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil { log.Fatal(err) }
	defer sqlDB.Close()

	http.HandleFunc("/api/auth/login", cors(handleLogin))
	http.HandleFunc("/api/auth/register", cors(handleRegister))
	http.HandleFunc("/api/developer/games", auth(handleGetGames))
	http.HandleFunc("/api/developer/stats", auth(handleGetStats))
	http.HandleFunc("/api/developer/profile", auth(handleGetProfile))
	http.HandleFunc("/api/developer/apps", auth(handleGetDevApps))
	http.HandleFunc("/api/developer/app", auth(handleCreateApp))
	http.HandleFunc("/api/developer/app/delete/", auth(handleDeleteApp))

	http.Handle("/", http.FileServer(http.Dir("web")))

	log.Println("[DEV-PORTAL] Listening on :8787")
	log.Fatal(http.ListenAndServe(":8787", nil))
}
