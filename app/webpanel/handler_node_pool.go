package webpanel

import (
	"encoding/json"
	"net/http"
	"strings"
)

// --- Subscription Handlers ---

// handleSubscriptions handles GET /api/v1/subscriptions (list) and POST (add).
func (wp *WebPanel) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wp.listSubscriptionsHandler(w, r)
	case http.MethodPost:
		wp.addSubscriptionHandler(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleSubscriptionByID handles DELETE /api/v1/subscriptions/:id and POST /api/v1/subscriptions/:id/refresh.
func (wp *WebPanel) handleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	if id == "" {
		writeError(w, http.StatusBadRequest, "subscription ID is required")
		return
	}

	// Check for /refresh suffix
	if len(parts) >= 2 && parts[1] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		wp.refreshSubscriptionHandler(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		wp.deleteSubscriptionHandler(w, r, id)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) listSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}
	subs := wp.subManager.ListSubscriptions()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"subscriptions": subs,
	})
}

func (wp *WebPanel) addSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	var req struct {
		URL             string `json:"url"`
		Remark          string `json:"remark"`
		AutoRefresh     bool   `json:"autoRefresh"`
		RefreshInterval int    `json:"refreshIntervalMin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "URL is required")
		return
	}

	sub, err := wp.subManager.AddSubscription(req.URL, req.Remark, req.AutoRefresh, req.RefreshInterval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Subscription added successfully",
		"subscription": sub,
	})
}

func (wp *WebPanel) deleteSubscriptionHandler(w http.ResponseWriter, r *http.Request, id string) {
	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	if err := wp.subManager.DeleteSubscription(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription deleted successfully",
	})
}

func (wp *WebPanel) refreshSubscriptionHandler(w http.ResponseWriter, r *http.Request, id string) {
	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	if err := wp.subManager.RefreshSubscription(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription refreshed successfully",
	})
}

// --- Node Pool Handlers ---

// handleNodePool handles GET /api/v1/node-pool (list nodes).
func (wp *WebPanel) handleNodePool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	status := r.URL.Query().Get("status")
	nodes := wp.subManager.ListNodes(status)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
	})
}

// handleNodePoolByID handles POST /api/v1/node-pool/:id/promote, /demote, and DELETE.
func (wp *WebPanel) handleNodePoolByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/node-pool/")

	// Handle /api/v1/node-pool/config
	if strings.HasPrefix(path, "config") {
		wp.handleNodePoolConfig(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	if id == "" {
		writeError(w, http.StatusBadRequest, "node ID is required")
		return
	}

	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	// Check for action suffix
	if len(parts) >= 2 {
		action := parts[1]
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		switch action {
		case "promote":
			if err := wp.subManager.PromoteNode(id); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"message": "Node promoted successfully"})
		case "demote":
			if err := wp.subManager.DemoteNode(id); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"message": "Node demoted successfully"})
		default:
			writeError(w, http.StatusBadRequest, "unknown action: "+action)
		}
		return
	}

	// DELETE /api/v1/node-pool/:id
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := wp.subManager.DeleteNode(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Node deleted successfully"})
}

// handleNodePoolConfig handles GET/PUT /api/v1/node-pool/config.
func (wp *WebPanel) handleNodePoolConfig(w http.ResponseWriter, r *http.Request) {
	if wp.subManager == nil {
		writeError(w, http.StatusServiceUnavailable, "subscription manager not initialized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg := wp.subManager.GetValidationConfig()
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var cfg ValidationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		wp.subManager.UpdateValidationConfig(cfg)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Validation config updated"})
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
