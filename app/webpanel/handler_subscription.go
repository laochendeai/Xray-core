package webpanel

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xtls/xray-core/common/uuid"
)

// handleShareGenerate handles POST /api/v1/share/generate.
func (wp *WebPanel) handleShareGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShareLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	link, err := GenerateShareLink(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to generate share link: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"link": link,
	})
}

// handleSubscription handles GET /sub/:token (public endpoint).
func (wp *WebPanel) handleSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/sub/")
	if token == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Validate token (simple approach - token is a hash of the JWT secret)
	if wp.config.JwtSecret == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Token must match a predefined subscription token
	expectedToken := generateSubToken(wp.config.JwtSecret)
	if token != expectedToken {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Generate subscription content from config file
	if wp.config.ConfigPath == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	cfm := NewConfigFileManager(wp.config.ConfigPath)
	configData, err := cfm.ReadConfig()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	links, err := GenerateSubscriptionLinks(configData)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "inline")
	w.Write([]byte(links))
}

func generateSubToken(secret string) string {
	// Use first 8 chars of UUID generated from secret as token
	u := uuid.New()
	return u.String()[:8]
}
