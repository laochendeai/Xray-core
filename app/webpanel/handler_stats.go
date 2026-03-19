package webpanel

import (
	"encoding/json"
	"net/http"

	statsservice "github.com/xtls/xray-core/app/stats/command"
)

func (wp *WebPanel) handleSysStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := wp.grpcClient.Stats().GetSysStats(wp.grpcClient.Context(), &statsservice.SysStatsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get system stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"numGoroutine": resp.NumGoroutine,
		"numGC":        resp.NumGC,
		"alloc":        resp.Alloc,
		"totalAlloc":   resp.TotalAlloc,
		"sys":          resp.Sys,
		"mallocs":      resp.Mallocs,
		"frees":        resp.Frees,
		"liveObjects":  resp.LiveObjects,
		"pauseTotalNs": resp.PauseTotalNs,
		"uptime":       resp.Uptime,
	})
}

func (wp *WebPanel) handleQueryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = ""
	}
	reset := r.URL.Query().Get("reset") == "true"

	resp, err := wp.grpcClient.Stats().QueryStats(wp.grpcClient.Context(), &statsservice.QueryStatsRequest{
		Pattern: pattern,
		Reset_:  reset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query stats: "+err.Error())
		return
	}

	stats := make([]map[string]interface{}, 0, len(resp.Stat))
	for _, s := range resp.Stat {
		stats = append(stats, map[string]interface{}{
			"name":  s.Name,
			"value": s.Value,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stats": stats,
	})
}

func (wp *WebPanel) handleOnlineUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := wp.grpcClient.Stats().GetAllOnlineUsers(wp.grpcClient.Context(), &statsservice.GetAllOnlineUsersRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get online users: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": resp.Users,
		"count": len(resp.Users),
	})
}

func (wp *WebPanel) handleOnlineIPs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		writeError(w, http.StatusBadRequest, "email parameter is required")
		return
	}

	resp, err := wp.grpcClient.Stats().GetStatsOnlineIpList(wp.grpcClient.Context(), &statsservice.GetStatsRequest{
		Name: email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get online IPs: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name": resp.Name,
		"ips":  resp.Ips,
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
