package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := safeUploadPart(mux.Vars(r)["session"])
	if session == "" {
		http.Error(w, "Invalid upload session", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		http.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	name := strings.TrimPrefix(filepath.ToSlash(r.FormValue("path")), "/")
	if name == "" || strings.Contains(name, "../") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	_, file, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	destination := filepath.Join("storage", "uploads", session, filepath.FromSlash(name))
	if existing, statErr := os.Stat(destination); statErr == nil && existing.Size() == file.Size {
		json.NewEncoder(w).Encode(map[string]interface{}{"uploaded": true, "skipped": true, "size": file.Size})
		return
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		http.Error(w, "Cannot create upload directory", http.StatusInternalServerError)
		return
	}
	dst, err := os.Create(destination)
	if err != nil {
		http.Error(w, "Cannot save upload", http.StatusInternalServerError)
		return
	}
	src, err := file.Open()
	if err != nil {
		dst.Close()
		http.Error(w, "Cannot read upload", http.StatusBadRequest)
		return
	}
	written, copyErr := io.Copy(dst, src)
	src.Close()
	dst.Close()
	if copyErr != nil {
		http.Error(w, "Upload interrupted", http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"uploaded": true, "size": written})
}

func HandleFinalizeUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session := safeUploadPart(mux.Vars(r)["session"])
	if session == "" {
		http.Error(w, "Invalid upload session", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
		Description string `json:"description"`
		Version string `json:"version"`
		Category string `json:"category"`
		Tags string `json:"tags"`
		GameFolder string `json:"game_folder"`
		ExePath string `json:"exe_path"`
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Invalid game metadata", http.StatusBadRequest)
		return
	}
	if req.Version == "" { req.Version = "1.0.0" }
	base := filepath.Join("storage", "uploads", session)
	archivePath, size, err := zipUploadFiles(base, req.Name, req.Version, req.Files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	game := &db.Game{Name: req.Name, Description: req.Description, Version: req.Version, Category: req.Category, Tags: req.Tags, GameFolder: req.GameFolder, ExePath: req.ExePath, ArchivePath: archivePath, FileSize: size}
	if err := db.DB.AddGame(game); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	os.RemoveAll(base)
	json.NewEncoder(w).Encode(game)
}

func zipUploadFiles(base, gameName, version string, files []string) (string, int64, error) {
	var headers []*multipart.FileHeader
	for _, name := range files {
		name = filepath.ToSlash(name)
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "../") { return "", 0, fmt.Errorf("invalid uploaded path: %s", name) }
		path := filepath.Join(base, filepath.FromSlash(name))
		info, err := os.Stat(path)
		if err != nil || info.IsDir() { return "", 0, fmt.Errorf("file not uploaded: %s", name) }
		headers = append(headers, &multipart.FileHeader{Filename: name, Size: info.Size()})
	}
	return saveFolderArchiveFromPaths(base, gameName, version, headers)
}

func saveFolderArchiveFromPaths(base, gameName, version string, files []*multipart.FileHeader) (string, int64, error) {
	archivePath := filepath.Join("storage", "archives", fmt.Sprintf("%s_%s.zip", sanitizeName(gameName), sanitizeName(version)))
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil { return "", 0, err }
	dst, err := os.Create(archivePath); if err != nil { return "", 0, err }
	archive := zip.NewWriter(dst)
	for _, fh := range files {
		src, err := os.Open(filepath.Join(base, filepath.FromSlash(fh.Filename))); if err != nil { archive.Close(); dst.Close(); return "", 0, err }
		entry, err := archive.Create(filepath.ToSlash(fh.Filename)); if err != nil { src.Close(); archive.Close(); dst.Close(); return "", 0, err }
		if _, err = io.Copy(entry, src); err != nil { src.Close(); archive.Close(); dst.Close(); return "", 0, err }; src.Close()
	}
	if err := archive.Close(); err != nil { dst.Close(); return "", 0, err }; if err := dst.Close(); err != nil { return "", 0, err }
	info, err := os.Stat(archivePath); if err != nil { return "", 0, err }; return archivePath, info.Size(), nil
}

func safeUploadPart(value string) string { value = filepath.Base(value); if value == "." || value == ".." || value == "" { return "" }; return value }
