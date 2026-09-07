package db

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *Database

type Database struct {
	Conn *sql.DB
}

func Initialize(dataDir string) (*Database, error) {
	dbPath := filepath.Join(dataDir, "mkgames.db")

	dirErr := os.MkdirAll(dataDir, 0755)
	if dirErr != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", dirErr)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	db := &Database{Conn: conn}
	DB = db

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("[DB] Initialized at %s", dbPath)
	return db, nil
}

func (d *Database) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			passcode_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			version TEXT DEFAULT '1.0.0',
			category TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			cover_url TEXT DEFAULT '',
			background_url TEXT DEFAULT '',
			logo_url TEXT DEFAULT '',
			wide_cover_url TEXT DEFAULT '',
			archive_path TEXT DEFAULT '',
			game_folder TEXT DEFAULT '',
			exe_path TEXT DEFAULT '',
			file_size INTEGER DEFAULT 0,
			download_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS game_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL,
			version TEXT NOT NULL,
			archive_path TEXT DEFAULT '',
			file_size INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS downloads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL,
			ip TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS server_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			wan_host TEXT DEFAULT '',
			wan_port TEXT DEFAULT '8080',
			server_name TEXT DEFAULT 'MKGames Server',
			max_upload_mb INTEGER DEFAULT 4096
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			read INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_games_name ON games(name)`,
		`CREATE INDEX IF NOT EXISTS idx_downloads_game ON downloads(game_id)`,
		`CREATE INDEX IF NOT EXISTS idx_downloads_date ON downloads(downloaded_at)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			stars INTEGER DEFAULT 5,
			title TEXT DEFAULT '',
			text TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_game ON reviews(game_id)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			is_banned INTEGER DEFAULT 0,
			totp_secret TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS friendships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			friend_id INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (friend_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, friend_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS developer_apps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			app_name TEXT NOT NULL,
			api_key TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := d.Conn.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, q)
		}
	}

	var count int
	d.Conn.QueryRow("SELECT COUNT(*) FROM server_config").Scan(&count)
	if count == 0 {
		d.Conn.Exec("INSERT INTO server_config (id, wan_host, wan_port, server_name) VALUES (1, '', '8080', 'MKGames Server')")
	}

	return nil
}

func HashPasscode(passcode string) string {
	h := sha256.Sum256([]byte("mkgames_salt_" + passcode))
	return fmt.Sprintf("%x", h)
}

func (d *Database) VerifyPasscode(passcode string) bool {
	hash := HashPasscode(passcode)
	var count int
	d.Conn.QueryRow("SELECT COUNT(*) FROM admins WHERE passcode_hash = ?", hash).Scan(&count)
	return count > 0
}

func (d *Database) AddAdmin(username, passcode string) error {
	hash := HashPasscode(passcode)
	_, err := d.Conn.Exec("INSERT INTO admins (username, passcode_hash) VALUES (?, ?)", username, hash)
	return err
}

func (d *Database) RemoveAdmin(username string) error {
	result, err := d.Conn.Exec("DELETE FROM admins WHERE username = ?", username)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("admin '%s' not found", username)
	}
	return nil
}

func (d *Database) ListAdmins() ([]Admin, error) {
	rows, err := d.Conn.Query("SELECT id, username, created_at FROM admins ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []Admin
	for rows.Next() {
		var a Admin
		rows.Scan(&a.ID, &a.Username, &a.CreatedAt)
		admins = append(admins, a)
	}
	return admins, nil
}

func (d *Database) AddGame(g *Game) error {
	result, err := d.Conn.Exec(
		`INSERT INTO games (name, description, version, category, tags, 
		 cover_url, background_url, logo_url, wide_cover_url,
		 archive_path, game_folder, exe_path, file_size)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.Name, g.Description, g.Version, g.Category, g.Tags,
		g.CoverURL, g.BackgroundURL, g.LogoURL, g.WideCoverURL,
		g.ArchivePath, g.GameFolder, g.ExePath, g.FileSize,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	g.ID = int(id)

	d.Conn.Exec(
		"INSERT INTO game_versions (game_id, version, archive_path, file_size) VALUES (?, ?, ?, ?)",
		g.ID, g.Version, g.ArchivePath, g.FileSize,
	)

	d.AddNotification(g.ID, "New Game Added", fmt.Sprintf("%s v%s is now available", g.Name, g.Version))

	return nil
}

func (d *Database) UpdateGame(g *Game) error {
	_, err := d.Conn.Exec(
		`UPDATE games SET name=?, description=?, version=?, category=?, tags=?,
		 cover_url=?, background_url=?, logo_url=?, wide_cover_url=?,
		 archive_path=?, game_folder=?, exe_path=?, file_size=?, updated_at=?
		 WHERE id=?`,
		g.Name, g.Description, g.Version, g.Category, g.Tags,
		g.CoverURL, g.BackgroundURL, g.LogoURL, g.WideCoverURL,
		g.ArchivePath, g.GameFolder, g.ExePath, g.FileSize, time.Now(), g.ID,
	)
	return err
}

func (d *Database) DeleteGame(id int) error {
	_, err := d.Conn.Exec("DELETE FROM games WHERE id = ?", id)
	return err
}

func (d *Database) GetGame(id int) (*Game, error) {
	var g Game
	err := d.Conn.QueryRow(
		`SELECT id, name, description, version, category, tags,
		 cover_url, background_url, logo_url, wide_cover_url,
		 archive_path, game_folder, exe_path, file_size, download_count,
		 created_at, updated_at FROM games WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.Version, &g.Category, &g.Tags,
		&g.CoverURL, &g.BackgroundURL, &g.LogoURL, &g.WideCoverURL,
		&g.ArchivePath, &g.GameFolder, &g.ExePath, &g.FileSize, &g.DownloadCount,
		&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (d *Database) ListGames() ([]Game, error) {
	rows, err := d.Conn.Query(
		`SELECT id, name, description, version, category, tags,
		 cover_url, background_url, logo_url, wide_cover_url,
		 archive_path, game_folder, exe_path, file_size, download_count,
		 created_at, updated_at FROM games ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		rows.Scan(&g.ID, &g.Name, &g.Description, &g.Version, &g.Category, &g.Tags,
			&g.CoverURL, &g.BackgroundURL, &g.LogoURL, &g.WideCoverURL,
			&g.ArchivePath, &g.GameFolder, &g.ExePath, &g.FileSize, &g.DownloadCount,
			&g.CreatedAt, &g.UpdatedAt)
		games = append(games, g)
	}
	return games, nil
}

func (d *Database) IncrementDownload(gameID int, ip, ua string) error {
	_, err := d.Conn.Exec("UPDATE games SET download_count = download_count + 1 WHERE id = ?", gameID)
	if err != nil {
		return err
	}
	_, err = d.Conn.Exec("INSERT INTO downloads (game_id, ip, user_agent) VALUES (?, ?, ?)", gameID, ip, ua)
	return err
}

func (d *Database) GetDownloadStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalDownloads int
	d.Conn.QueryRow("SELECT COALESCE(SUM(download_count), 0) FROM games").Scan(&totalDownloads)
	stats["total_downloads"] = totalDownloads

	var totalGames int
	d.Conn.QueryRow("SELECT COUNT(*) FROM games").Scan(&totalGames)
	stats["total_games"] = totalGames

	rows, err := d.Conn.Query(
		`SELECT g.name, g.download_count FROM games g 
		 ORDER BY g.download_count DESC LIMIT 10`)
	if err == nil {
		defer rows.Close()
		var top []map[string]interface{}
		for rows.Next() {
			var name string
			var count int
			rows.Scan(&name, &count)
			top = append(top, map[string]interface{}{"name": name, "downloads": count})
		}
		stats["top_games"] = top
	}

	var todayDownloads int
	d.Conn.QueryRow(
		`SELECT COUNT(*) FROM downloads WHERE downloaded_at >= date('now')`).Scan(&todayDownloads)
	stats["today_downloads"] = todayDownloads

	var weekDownloads int
	d.Conn.QueryRow(
		`SELECT COUNT(*) FROM downloads WHERE downloaded_at >= date('now', '-7 days')`).Scan(&weekDownloads)
	stats["week_downloads"] = weekDownloads

	return stats, nil
}

func (d *Database) GetHealthStatus() map[string]interface{} {
	health := make(map[string]interface{})
	health["status"] = "online"
	health["uptime"] = time.Since(startTime).String()

	var gameCount int
	d.Conn.QueryRow("SELECT COUNT(*) FROM games").Scan(&gameCount)
	health["total_games"] = gameCount

	var downloadCount int
	d.Conn.QueryRow("SELECT COUNT(*) FROM downloads").Scan(&downloadCount)
	health["total_downloads"] = downloadCount

	var adminCount int
	d.Conn.QueryRow("SELECT COUNT(*) FROM admins").Scan(&adminCount)
	health["total_admins"] = adminCount

	return health
}

var startTime = time.Now()

func (d *Database) GetConfig() (*ServerConfig, error) {
	var c ServerConfig
	err := d.Conn.QueryRow(
		"SELECT id, wan_host, wan_port, server_name, max_upload_mb FROM server_config WHERE id = 1",
	).Scan(&c.ID, &c.WANHost, &c.WANPort, &c.ServerName, &c.MaxUploadMB)
	if err != nil {
		return &ServerConfig{WANPort: "8080", ServerName: "MKGames Server", MaxUploadMB: 4096}, nil
	}
	return &c, nil
}

func (d *Database) UpdateConfig(c *ServerConfig) error {
	_, err := d.Conn.Exec(
		"UPDATE server_config SET wan_host=?, wan_port=?, server_name=?, max_upload_mb=? WHERE id=1",
		c.WANHost, c.WANPort, c.ServerName, c.MaxUploadMB,
	)
	return err
}

func (d *Database) AddNotification(gameID int, title, message string) error {
	_, err := d.Conn.Exec(
		"INSERT INTO notifications (game_id, title, message) VALUES (?, ?, ?)",
		gameID, title, message,
	)
	return err
}

func (d *Database) GetNotifications(limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := d.Conn.Query(
		"SELECT id, game_id, title, message, read, created_at FROM notifications ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []Notification
	for rows.Next() {
		var n Notification
		rows.Scan(&n.ID, &n.GameID, &n.Title, &n.Message, &n.Read, &n.CreatedAt)
		notifs = append(notifs, n)
	}
	return notifs, nil
}

func (d *Database) AddGameVersion(gameID int, version, archivePath string, fileSize int64) error {
	_, err := d.Conn.Exec(
		"INSERT INTO game_versions (game_id, version, archive_path, file_size) VALUES (?, ?, ?, ?)",
		gameID, version, archivePath, fileSize,
	)
	return err
}

func (d *Database) GetGameVersions(gameID int) ([]GameVersion, error) {
	rows, err := d.Conn.Query(
		"SELECT id, game_id, version, archive_path, file_size, created_at FROM game_versions WHERE game_id = ? ORDER BY created_at DESC",
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []GameVersion
	for rows.Next() {
		var v GameVersion
		rows.Scan(&v.ID, &v.GameID, &v.Version, &v.ArchivePath, &v.FileSize, &v.CreatedAt)
		versions = append(versions, v)
	}
	return versions, nil
}

func (d *Database) AddReview(gameID int, username string, stars int, title, text string) error {
	_, err := d.Conn.Exec(
		"INSERT INTO reviews (game_id, username, stars, title, text) VALUES (?, ?, ?, ?, ?)",
		gameID, username, stars, title, text,
	)
	return err
}

func (d *Database) GetReviews(gameID int) ([]Review, error) {
	rows, err := d.Conn.Query(
		"SELECT id, game_id, username, stars, title, text, created_at FROM reviews WHERE game_id = ? ORDER BY created_at DESC",
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		rows.Scan(&r.ID, &r.GameID, &r.Username, &r.Stars, &r.Title, &r.Text, &r.CreatedAt)
		reviews = append(reviews, r)
	}
	return reviews, nil
}

func (d *Database) GetReviewStats(gameID int) (avgStars float64, count int) {
	d.Conn.QueryRow("SELECT COALESCE(AVG(stars), 0), COUNT(*) FROM reviews WHERE game_id = ?", gameID).Scan(&avgStars, &count)
	return
}

func (d *Database) RegisterUser(username, passwordHash, displayName string) error {
	_, err := d.Conn.Exec("INSERT INTO users (username, password_hash, display_name) VALUES (?, ?, ?)", username, passwordHash, displayName)
	return err
}

func (d *Database) GetUser(username string) (*User, error) {
	var u User
	err := d.Conn.QueryRow("SELECT id, username, display_name, avatar_url, is_banned, totp_secret, created_at FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.IsBanned, &u.TOTPSecret, &u.CreatedAt)
	if err != nil { return nil, err }
	return &u, nil
}

func (d *Database) GetUserByID(id int) (*User, error) {
	var u User
	err := d.Conn.QueryRow("SELECT id, username, display_name, avatar_url, is_banned, totp_secret, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.IsBanned, &u.TOTPSecret, &u.CreatedAt)
	if err != nil { return nil, err }
	return &u, nil
}

func (d *Database) UpdateUserProfile(userID int, displayName, avatarURL string) error {
	_, err := d.Conn.Exec("UPDATE users SET display_name=?, avatar_url=? WHERE id=?", displayName, avatarURL, userID)
	return err
}

func (d *Database) BanUser(userID int, banned bool) error {
	val := 0
	if banned { val = 1 }
	_, err := d.Conn.Exec("UPDATE users SET is_banned=? WHERE id=?", val, userID)
	return err
}

func (d *Database) SearchUsers(query string) ([]User, error) {
	rows, err := d.Conn.Query("SELECT id, username, display_name, avatar_url, is_banned, created_at FROM users WHERE username LIKE ? OR display_name LIKE ? ORDER BY username LIMIT 20", "%"+query+"%", "%"+query+"%")
	if err != nil { return nil, err }
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.IsBanned, &u.CreatedAt)
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) ListUsers() ([]User, error) {
	rows, err := d.Conn.Query("SELECT id, username, display_name, avatar_url, is_banned, created_at FROM users ORDER BY username")
	if err != nil { return nil, err }
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.IsBanned, &u.CreatedAt)
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) SendFriendRequest(userID, friendID int) error {
	_, err := d.Conn.Exec("INSERT OR IGNORE INTO friendships (user_id, friend_id, status) VALUES (?, ?, 'pending')", userID, friendID)
	return err
}

func (d *Database) AcceptFriendRequest(userID, friendID int) error {
	_, err := d.Conn.Exec("UPDATE friendships SET status='accepted' WHERE user_id=? AND friend_id=? AND status='pending'", friendID, userID)
	if err != nil { return err }
	_, err = d.Conn.Exec("INSERT OR IGNORE INTO friendships (user_id, friend_id, status) VALUES (?, ?, 'accepted')", userID, friendID)
	return err
}

func (d *Database) RemoveFriend(userID, friendID int) error {
	d.Conn.Exec("DELETE FROM friendships WHERE (user_id=? AND friend_id=?) OR (user_id=? AND friend_id=?)", userID, friendID, friendID, userID)
	return nil
}

func (d *Database) GetFriends(userID int) ([]Friendship, error) {
	rows, err := d.Conn.Query(`
		SELECT f.id, f.user_id, f.friend_id, f.status, f.created_at,
			CASE WHEN f.user_id=? THEN u2.username ELSE u1.username END as friend_name,
			CASE WHEN f.user_id=? THEN u2.avatar_url ELSE u1.avatar_url END as friend_avatar
		FROM friendships f
		JOIN users u1 ON f.user_id = u1.id
		JOIN users u2 ON f.friend_id = u2.id
		WHERE (f.user_id=? OR f.friend_id=?) AND f.status='accepted'
		ORDER BY friend_name`, userID, userID, userID, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var friends []Friendship
	for rows.Next() {
		var f Friendship
		rows.Scan(&f.ID, &f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.FriendName, &f.FriendAvatar)
		friends = append(friends, f)
	}
	return friends, nil
}

func (d *Database) GetFriendRequests(userID int) ([]Friendship, error) {
	rows, err := d.Conn.Query(`
		SELECT f.id, f.user_id, f.friend_id, f.status, f.created_at, u.username as friend_name, u.avatar_url as friend_avatar
		FROM friendships f JOIN users u ON f.user_id = u.id
		WHERE f.friend_id=? AND f.status='pending' ORDER BY f.created_at DESC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var reqs []Friendship
	for rows.Next() {
		var f Friendship
		rows.Scan(&f.ID, &f.UserID, &f.FriendID, &f.Status, &f.CreatedAt, &f.FriendName, &f.FriendAvatar)
		reqs = append(reqs, f)
	}
	return reqs, nil
}

func (d *Database) GenerateUserToken(userID int) (string, error) {
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d_%d_%s", userID, time.Now().UnixNano(), "mkgames_token"))))
	_, err := d.Conn.Exec("INSERT INTO user_tokens (user_id, token) VALUES (?, ?)", userID, token)
	return token, err
}

func (d *Database) ValidateUserToken(token string) (*User, error) {
	var userID int
	err := d.Conn.QueryRow("SELECT user_id FROM user_tokens WHERE token=?", token).Scan(&userID)
	if err != nil { return nil, err }
	return d.GetUserByID(userID)
}

func (d *Database) SetTOTPSecret(userID int, secret string) error {
	_, err := d.Conn.Exec("UPDATE users SET totp_secret=? WHERE id=?", secret, userID)
	return err
}
