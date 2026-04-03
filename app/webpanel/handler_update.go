package webpanel

import "net/http"

func (wp *WebPanel) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.releaseChecker == nil {
		writeError(w, http.StatusInternalServerError, "Update checker is not configured")
		return
	}

	refresh := r.URL.Query().Get("refresh") == "true"
	status := wp.releaseChecker.Check(r.Context(), refresh)
	writeJSON(w, http.StatusOK, status)
}
