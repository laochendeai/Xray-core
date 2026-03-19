package webpanel

import (
	"net/http"

	loggerservice "github.com/xtls/xray-core/app/log/command"
)

// handleLoggerRestart handles POST /api/v1/logger/restart.
func (wp *WebPanel) handleLoggerRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := wp.grpcClient.Logger().RestartLogger(wp.grpcClient.Context(), &loggerservice.RestartLoggerRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to restart logger: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Logger restarted successfully",
	})
}
