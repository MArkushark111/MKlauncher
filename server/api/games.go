package api

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func HandleListGames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	games, err := db.DB.ListGames()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if games == nil {
		games = []db.Game{}
	}
	json.NewEncoder(w).Encode(games)
}

func HandleGetGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}

	game, err := db.DB.GetGame(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game not found"})
		return
	}

	json.NewEncoder(w).Encode(game)
}

func HandleAddGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := r.ParseMultipartForm(4096 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid form data"})
		return
	}

	game := &db.Game{
		Name:          r.FormValue("name"),
		Description:   r.FormValue("description"),
		Version:       r.FormValue("version"),
		Category:      r.FormValue("category"),
		Tags:          r.FormValue("tags"),
		GameFolder:    r.FormValue("game_folder"),
		ExePath:       r.FormValue("exe_path"),
		ArchivePath:   r.FormValue("archive_path"),
	}

	if game.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game name required"})
		return
	}
	if game.Version == "" {
		game.Version = "1.0.0"
	}

	config, _ := db.DB.GetConfig()
	storageDir := filepath.Join("storage", "games", sanitizeName(game.Name))
	os.MkdirAll(storageDir, 0755)

	if _, fh, err := r.FormFile("cover"); err == nil {
		game.CoverURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_cover", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("background"); err == nil {
		game.BackgroundURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_bg", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("logo"); err == nil {
		game.LogoURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_logo", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("wide_cover"); err == nil {
		game.WideCoverURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_wide", timeNowUnix()))
	}

	if _, fh, err := r.FormFile("archive"); err == nil {
		archivePath := filepath.Join("storage", "archives", fmt.Sprintf("%s_%s", sanitizeName(game.Name), fh.Filename))
		dst, createErr := os.Create(archivePath)
		if createErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save archive"})
			return
		}
		defer dst.Close()
		src, _ := fh.Open()
		defer src.Close()
		written, _ := io.Copy(dst, src)
		game.ArchivePath = archivePath
		game.FileSize = written
	}
	if game.ArchivePath == "" {
		files := r.MultipartForm.File["game_files"]
		if len(files) > 0 {
			archivePath, written, archiveErr := saveFolderArchive(game.Name, game.Version, files)
			if archiveErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": archiveErr.Error()})
				return
			}
			game.ArchivePath = archivePath
			game.FileSize = written
		}
	}
	if game.ArchivePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Select an archive or a game folder"})
		return
	}

	if config != nil && config.WANHost != "" && config.WANPort != "" {
		if game.CoverURL != "" {
			game.CoverURL = fmt.Sprintf("http://%s:%s/covers/%s", config.WANHost, config.WANPort, filepath.Base(game.CoverURL))
		}
		if game.BackgroundURL != "" {
			game.BackgroundURL = fmt.Sprintf("http://%s:%s/covers/%s", config.WANHost, config.WANPort, filepath.Base(game.BackgroundURL))
		}
		if game.LogoURL != "" {
			game.LogoURL = fmt.Sprintf("http://%s:%s/covers/%s", config.WANHost, config.WANPort, filepath.Base(game.LogoURL))
		}
		if game.WideCoverURL != "" {
			game.WideCoverURL = fmt.Sprintf("http://%s:%s/covers/%s", config.WANHost, config.WANPort, filepath.Base(game.WideCoverURL))
		}
	}

	if err := db.DB.AddGame(game); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[API] Game added: %s (ID: %d)", game.Name, game.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

func HandleUpdateGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}

	game, err := db.DB.GetGame(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game not found"})
		return
	}

	if r.FormValue("name") != "" {
		game.Name = r.FormValue("name")
	}
	if r.FormValue("description") != "" {
		game.Description = r.FormValue("description")
	}
	if r.FormValue("version") != "" {
		game.Version = r.FormValue("version")
	}
	if r.FormValue("category") != "" {
		game.Category = r.FormValue("category")
	}
	if r.FormValue("tags") != "" {
		game.Tags = r.FormValue("tags")
	}
	if r.FormValue("game_folder") != "" {
		game.GameFolder = r.FormValue("game_folder")
	}
	if r.FormValue("exe_path") != "" {
		game.ExePath = r.FormValue("exe_path")
	}

	if _, fh, err := r.FormFile("cover"); err == nil {
		game.CoverURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_cover", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("background"); err == nil {
		game.BackgroundURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_bg", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("logo"); err == nil {
		game.LogoURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_logo", timeNowUnix()))
	}
	if _, fh, err := r.FormFile("wide_cover"); err == nil {
		game.WideCoverURL = saveUploadedFile(fh, "storage", "covers", fmt.Sprintf("%d_wide", timeNowUnix()))
	}

	if err := db.DB.UpdateGame(game); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(game)
}

func HandleAddGameVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}
	game, err := db.DB.GetGame(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game not found"})
		return
	}

	config, _ := db.DB.GetConfig()
	maxMB := 4096
	if config != nil && config.MaxUploadMB > 0 {
		maxMB = config.MaxUploadMB
	}
	if err := r.ParseMultipartForm(int64(maxMB << 20)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid upload"})
		return
	}
	version := strings.TrimSpace(r.FormValue("version"))
	if version == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Version is required"})
		return
	}
	_, fh, err := r.FormFile("archive")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No archive file provided"})
		return
	}
	if fh.Size > int64(maxMB<<20) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("File exceeds max size of %dMB", maxMB)})
		return
	}

	archivePath, written, err := saveGameVersionArchive(id, version, fh)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save archive"})
		return
	}
	if err := db.DB.AddGameVersion(id, version, archivePath, written); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	game.Version = version
	game.ArchivePath = archivePath
	game.FileSize = written
	if err := db.DB.UpdateGame(game); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Game version uploaded",
		"version": version,
		"size":    written,
	})
}

func HandleDeleteGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}

	game, err := db.DB.GetGame(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Game not found"})
		return
	}

	if game.ArchivePath != "" {
		os.Remove(game.ArchivePath)
	}
	gameDir := filepath.Join("storage", "games", sanitizeName(game.Name))
	os.RemoveAll(gameDir)

	if err := db.DB.DeleteGame(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[API] Game deleted: %s (ID: %d)", game.Name, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Game deleted"})
}

func HandleDownloadGame(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid game ID", http.StatusBadRequest)
		return
	}

	game, err := db.DB.GetGame(id)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if game.ArchivePath == "" || !fileExists(game.ArchivePath) {
		http.Error(w, "Archive not available", http.StatusNotFound)
		return
	}

	db.DB.IncrementDownload(id, r.RemoteAddr, r.UserAgent())

	filename := filepath.Base(game.ArchivePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, game.ArchivePath)
}

func HandleBrowseArchive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	archivePath, archivePrefix := splitArchivePath(req.Path)
	if archivePath != "" {
		entries, err := listArchiveEntries(archivePath, archivePrefix)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":    "directory",
			"path":    req.Path,
			"entries": entries,
		})
		return
	}

	fsPath := strings.TrimPrefix(filepath.Clean(req.Path), string(filepath.Separator))
	info, err := os.Stat(fsPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Path not found"})
		return
	}

	if !info.IsDir() {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "file",
			"name": filepath.Base(req.Path),
			"path": req.Path,
			"size": info.Size(),
		})
		return
	}

	entries, err := os.ReadDir(fsPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var items []map[string]interface{}
	for _, entry := range entries {
		info, _ := entry.Info()
		item := map[string]interface{}{
			"name":  entry.Name(),
			"path":  filepath.Join(req.Path, entry.Name()),
			"isDir": entry.IsDir(),
		}
		if info != nil {
			item["size"] = info.Size()
		}
		items = append(items, item)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":    "directory",
		"path":    req.Path,
		"entries": items,
	})
}

func splitArchivePath(path string) (string, string) {
	parts := strings.SplitN(path, "::", 2)
	if len(parts) == 1 {
		lower := strings.ToLower(parts[0])
		if !strings.HasSuffix(lower, ".zip") && !strings.HasSuffix(lower, ".rar") &&
			!strings.HasSuffix(lower, ".7z") && !strings.HasSuffix(lower, ".tar") &&
			!strings.HasSuffix(lower, ".gz") && !strings.HasSuffix(lower, ".tgz") {
			return "", ""
		}
	}
	return strings.TrimPrefix(parts[0], "/"), func() string {
		if len(parts) == 2 {
			return strings.Trim(parts[1], "/")
		}
		return ""
	}()
}

func listArchiveEntries(archivePath, prefix string) ([]map[string]interface{}, error) {
	output, err := exec.Command("7z", "l", "-slt", archivePath).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read archive: %w", err)
	}

	var entries []map[string]interface{}
	var name string
	var size int64
	for _, line := range strings.Split(string(output)+"\n", "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Path = "):
			name = strings.TrimPrefix(line, "Path = ")
			size = 0
		case strings.HasPrefix(line, "Size = "):
			size, _ = strconv.ParseInt(strings.TrimPrefix(line, "Size = "), 10, 64)
		case line == "" && name != "":
			if name == filepath.Base(archivePath) || prefix != "" && !strings.HasPrefix(name, prefix+"/") {
				name = ""
				continue
			}
			relative := strings.TrimPrefix(strings.TrimPrefix(name, prefix), "/")
			parts := strings.SplitN(relative, "/", 2)
			entryName := parts[0]
			entryPath := strings.TrimSuffix(archivePath+"::"+strings.Trim(strings.TrimPrefix(name, prefix), "/"), "/")
			isDir := len(parts) == 2 || strings.HasSuffix(name, "/")
			duplicate := false
			for _, existing := range entries {
				if existing["path"] == entryPath {
					duplicate = true
					break
				}
			}
			if !duplicate {
				entries = append(entries, map[string]interface{}{"name": entryName, "path": entryPath, "isDir": isDir, "size": size})
			}
			name = ""
		}
	}
	return entries, nil
}

func HandleListArchives(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	archivesDir := "storage/archives"
	os.MkdirAll(archivesDir, 0755)

	entries, err := os.ReadDir(archivesDir)
	if err != nil {
		json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}

	var archives []map[string]interface{}
	supportedExts := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".tar.gz", ".tgz"}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lowerName := strings.ToLower(entry.Name())
		supported := false
		for _, ext := range supportedExts {
			if strings.HasSuffix(lowerName, ext) {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}

		info, _ := entry.Info()
		archives = append(archives, map[string]interface{}{
			"name": entry.Name(),
			"path": filepath.Join(archivesDir, entry.Name()),
			"size": info.Size(),
		})
	}

	json.NewEncoder(w).Encode(archives)
}

func HandleUploadArchive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	r.ParseMultipartForm(int64(4096 << 20))

	_, fh, err := r.FormFile("archive")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No archive file provided"})
		return
	}

	config, _ := db.DB.GetConfig()
	maxMB := 4096
	if config != nil && config.MaxUploadMB > 0 {
		maxMB = config.MaxUploadMB
	}

	if fh.Size > int64(maxMB<<20) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("File exceeds max size of %dMB", maxMB)})
		return
	}

	archivesDir := "storage/archives"
	os.MkdirAll(archivesDir, 0755)

	dstPath := filepath.Join(archivesDir, fh.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	src, _ := fh.Open()
	defer src.Close()

	written, _ := io.Copy(dst, src)

	log.Printf("[API] Archive uploaded: %s (%d bytes)", fh.Filename, written)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Archive uploaded successfully",
		"path":    dstPath,
		"size":    written,
	})
}

func saveUploadedFile(fh *multipart.FileHeader, dirs ...string) string {
	dir := filepath.Join(dirs...)
	os.MkdirAll(dir, 0755)

	dstPath := filepath.Join(dir, fh.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return ""
	}
	defer dst.Close()

	src, err := fh.Open()
	if err != nil {
		return ""
	}
	defer src.Close()

	io.Copy(dst, src)
	return dstPath
}

func saveFolderArchive(gameName, version string, files []*multipart.FileHeader) (string, int64, error) {
	archiveDir := filepath.Join("storage", "archives")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", 0, err
	}
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("%s_%s.tar.gz", sanitizeName(gameName), sanitizeName(version)))
	dst, err := os.Create(archivePath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()
	gz := gzip.NewWriter(dst)
	archive := tar.NewWriter(gz)
	for _, fh := range files {
		relativePath := strings.TrimPrefix(filepath.ToSlash(fh.Filename), "/")
		if relativePath == "" || strings.Contains(relativePath, "../") {
			continue
		}
		src, openErr := fh.Open()
		if openErr != nil {
			archive.Close()
			gz.Close()
			return "", 0, openErr
		}
		header := &tar.Header{Name: relativePath, Mode: 0755, Size: fh.Size, ModTime: time.Now()}
		if writeErr := archive.WriteHeader(header); writeErr != nil {
			src.Close()
			archive.Close()
			gz.Close()
			return "", 0, writeErr
		}
		_, copyErr := io.Copy(archive, src)
		src.Close()
		if copyErr != nil {
			archive.Close()
			gz.Close()
			return "", 0, copyErr
		}
	}
	if err := archive.Close(); err != nil {
		gz.Close()
		return "", 0, err
	}
	if err := gz.Close(); err != nil {
		return "", 0, err
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		return "", 0, err
	}
	return archivePath, info.Size(), nil
}

func saveGameVersionArchive(gameID int, version string, fh *multipart.FileHeader) (string, int64, error) {
	archiveDir := "storage/archives"
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", 0, err
	}
	safeVersion := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(version)
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("game_%d_v%s_%s", gameID, safeVersion, filepath.Base(fh.Filename)))
	dst, err := os.Create(archivePath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()
	src, err := fh.Open()
	if err != nil {
		return "", 0, err
	}
	defer src.Close()
	written, err := io.Copy(dst, src)
	return archivePath, written, err
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		" ", "_", "/", "_", "\\", "_",
		":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(name)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
