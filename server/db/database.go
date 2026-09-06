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

var DB *sql.DB

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
	DB = conn

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
