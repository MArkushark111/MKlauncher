package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const hydraCacheFile = "/var/lib/mkgames/hydra_sources.json"

var (
	hydraData     []byte
	hydraDataLock sync.RWMutex
)

func init() {
	data, err := os.ReadFile(hydraCacheFile)
	if err == nil {
		hydraData = data
		log.Printf("[HYDRA] Loaded %d bytes from cache", len(data))
	}
}

func HandleGetHydraSources(w http.ResponseWriter, r *http.Request) {
	hydraDataLock.RLock()
	defer hydraDataLock.RUnlock()

	if len(hydraData) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(hydraData)
}

func HandleUpdateHydraSources(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	var sources []json.RawMessage
	if err := json.Unmarshal(body, &sources); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(filepath.Dir(hydraCacheFile), 0755); err != nil {
		log.Printf("[HYDRA] Failed to create cache dir: %v", err)
	}

	if err := os.WriteFile(hydraCacheFile, body, 0644); err != nil {
		log.Printf("[HYDRA] Failed to write cache: %v", err)
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	hydraDataLock.Lock()
	hydraData = body
	hydraDataLock.Unlock()

	log.Printf("[HYDRA] Updated cache with %d sources", len(sources))
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true,"sources":` + string(len(sources)) + `}`))
}
