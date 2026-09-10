package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	hydraCache     = make(map[string]hydraCacheEntry)
	hydraCacheLock sync.Mutex
)

type hydraCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var hydraSourceUrls = []string{
	"https://hydralinks.cloud/sources/fitgirl.json",
	"https://hydralinks.cloud/sources/dodi.json",
	"https://hydralinks.cloud/sources/steamrip.json",
	"https://hydralinks.cloud/sources/gog.json",
	"https://hydralinks.cloud/sources/onlinefix.json",
	"https://hydralinks.cloud/sources/kaoskrew.json",
	"https://hydralinks.cloud/sources/xatab.json",
	"https://hydralinks.cloud/sources/empress.json",
}

func HandleGetHydraSources(w http.ResponseWriter, r *http.Request) {
	sourceURL := r.URL.Query().Get("url")
	if sourceURL == "" {
		handleAllHydraSources(w, r)
		return
	}

	hydraCacheLock.Lock()
	cached, ok := hydraCache[sourceURL]
	hydraCacheLock.Unlock()

	if ok && time.Now().Before(cached.expiresAt) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "hit")
		w.Write(cached.data)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[HYDRA-PROXY] Failed to fetch %s: %v", sourceURL, err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		log.Printf("[HYDRA-PROXY] Upstream returned %d for %s", resp.StatusCode, sourceURL)
		http.Error(w, fmt.Sprintf(`{"error":"upstream %d","url":"%s"}`, resp.StatusCode, sourceURL), http.StatusBadGateway)
		return
	}

	hydraCacheLock.Lock()
	hydraCache[sourceURL] = hydraCacheEntry{data: data, expiresAt: time.Now().Add(1 * time.Hour)}
	hydraCacheLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "miss")
	w.Write(data)
}

func handleAllHydraSources(w http.ResponseWriter, r *http.Request) {
	type sourceResult struct {
		URL  string `json:"url"`
		Data []byte `json:"-"`
		Err  string `json:"error,omitempty"`
	}

	results := make([]sourceResult, len(hydraSourceUrls))
	var wg sync.WaitGroup

	for i, url := range hydraSourceUrls {
		wg.Add(1)
		go func(idx int, srcURL string) {
			defer wg.Done()

			hydraCacheLock.Lock()
			cached, ok := hydraCache[srcURL]
			hydraCacheLock.Unlock()

			if ok && time.Now().Before(cached.expiresAt) {
				results[idx] = sourceResult{URL: srcURL, Data: cached.data}
				return
			}

			client := &http.Client{Timeout: 30 * time.Second}
			req, err := http.NewRequest("GET", srcURL, nil)
			if err != nil {
				results[idx] = sourceResult{URL: srcURL, Err: err.Error()}
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "application/json, text/plain, */*")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")

			resp, err := client.Do(req)
			if err != nil {
				results[idx] = sourceResult{URL: srcURL, Err: err.Error()}
				return
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				results[idx] = sourceResult{URL: srcURL, Err: err.Error()}
				return
			}

			if resp.StatusCode == 200 {
				hydraCacheLock.Lock()
				hydraCache[srcURL] = hydraCacheEntry{data: data, expiresAt: time.Now().Add(1 * time.Hour)}
				hydraCacheLock.Unlock()
			}

			results[idx] = sourceResult{URL: srcURL, Data: data}
		}(i, url)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("["))
	first := true
	for _, res := range results {
		if res.Err != "" || len(res.Data) == 0 {
			log.Printf("[HYDRA-PROXY] Skipping %s: err=%s", res.URL, res.Err)
			continue
		}
		if !first {
			w.Write([]byte(","))
		}
		first = false
		w.Write(res.Data)
	}
	w.Write([]byte("]"))
}
