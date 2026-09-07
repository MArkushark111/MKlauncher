package db

import "time"

type Admin struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasscodeHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Game struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Version       string    `json:"version"`
	Category      string    `json:"category"`
	Tags          string    `json:"tags"`
	CoverURL      string    `json:"cover_url"`
	BackgroundURL string    `json:"background_url"`
	LogoURL       string    `json:"logo_url"`
	WideCoverURL  string    `json:"wide_cover_url"`
	ArchivePath   string    `json:"-"`
	GameFolder    string    `json:"game_folder"`
	ExePath       string    `json:"exe_path"`
	FileSize      int64     `json:"file_size"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GameVersion struct {
	ID          int       `json:"id"`
	GameID      int       `json:"game_id"`
	Version     string    `json:"version"`
	ArchivePath string    `json:"-"`
	FileSize    int64     `json:"file_size"`
	CreatedAt   time.Time `json:"created_at"`
}

type Download struct {
	ID           int       `json:"id"`
	GameID       int       `json:"game_id"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

type ServerConfig struct {
	ID            int    `json:"id"`
	WANHost      string `json:"wan_host"`
	WANPort      string `json:"wan_port"`
	ServerName   string `json:"server_name"`
	MaxUploadMB  int    `json:"max_upload_mb"`
}

type Notification struct {
	ID        int       `json:"id"`
	GameID    int       `json:"game_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type Review struct {
	ID        int       `json:"id"`
	GameID    int       `json:"game_id"`
	Username  string    `json:"username"`
	Stars     int       `json:"stars"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
