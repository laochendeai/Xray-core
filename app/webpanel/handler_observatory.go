package webpanel

import (
	"net/http"

	observatoryservice "github.com/xtls/xray-core/app/observatory/command"
)

// handleObservatoryStatus handles GET /api/v1/observatory/status.
func (wp *WebPanel) handleObservatoryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := wp.grpcClient.Observatory().GetOutboundStatus(wp.grpcClient.Context(), &observatoryservice.GetOutboundStatusRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get observatory status: "+err.Error())
		return
	}

	status := make([]map[string]interface{}, 0)
	if resp.Status != nil {
		for _, s := range resp.Status.Status {
			status = append(status, map[string]interface{}{
				"outboundTag": s.OutboundTag,
				"alive":       s.Alive,
				"delay":       s.Delay,
				"lastSeenTime": s.LastSeenTime,
				"lastTryTime":  s.LastTryTime,
				"lastErrorReason": s.LastErrorReason,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": status,
	})
}
