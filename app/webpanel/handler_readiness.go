package webpanel

import "net/http"

func (wp *WebPanel) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, wp.readinessSnapshot(r.Context()))
}
