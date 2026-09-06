package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mkgames-server/api"
	"mkgames-server/db"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const (
	Version    = "1.0.1"
	ConfigDir  = "/etc/mkgames"
	DataDir    = "/var/lib/mkgames"
	LogFile    = "/var/log/mkgames/server.log"
	StorageDir = "storage"
	AdminWebDir = "../admin/web"
)

var (
	logFile *os.File
)

func main() {
	loadLocalEnv()
	if len(os.Args) > 1 {
		handleCLI(os.Args[1:])
		return
	}

	cmd := flag.String("cmd", "", "CLI command")
	flag.Parse()
	if *cmd != "" {
		handleCLI([]string{*cmd})
		return
	}

	setupLogging()
	log.Printf("[SERVER] MKGames Server v%s starting...", Version)

	if err := os.MkdirAll(DataDir, 0755); err != nil {
		log.Fatalf("[FATAL] Cannot create data dir: %v", err)
	}

	database, err := db.Initialize(DataDir)
	if err != nil {
		log.Fatalf("[FATAL] Database init failed: %v", err)
	}
	defer database.Conn.Close()

	var existingAdmins []db.Admin
	existingAdmins, _ = db.DB.ListAdmins()
	if len(existingAdmins) == 0 {
		passcode := strings.TrimSpace(os.Getenv("MKGAMES_ADMIN_PASSCODE"))
		if passcode == "" {
			log.Fatal("[FATAL] No admins found. Set MKGAMES_ADMIN_PASSCODE before first start")
		}
		if err := db.DB.AddAdmin("admin", passcode); err != nil {
			log.Fatalf("[FATAL] Failed to create initial admin: %v", err)
		}
		log.Println("[SERVER] Initial admin created")
	}

	router := mux.NewRouter()
	adminWebDir := os.Getenv("MKGAMES_ADMIN_WEB_DIR")
	if adminWebDir == "" {
		adminWebDir = AdminWebDir
	}

	router.HandleFunc("/api/auth", api.HandleAuth).Methods("POST")

	router.HandleFunc("/api/games", api.HandleListGames).Methods("GET")
	router.HandleFunc("/api/games/{id:[0-9]+}", api.HandleGetGame).Methods("GET")
	router.HandleFunc("/api/games", api.RequireAuth(api.HandleAddGame)).Methods("POST")
	router.HandleFunc("/api/uploads/{session}/file", api.RequireAuth(api.HandleUploadFile)).Methods("POST")
	router.HandleFunc("/api/uploads/{session}/finalize", api.RequireAuth(api.HandleFinalizeUpload)).Methods("POST")
	router.HandleFunc("/api/games/{id:[0-9]+}", api.RequireAuth(api.HandleUpdateGame)).Methods("PUT")
	router.HandleFunc("/api/games/{id:[0-9]+}", api.RequireAuth(api.HandleDeleteGame)).Methods("DELETE")

	router.HandleFunc("/api/games/{id:[0-9]+}/download", api.HandleDownloadGame).Methods("GET")
	router.HandleFunc("/api/games/{id:[0-9]+}/versions", api.HandleGetGameVersions).Methods("GET")
	router.HandleFunc("/api/games/{id:[0-9]+}/versions", api.RequireAuth(api.HandleAddGameVersion)).Methods("POST")

	router.HandleFunc("/api/archives", api.RequireAuth(api.HandleListArchives)).Methods("GET")
	router.HandleFunc("/api/archives/upload", api.RequireAuth(api.HandleUploadArchive)).Methods("POST")
	router.HandleFunc("/api/archives/delete", api.RequireAuth(api.HandleDeleteArchive)).Methods("POST")
	router.HandleFunc("/api/archive/browse", api.RequireAuth(api.HandleBrowseArchive)).Methods("POST")

	router.HandleFunc("/api/admin/add", api.RequireAuth(api.HandleAddAdmin)).Methods("POST")
	router.HandleFunc("/api/admin/{username}", api.RequireAuth(api.HandleRemoveAdmin)).Methods("DELETE")
	router.HandleFunc("/api/admins", api.RequireAuth(api.HandleListAdmins)).Methods("GET")

	router.HandleFunc("/api/stats", api.RequireAuth(api.HandleGetStats)).Methods("GET")
	router.HandleFunc("/api/health", api.HandleGetHealth).Methods("GET")
	router.HandleFunc("/api/config", api.RequireAuth(api.HandleGetConfig)).Methods("GET")
	router.HandleFunc("/api/config", api.RequireAuth(api.HandleUpdateConfig)).Methods("PUT")
	router.HandleFunc("/api/server/restart", api.RequireAuth(api.HandleRestartServer)).Methods("POST")
	router.HandleFunc("/api/server/wipe", api.RequireAuth(api.HandleWipeServer)).Methods("POST")
	router.HandleFunc("/api/notifications", api.RequireAuth(api.HandleGetNotifications)).Methods("GET")

	router.PathPrefix("/covers/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coversDir := filepath.Join(StorageDir, "covers")
		fileName := strings.TrimPrefix(r.URL.Path, "/covers/")
		fileName = strings.TrimPrefix(fileName, "/")
		directPath := filepath.Join(coversDir, fileName)
		if info, err := os.Stat(directPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, directPath)
			return
		}
		entries, _ := os.ReadDir(coversDir)
		for _, entry := range entries {
			if entry.IsDir() {
				subPath := filepath.Join(coversDir, entry.Name(), fileName)
				if info, err := os.Stat(subPath); err == nil && !info.IsDir() {
					http.ServeFile(w, r, subPath)
					return
				}
			}
		}
		http.NotFound(w, r)
	})

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/api/") {
			filePath := filepath.Join(adminWebDir, r.URL.Path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, filePath)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(adminWebDir, "index.html"))
	})

	addr := ":8080"
	config, err := db.DB.GetConfig()
	if err == nil && config.WANPort != "" {
		addr = ":" + config.WANPort
	}
	allowedOrigins := os.Getenv("MKGAMES_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins(origins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		handlers.AllowCredentials(),
	)(router)

	log.Printf("[SERVER] Listening on %s", addr)
	log.Printf("[SERVER] Admin panel: http://localhost%s", addr)
	log.Printf("[SERVER] WAN: %s:%s", config.WANHost, config.WANPort)

	go api.CleanupTokens()

	server := &http.Server{
		Addr:         addr,
		Handler:      corsHandler,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("[SERVER] Received signal %v, shutting down...", sig)
		server.Close()
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[FATAL] Server error: %v", err)
	}

	log.Println("[SERVER] Server stopped")
	if logFile != nil {
		logFile.Close()
	}
}

func loadLocalEnv() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	envPaths := []string{filepath.Join(ConfigDir, "server.env"), filepath.Join(filepath.Dir(execPath), "server.env")}
	var data []byte
	err = nil
	for _, envPath := range envPaths {
		data, err = os.ReadFile(envPath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && os.Getenv(strings.TrimSpace(parts[0])) == "" {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func handleCLI(args []string) {
	if len(args) == 0 {
		printUsage()
		return
	}

	command := args[0]
	switch command {
	case "start":
		startServer()
	case "stop":
		stopServer()
	case "restart":
		stopServer()
		time.Sleep(2 * time.Second)
		startServer()
	case "status":
		showStatus()
	case "wan":
		showWAN()
	case "add-admin":
		if len(args) < 3 {
			fmt.Println("Usage: mkgames add-admin <username> <passcode>")
			return
		}
		addAdminCLI(args[1], args[2])
	case "remove-admin":
		if len(args) < 2 {
			fmt.Println("Usage: mkgames remove-admin <username>")
			return
		}
		removeAdminCLI(args[1])
	case "list-games":
		listGamesCLI()
	case "list-admins":
		listAdminsCLI()
	case "install-service":
		installSystemdService()
	case "uninstall-service":
		uninstallSystemdService()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`
╔══════════════════════════════════════════╗
║         MKGAMES SERVER CLI               ║
║         Version: ` + Version + `                    ║
╚══════════════════════════════════════════╝

Usage: mkgames <command>

Commands:
  start              Start the server
  stop               Stop the server
  restart            Restart the server
  status             Show server status
  wan                Display WAN:PORT
  add-admin <u> <p>  Add a new admin
  remove-admin <u>   Remove an admin
  list-games         List all games
  list-admins        List all admins
  install-service    Install systemd service
  uninstall-service  Remove systemd service
  help               Show this help
`)
}

func startServer() {
	fmt.Println("[*] Starting MKGames Server...")

	execPath, _ := os.Executable()
	dir := filepath.Dir(execPath)

	cmd := exec.Command("nohup", execPath, ">"+LogFile, "2>&1", "&")
	cmd.Dir = dir

	if err := cmd.Start(); err != nil {
		fmt.Printf("[!] Failed to start: %v\n", err)
		return
	}

	fmt.Println("[+] Server started")
}

func stopServer() {
	fmt.Println("[*] Stopping MKGames Server...")

	cmd := exec.Command("pkill", "-f", "mkgames-server")
	if err := cmd.Run(); err != nil {
		fmt.Println("[!] Server not running or already stopped")
		return
	}
	fmt.Println("[+] Server stopped")
}

func showStatus() {
	cmd := exec.Command("pgrep", "-f", "mkgames-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Status: STOPPED")
		return
	}

	pid := strings.TrimSpace(string(output))
	fmt.Printf("Status: RUNNING (PID: %s)\n", pid)

	resp, err := http.Get("http://localhost:8080/api/health")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(health)
}

func showWAN() {
	database, err := db.Initialize(DataDir)
	if err != nil {
		fmt.Printf("[!] Database error: %v\n", err)
		return
	}
	defer database.Conn.Close()

	config, err := db.DB.GetConfig()
	if err != nil {
		fmt.Printf("[!] Config error: %v\n", err)
		return
	}

	if config.WANHost == "" {
		fmt.Println("[!] WAN host not configured yet.")
		fmt.Println("[*] Configure via admin panel at http://localhost:8080")
		return
	}

	fmt.Printf("WAN: %s:%s\n", config.WANHost, config.WANPort)
}

func addAdminCLI(username, passcode string) {
	database, err := db.Initialize(DataDir)
	if err != nil {
		fmt.Printf("[!] Database error: %v\n", err)
		return
	}
	defer database.Conn.Close()

	if err := db.DB.AddAdmin(username, passcode); err != nil {
		fmt.Printf("[!] Failed to add admin: %v\n", err)
		return
	}
	fmt.Printf("[+] Admin '%s' added successfully\n", username)
}

func removeAdminCLI(username string) {
	database, err := db.Initialize(DataDir)
	if err != nil {
		fmt.Printf("[!] Database error: %v\n", err)
		return
	}
	defer database.Conn.Close()

	if err := db.DB.RemoveAdmin(username); err != nil {
		fmt.Printf("[!] %v\n", err)
		return
	}
	fmt.Printf("[+] Admin '%s' removed\n", username)
}

func listGamesCLI() {
	database, err := db.Initialize(DataDir)
	if err != nil {
		fmt.Printf("[!] Database error: %v\n", err)
		return
	}
	defer database.Conn.Close()

	games, err := db.DB.ListGames()
	if err != nil {
		fmt.Printf("[!] Error: %v\n", err)
		return
	}

	if len(games) == 0 {
		fmt.Println("No games found")
		return
	}

	fmt.Println("ID | Name | Version | Downloads")
	fmt.Println("---|------|---------|----------")
	for _, g := range games {
		fmt.Printf("%d | %s | %s | %d\n", g.ID, g.Name, g.Version, g.DownloadCount)
	}
}

func listAdminsCLI() {
	database, err := db.Initialize(DataDir)
	if err != nil {
		fmt.Printf("[!] Database error: %v\n", err)
		return
	}
	defer database.Conn.Close()

	admins, err := db.DB.ListAdmins()
	if err != nil {
		fmt.Printf("[!] Error: %v\n", err)
		return
	}

	if len(admins) == 0 {
		fmt.Println("No admins found")
		return
	}

	fmt.Println("Username | Created")
	fmt.Println("---------|--------")
	for _, a := range admins {
		fmt.Printf("%s | %s\n", a.Username, a.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

func installSystemdService() {
	serviceContent := `[Unit]
Description=MKGames Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/mkgames
ExecStart=/opt/mkgames/mkgames-server
Restart=always
RestartSec=5
StandardOutput=append:/var/log/mkgames/server.log
StandardError=append:/var/log/mkgames/server.log

[Install]
WantedBy=multi-user.target
`

	servicePath := "/etc/systemd/system/mkgames.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		fmt.Printf("[!] Failed to write service file: %v\n", err)
		return
	}

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "mkgames").Run()
	fmt.Println("[+] Systemd service installed and enabled")
	fmt.Println("[*] Start with: systemctl start mkgames")
}

func uninstallSystemdService() {
	exec.Command("systemctl", "stop", "mkgames").Run()
	exec.Command("systemctl", "disable", "mkgames").Run()
	os.Remove("/etc/systemd/system/mkgames.service")
	exec.Command("systemctl", "daemon-reload").Run()
	fmt.Println("[+] Systemd service removed")
}

func setupLogging() {
	logDir := filepath.Dir(LogFile)
	os.MkdirAll(logDir, 0755)

	var err error
	logFile, err = os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[WARN] Cannot open log file: %v, using stdout", err)
		return
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
