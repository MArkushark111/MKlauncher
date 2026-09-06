package api

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"time"

	"mkgames-server/db"

	"github.com/gorilla/mux"
)

func HandleAddAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Username string `json:"username"`
		Passcode string `json:"passcode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Username == "" || req.Passcode == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username and passcode required"})
		return
	}

	if err := db.DB.AddAdmin(req.Username, req.Passcode); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Admin added"})
}

func HandleRemoveAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := mux.Vars(r)["username"]
	if err := db.DB.RemoveAdmin(username); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Admin removed"})
}

func HandleListAdmins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	admins, err := db.DB.ListAdmins()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if admins == nil {
		admins = []db.Admin{}
	}
	json.NewEncoder(w).Encode(admins)
}

func HandleGetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := db.DB.GetDownloadStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(stats)
}

func HandleGetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := db.DB.GetHealthStatus()
	json.NewEncoder(w).Encode(health)
}

func HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	config, err := db.DB.GetConfig()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(config)
}

func HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var config db.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if err := db.DB.UpdateConfig(&config); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(config)
}

func HandleRestartServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if _, err := exec.LookPath("systemctl"); err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server restart requires systemd"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Server restart requested"})
	go func() {
		time.Sleep(250 * time.Millisecond)
		_ = exec.Command("systemctl", "restart", "mkgames").Run()
	}()
}

func HandleGetNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	notifs, err := db.DB.GetNotifications(50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if notifs == nil {
		notifs = []db.Notification{}
	}
	json.NewEncoder(w).Encode(notifs)
}

func HandleGetGameVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := mux.Vars(r)["id"]
	var id int
	if _, err := jsonDecInt(idStr, &id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid game ID"})
		return
	}

	versions, err := db.DB.GetGameVersions(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if versions == nil {
		versions = []db.GameVersion{}
	}
	json.NewEncoder(w).Encode(versions)
}

func jsonDecInt(s string, v *int) (int, error) {
	*v = 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		*v = *v*10 + int(c-'0')
	}
	return *v, nil
}
