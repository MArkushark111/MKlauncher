package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.DB.Conn.Query("SELECT id, name, icon, sort_order FROM categories ORDER BY sort_order")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Category struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Icon      string `json:"icon"`
		SortOrder int    `json:"sort_order"`
	}
	var categories []Category
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder)
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []Category{}
	}
	json.NewEncoder(w).Encode(categories)
}

func HandleGetGameCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT c.id, c.name, c.icon, c.sort_order FROM categories c
		 JOIN game_categories gc ON c.id = gc.category_id
		 WHERE gc.game_id = ? ORDER BY c.sort_order`, gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Category struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Icon      string `json:"icon"`
		SortOrder int    `json:"sort_order"`
	}
	var categories []Category
	for rows.Next() {
		var c Category
		rows.Scan(&c.ID, &c.Name, &c.Icon, &c.SortOrder)
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []Category{}
	}
	json.NewEncoder(w).Encode(categories)
}

func HandleSetGameCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		CategoryIDs []int `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	db.DB.Conn.Exec("DELETE FROM game_categories WHERE game_id = ?", gameID)
	for _, catID := range req.CategoryIDs {
		db.DB.Conn.Exec("INSERT OR IGNORE INTO game_categories (game_id, category_id) VALUES (?, ?)", gameID, catID)
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Categories updated"})
}

func HandleGetScreenshots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		"SELECT id, game_id, url, sort_order, created_at FROM screenshots WHERE game_id = ? ORDER BY sort_order",
		gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Screenshot struct {
		ID        int    `json:"id"`
		GameID    int    `json:"game_id"`
		URL       string `json:"url"`
		SortOrder int    `json:"sort_order"`
		CreatedAt string `json:"created_at"`
	}
	var screenshots []Screenshot
	for rows.Next() {
		var s Screenshot
		rows.Scan(&s.ID, &s.GameID, &s.URL, &s.SortOrder, &s.CreatedAt)
		screenshots = append(screenshots, s)
	}
	if screenshots == nil {
		screenshots = []Screenshot{}
	}
	json.NewEncoder(w).Encode(screenshots)
}

func HandleAddScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "URL required"})
		return
	}
	var maxOrder int
	db.DB.Conn.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM screenshots WHERE game_id = ?", gameID).Scan(&maxOrder)
	result, err := db.DB.Conn.Exec(
		"INSERT INTO screenshots (game_id, url, sort_order) VALUES (?, ?, ?)",
		gameID, req.URL, maxOrder+1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"game_id": gameID,
		"url":   req.URL,
	})
}

func HandleDeleteScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid screenshot ID"})
		return
	}
	result, err := db.DB.Conn.Exec("DELETE FROM screenshots WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Screenshot not found"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Screenshot deleted"})
}

func HandleGetNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.DB.Conn.Query(
		"SELECT id, title, content, image_url, game_id, is_pinned, created_at FROM news ORDER BY is_pinned DESC, created_at DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type NewsItem struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		ImageURL  string `json:"image_url"`
		GameID    int    `json:"game_id"`
		IsPinned  bool   `json:"is_pinned"`
		CreatedAt string `json:"created_at"`
	}
	var items []NewsItem
	for rows.Next() {
		var n NewsItem
		rows.Scan(&n.ID, &n.Title, &n.Content, &n.ImageURL, &n.GameID, &n.IsPinned, &n.CreatedAt)
		items = append(items, n)
	}
	if items == nil {
		items = []NewsItem{}
	}
	json.NewEncoder(w).Encode(items)
}

func HandleAddNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	var req struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		ImageURL string `json:"image_url"`
		GameID   int    `json:"game_id"`
		IsPinned bool   `json:"is_pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Title == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Title and content required"})
		return
	}
	pinned := 0
	if req.IsPinned {
		pinned = 1
	}
	result, err := db.DB.Conn.Exec(
		"INSERT INTO news (title, content, image_url, game_id, is_pinned) VALUES (?, ?, ?, ?, ?)",
		req.Title, req.Content, req.ImageURL, req.GameID, pinned)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         id,
		"title":      req.Title,
		"content":    req.Content,
		"image_url":  req.ImageURL,
		"game_id":    req.GameID,
		"is_pinned":  req.IsPinned,
	})
}

func HandleDeleteNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid news ID"})
		return
	}
	result, err := db.DB.Conn.Exec("DELETE FROM news WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "News not found"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "News deleted"})
}

func HandleGetFeatured(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.DB.Conn.Query(
		`SELECT g.id, g.name, g.description, g.version, g.category, g.tags,
		 g.cover_url, g.background_url, g.logo_url, g.wide_cover_url,
		 g.file_size, g.download_count, g.created_at, g.updated_at
		 FROM featured_games fg
		 JOIN games g ON fg.game_id = g.id
		 ORDER BY fg.sort_order`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Game struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		Version       string `json:"version"`
		Category      string `json:"category"`
		Tags          string `json:"tags"`
		CoverURL      string `json:"cover_url"`
		BackgroundURL string `json:"background_url"`
		LogoURL       string `json:"logo_url"`
		WideCoverURL  string `json:"wide_cover_url"`
		FileSize      int64  `json:"file_size"`
		DownloadCount int    `json:"download_count"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}
	var games []Game
	for rows.Next() {
		var g Game
		rows.Scan(&g.ID, &g.Name, &g.Description, &g.Version, &g.Category, &g.Tags,
			&g.CoverURL, &g.BackgroundURL, &g.LogoURL, &g.WideCoverURL,
			&g.FileSize, &g.DownloadCount, &g.CreatedAt, &g.UpdatedAt)
		games = append(games, g)
	}
	if games == nil {
		games = []Game{}
	}
	json.NewEncoder(w).Encode(games)
}

func HandleSetFeatured(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		Featured bool `json:"featured"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Featured {
		var maxOrder int
		db.DB.Conn.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM featured_games").Scan(&maxOrder)
		db.DB.Conn.Exec("INSERT OR IGNORE INTO featured_games (game_id, sort_order) VALUES (?, ?)", gameID, maxOrder+1)
	} else {
		db.DB.Conn.Exec("DELETE FROM featured_games WHERE game_id = ?", gameID)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"game_id":  gameID,
		"featured": req.Featured,
	})
}

func HandleTrackPlaytime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	var req struct {
		GameID  int `json:"game_id"`
		Seconds int `json:"seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.GameID == 0 || req.Seconds <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "game_id and positive seconds required"})
		return
	}
	_, err := db.DB.Conn.Exec(
		`INSERT INTO playtime (user_id, game_id, seconds, last_played)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id, game_id) DO UPDATE SET seconds = seconds + ?, last_played = CURRENT_TIMESTAMP`,
		userID, req.GameID, req.Seconds, req.Seconds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Playtime tracked"})
}

func HandleGetPlaytime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT p.game_id, p.seconds, p.last_played, g.name
		 FROM playtime p JOIN games g ON p.game_id = g.id
		 WHERE p.user_id = ? ORDER BY p.last_played DESC`, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type PlaytimeEntry struct {
		GameID     int    `json:"game_id"`
		GameName   string `json:"game_name"`
		Seconds    int    `json:"seconds"`
		LastPlayed string `json:"last_played"`
	}
	var entries []PlaytimeEntry
	for rows.Next() {
		var p PlaytimeEntry
		rows.Scan(&p.GameID, &p.Seconds, &p.LastPlayed, &p.GameName)
		entries = append(entries, p)
	}
	if entries == nil {
		entries = []PlaytimeEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

func HandleUpdateRecentlyPlayed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	var req struct {
		GameID int `json:"game_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.GameID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "game_id required"})
		return
	}
	_, err := db.DB.Conn.Exec(
		`INSERT INTO recently_played (user_id, game_id, played_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id, game_id) DO UPDATE SET played_at = CURRENT_TIMESTAMP`,
		userID, req.GameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Recently played updated"})
}

func HandleGetRecentlyPlayed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT rp.game_id, rp.played_at, g.name, g.cover_url
		 FROM recently_played rp JOIN games g ON rp.game_id = g.id
		 WHERE rp.user_id = ? ORDER BY rp.played_at DESC LIMIT 10`, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type RecentGame struct {
		GameID    int    `json:"game_id"`
		PlayedAt  string `json:"played_at"`
		Name      string `json:"name"`
		CoverURL  string `json:"cover_url"`
	}
	var games []RecentGame
	for rows.Next() {
		var g RecentGame
		rows.Scan(&g.GameID, &g.PlayedAt, &g.Name, &g.CoverURL)
		games = append(games, g)
	}
	if games == nil {
		games = []RecentGame{}
	}
	json.NewEncoder(w).Encode(games)
}

func HandleGetWishlist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT w.game_id, w.created_at, g.name, g.cover_url, g.description, g.version
		 FROM wishlist w JOIN games g ON w.game_id = g.id
		 WHERE w.user_id = ? ORDER BY w.created_at DESC`, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type WishlistItem struct {
		GameID      int    `json:"game_id"`
		CreatedAt   string `json:"created_at"`
		Name        string `json:"name"`
		CoverURL    string `json:"cover_url"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}
	var items []WishlistItem
	for rows.Next() {
		var w WishlistItem
		rows.Scan(&w.GameID, &w.CreatedAt, &w.Name, &w.CoverURL, &w.Description, &w.Version)
		items = append(items, w)
	}
	if items == nil {
		items = []WishlistItem{}
	}
	json.NewEncoder(w).Encode(items)
}

func HandleToggleWishlist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["game_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var exists int
	db.DB.Conn.QueryRow("SELECT COUNT(*) FROM wishlist WHERE user_id = ? AND game_id = ?", userID, gameID).Scan(&exists)
	if exists > 0 {
		db.DB.Conn.Exec("DELETE FROM wishlist WHERE user_id = ? AND game_id = ?", userID, gameID)
		json.NewEncoder(w).Encode(map[string]interface{}{"wishlisted": false})
	} else {
		db.DB.Conn.Exec("INSERT INTO wishlist (user_id, game_id) VALUES (?, ?)", userID, gameID)
		json.NewEncoder(w).Encode(map[string]interface{}{"wishlisted": true})
	}
}

func HandleGetAchievements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		"SELECT id, game_id, name, description, icon_url, secret, created_at FROM achievements WHERE game_id = ?",
		gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Achievement struct {
		ID          int    `json:"id"`
		GameID      int    `json:"game_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
		Secret      bool   `json:"secret"`
		CreatedAt   string `json:"created_at"`
	}
	var achievements []Achievement
	for rows.Next() {
		var a Achievement
		rows.Scan(&a.ID, &a.GameID, &a.Name, &a.Description, &a.IconURL, &a.Secret, &a.CreatedAt)
		achievements = append(achievements, a)
	}
	if achievements == nil {
		achievements = []Achievement{}
	}
	json.NewEncoder(w).Encode(achievements)
}

func HandleAddAchievement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
		Secret      bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name required"})
		return
	}
	secret := 0
	if req.Secret {
		secret = 1
	}
	result, err := db.DB.Conn.Exec(
		"INSERT INTO achievements (game_id, name, description, icon_url, secret) VALUES (?, ?, ?, ?, ?)",
		gameID, req.Name, req.Description, req.IconURL, secret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"game_id": gameID,
		"name":    req.Name,
	})
}

func HandleUnlockAchievement(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	achievementID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid achievement ID"})
		return
	}
	var gameID int
	err = db.DB.Conn.QueryRow("SELECT game_id FROM achievements WHERE id = ?", achievementID).Scan(&gameID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Achievement not found"})
		return
	}
	var alreadyUnlocked int
	db.DB.Conn.QueryRow("SELECT COUNT(*) FROM user_achievements WHERE user_id = ? AND achievement_id = ?", userID, achievementID).Scan(&alreadyUnlocked)
	if alreadyUnlocked > 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"unlocked": true, "already_unlocked": true})
		return
	}
	_, err = db.DB.Conn.Exec(
		"INSERT INTO user_achievements (user_id, achievement_id, game_id) VALUES (?, ?, ?)",
		userID, achievementID, gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"unlocked": true, "already_unlocked": false})
}

func HandleGetUserAchievements(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT ua.id, ua.achievement_id, ua.game_id, a.name, a.description, a.icon_url, g.name as game_name, ua.unlocked_at
		 FROM user_achievements ua
		 JOIN achievements a ON ua.achievement_id = a.id
		 JOIN games g ON ua.game_id = g.id
		 WHERE ua.user_id = ? ORDER BY ua.unlocked_at DESC`, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type UserAchievement struct {
		ID            int    `json:"id"`
		AchievementID int    `json:"achievement_id"`
		GameID        int    `json:"game_id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		IconURL       string `json:"icon_url"`
		GameName      string `json:"game_name"`
		UnlockedAt    string `json:"unlocked_at"`
	}
	var achievements []UserAchievement
	for rows.Next() {
		var a UserAchievement
		rows.Scan(&a.ID, &a.AchievementID, &a.GameID, &a.Name, &a.Description, &a.IconURL, &a.GameName, &a.UnlockedAt)
		achievements = append(achievements, a)
	}
	if achievements == nil {
		achievements = []UserAchievement{}
	}
	json.NewEncoder(w).Encode(achievements)
}

func HandleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["game_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT l.id, l.user_id, l.score, l.updated_at, u.username, u.avatar_url
		 FROM leaderboard l
		 JOIN users u ON l.user_id = u.id
		 WHERE l.game_id = ?
		 ORDER BY l.score DESC LIMIT 50`, gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type LeaderboardEntry struct {
		Rank      int    `json:"rank"`
		UserID    int    `json:"user_id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		Score     int    `json:"score"`
		UpdatedAt string `json:"updated_at"`
	}
	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		rows.Scan(&e.Rank, &e.UserID, &e.Score, &e.UpdatedAt, &e.Username, &e.AvatarURL)
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []LeaderboardEntry{}
	}
	json.NewEncoder(w).Encode(entries)
}

func HandleUpdateScore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["game_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		Score int `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	_, err = db.DB.Conn.Exec(
		`INSERT INTO leaderboard (user_id, game_id, score, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id, game_id) DO UPDATE SET score = MAX(score, ?), updated_at = CURRENT_TIMESTAMP`,
		userID, gameID, req.Score, req.Score)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Score updated"})
}

func HandleGetChatMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "global"
	}
	beforeID := 0
	if beforeStr := r.URL.Query().Get("before_id"); beforeStr != "" {
		beforeID, _ = strconv.Atoi(beforeStr)
	}
	rows, err := db.DB.Conn.Query(
		`SELECT cm.id, cm.sender_id, cm.receiver_id, cm.channel, cm.message, cm.created_at, u.username, u.avatar_url
		 FROM chat_messages cm
		 JOIN users u ON cm.sender_id = u.id
		 WHERE cm.channel = ? AND (? = 0 OR cm.id < ?)
		 ORDER BY cm.id DESC LIMIT 50`, channel, beforeID, beforeID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type ChatMessage struct {
		ID         int    `json:"id"`
		SenderID   int    `json:"sender_id"`
		ReceiverID int    `json:"receiver_id"`
		Channel    string `json:"channel"`
		Message    string `json:"message"`
		CreatedAt  string `json:"created_at"`
		Username   string `json:"username"`
		AvatarURL  string `json:"avatar_url"`
	}
	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Channel, &m.Message, &m.CreatedAt, &m.Username, &m.AvatarURL)
		messages = append(messages, m)
	}
	if messages == nil {
		messages = []ChatMessage{}
	}
	json.NewEncoder(w).Encode(messages)
}

func HandleSendChatMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	var req struct {
		Message    string `json:"message"`
		Channel    string `json:"channel"`
		ReceiverID int    `json:"receiver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Message required"})
		return
	}
	if req.Channel == "" {
		req.Channel = "global"
	}
	result, err := db.DB.Conn.Exec(
		"INSERT INTO chat_messages (sender_id, receiver_id, channel, message) VALUES (?, ?, ?, ?)",
		userID, req.ReceiverID, req.Channel, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"message": req.Message,
	})
}

func HandleGetFriendsOnline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT f.id, f.user_id, f.friend_id, f.status,
		 CASE WHEN f.user_id=? THEN u2.id ELSE u1.id END as friend_uid,
		 CASE WHEN f.user_id=? THEN u2.username ELSE u1.username END as friend_name,
		 CASE WHEN f.user_id=? THEN u2.avatar_url ELSE u1.avatar_url END as friend_avatar
		 FROM friendships f
		 JOIN users u1 ON f.user_id = u1.id
		 JOIN users u2 ON f.friend_id = u2.id
		 WHERE (f.user_id=? OR f.friend_id=?) AND f.status='accepted'
		 ORDER BY friend_name`, userID, userID, userID, userID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type FriendOnline struct {
		ID        int    `json:"id"`
		UserID    int    `json:"user_id"`
		FriendID  int    `json:"friend_id"`
		Status    string `json:"status"`
		FriendUID int    `json:"friend_uid"`
		Name      string `json:"name"`
		Avatar    string `json:"avatar"`
		Online    bool   `json:"online"`
	}
	var friends []FriendOnline
	for rows.Next() {
		var f FriendOnline
		rows.Scan(&f.ID, &f.UserID, &f.FriendID, &f.Status, &f.FriendUID, &f.Name, &f.Avatar)
		f.Online = false
		friends = append(friends, f)
	}
	if friends == nil {
		friends = []FriendOnline{}
	}
	json.NewEncoder(w).Encode(friends)
}

func HandleGetMods(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT m.id, m.game_id, m.user_id, m.name, m.description, m.version,
		 m.file_size, m.downloads, m.created_at, u.username
		 FROM mods m JOIN users u ON m.user_id = u.id
		 WHERE m.game_id = ? ORDER BY m.downloads DESC`, gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Mod struct {
		ID          int    `json:"id"`
		GameID      int    `json:"game_id"`
		UserID      int    `json:"user_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		FileSize    int64  `json:"file_size"`
		Downloads   int    `json:"downloads"`
		CreatedAt   string `json:"created_at"`
		Author      string `json:"author"`
	}
	var mods []Mod
	for rows.Next() {
		var m Mod
		rows.Scan(&m.ID, &m.GameID, &m.UserID, &m.Name, &m.Description, &m.Version,
			&m.FileSize, &m.Downloads, &m.CreatedAt, &m.Author)
		mods = append(mods, m)
	}
	if mods == nil {
		mods = []Mod{}
	}
	json.NewEncoder(w).Encode(mods)
}

func HandleAddMod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name required"})
		return
	}
	if req.Version == "" {
		req.Version = "1.0.0"
	}
	result, err := db.DB.Conn.Exec(
		"INSERT INTO mods (game_id, user_id, name, description, version) VALUES (?, ?, ?, ?, ?)",
		gameID, userID, req.Name, req.Description, req.Version)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"game_id": gameID,
		"name":    req.Name,
		"version": req.Version,
	})
}

func HandleDeleteMod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid mod ID"})
		return
	}
	db.DB.Conn.Exec("DELETE FROM mods WHERE id = ?", id)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleGetServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT id, game_id, name, ip, port, max_players, current_players, map_name, is_official, created_at
		 FROM multiplayer_servers WHERE game_id = ? ORDER BY current_players DESC`, gameID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type Server struct {
		ID             int    `json:"id"`
		GameID         int    `json:"game_id"`
		Name           string `json:"name"`
		IP             string `json:"ip"`
		Port           int    `json:"port"`
		MaxPlayers     int    `json:"max_players"`
		CurrentPlayers int    `json:"current_players"`
		MapName        string `json:"map_name"`
		IsOfficial     bool   `json:"is_official"`
		CreatedAt      string `json:"created_at"`
	}
	var servers []Server
	for rows.Next() {
		var s Server
		rows.Scan(&s.ID, &s.GameID, &s.Name, &s.IP, &s.Port, &s.MaxPlayers,
			&s.CurrentPlayers, &s.MapName, &s.IsOfficial, &s.CreatedAt)
		servers = append(servers, s)
	}
	if servers == nil {
		servers = []Server{}
	}
	json.NewEncoder(w).Encode(servers)
}

func HandleAddServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	gameID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	var req struct {
		Name       string `json:"name"`
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		MaxPlayers int    `json:"max_players"`
		MapName    string `json:"map_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	if req.Name == "" || req.IP == "" || req.Port == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name, ip, and port required"})
		return
	}
	if req.MaxPlayers <= 0 {
		req.MaxPlayers = 32
	}
	result, err := db.DB.Conn.Exec(
		`INSERT INTO multiplayer_servers (game_id, name, ip, port, max_players, map_name)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		gameID, req.Name, req.IP, req.Port, req.MaxPlayers, req.MapName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"game_id": gameID,
		"name":    req.Name,
	})
}

func HandleUpdateServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	serverID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid server ID"})
		return
	}
	var req struct {
		CurrentPlayers int    `json:"current_players"`
		MapName        string `json:"map_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	_, err = db.DB.Conn.Exec(
		"UPDATE multiplayer_servers SET current_players = ?, map_name = ? WHERE id = ?",
		req.CurrentPlayers, req.MapName, serverID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Server updated"})
}

func HandleDeleteServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	serverID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid server ID"})
		return
	}
	result, err := db.DB.Conn.Exec("DELETE FROM multiplayer_servers WHERE id = ?", serverID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server not found"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Server deleted"})
}

func HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
		return
	}
	user, err := db.DB.GetUserByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}
	var gameCount int
	db.DB.Conn.QueryRow("SELECT COUNT(DISTINCT game_id) FROM playtime WHERE user_id = ?", id).Scan(&gameCount)
	var reviewCount int
	db.DB.Conn.QueryRow("SELECT COUNT(*) FROM reviews WHERE username = ?", user.Username).Scan(&reviewCount)
	var achievementCount int
	db.DB.Conn.QueryRow("SELECT COUNT(*) FROM user_achievements WHERE user_id = ?", id).Scan(&achievementCount)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"display_name":    user.DisplayName,
		"avatar_url":      user.AvatarURL,
		"game_count":      gameCount,
		"review_count":    reviewCount,
		"achievement_count": achievementCount,
	})
}

func HandleGetUserGames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user ID"})
		return
	}
	rows, err := db.DB.Conn.Query(
		`SELECT p.game_id, g.name, g.cover_url, p.seconds, p.last_played
		 FROM playtime p JOIN games g ON p.game_id = g.id
		 WHERE p.user_id = ? ORDER BY p.last_played DESC`, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	type UserGame struct {
		GameID     int    `json:"game_id"`
		Name       string `json:"name"`
		CoverURL   string `json:"cover_url"`
		Seconds    int    `json:"seconds"`
		LastPlayed string `json:"last_played"`
	}
	var games []UserGame
	for rows.Next() {
		var g UserGame
		rows.Scan(&g.GameID, &g.Name, &g.CoverURL, &g.Seconds, &g.LastPlayed)
		games = append(games, g)
	}
	if games == nil {
		games = []UserGame{}
	}
	json.NewEncoder(w).Encode(games)
}

func HandleAddCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name required"})
		return
	}
	var maxOrder int
	db.DB.Conn.QueryRow("SELECT COALESCE(MAX(sort_order),0) FROM categories").Scan(&maxOrder)
	_, err := db.DB.Conn.Exec("INSERT INTO categories (name, icon, sort_order) VALUES (?, ?, ?)", req.Name, req.Icon, maxOrder+1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleGetLauncherVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var id int
	var version, changelog, filePath string
	var fileSize int
	var createdAt string
	err := db.DB.Conn.QueryRow("SELECT id, version, changelog, file_path, file_size, created_at FROM launcher_updates ORDER BY id DESC LIMIT 1").
		Scan(&id, &version, &changelog, &filePath, &fileSize, &createdAt)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"has_update": false})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_update": true,
		"id":         id,
		"version":    version,
		"changelog":  changelog,
		"file_size":  fileSize,
		"created_at": createdAt,
	})
}

func HandleUploadLauncherUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseMultipartForm(1024 << 20)
	version := r.FormValue("version")
	changelog := r.FormValue("changelog")
	if version == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Version required"})
		return
	}
	fh, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer fh.Close()
	os.MkdirAll("storage/updates", 0755)
	filename := fmt.Sprintf("MKLauncher-Setup-%s.exe", version)
	dstPath := filepath.Join("storage/updates", filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
		return
	}
	defer dst.Close()
	written, err := io.Copy(dst, fh)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write file"})
		return
	}
	db.DB.Conn.Exec("INSERT INTO launcher_updates (version, changelog, file_path, file_size) VALUES (?, ?, ?, ?)",
		version, changelog, dstPath, written)
	log.Printf("[API] Launcher update uploaded: v%s (%d bytes)", version, written)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "version": version, "file_size": written})
}

func HandleDownloadLauncherUpdate(w http.ResponseWriter, r *http.Request) {
	var filePath string
	err := db.DB.Conn.QueryRow("SELECT file_path FROM launcher_updates ORDER BY id DESC LIMIT 1").Scan(&filePath)
	if err != nil || filePath == "" {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
		http.ServeFile(w, r, filePath)
		return
	}
	http.NotFound(w, r)
}

func HandleSubmitReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		GameID      int    `json:"game_id"`
		GameName    string `json:"game_name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	if req.Description == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Description required"})
		return
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Login required"})
		return
	}
	uid, _ := strconv.Atoi(userID)
	var username string
	db.DB.Conn.QueryRow("SELECT username FROM users WHERE id=?", uid).Scan(&username)
	if username == "" {
		username = "user_" + userID
	}
	db.DB.Conn.Exec("INSERT INTO reports (user_id, username, game_id, game_name, description) VALUES (?, ?, ?, ?, ?)",
		uid, username, req.GameID, req.GameName, req.Description)
	log.Printf("[API] Report submitted by %s for game %s", username, req.GameName)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleListReports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.DB.Conn.Query("SELECT id, user_id, username, game_id, game_name, description, status, admin_reply, created_at FROM reports ORDER BY id DESC")
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	defer rows.Close()
	var reports []map[string]interface{}
	for rows.Next() {
		var id, uid, gameID int
		var username, gameName, desc, status, adminReply, createdAt string
		rows.Scan(&id, &uid, &username, &gameID, &gameName, &desc, &status, &adminReply, &createdAt)
		reports = append(reports, map[string]interface{}{
			"id": id, "user_id": uid, "username": username, "game_id": gameID,
			"game_name": gameName, "description": desc, "status": status,
			"admin_reply": adminReply, "created_at": createdAt,
		})
	}
	if reports == nil {
		reports = []map[string]interface{}{}
	}
	json.NewEncoder(w).Encode(reports)
}

func HandleReplyReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		ReportID int    `json:"report_id"`
		Reply    string `json:"reply"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	if req.Status == "" { req.Status = "replied" }
	db.DB.Conn.Exec("UPDATE reports SET admin_reply=?, status=? WHERE id=?", req.Reply, req.Status, req.ReportID)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
