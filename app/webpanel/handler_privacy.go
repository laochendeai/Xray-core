package webpanel

import "net/http"

type PrivacyDiagnosticsContextResponse struct {
	Supported   bool                `json:"supported"`
	Unsupported string              `json:"unsupportedReason,omitempty"`
	TunStatus   *TunStatus          `json:"tunStatus,omitempty"`
	TunSettings *TunEditableSettings `json:"tunSettings,omitempty"`
}

func (wp *WebPanel) handlePrivacyDiagnosticsContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeJSON(w, http.StatusOK, PrivacyDiagnosticsContextResponse{
			Supported:   false,
			Unsupported: "TUN manager is not configured",
		})
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	settings, err := wp.tunManager.EditableSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load privacy diagnostics context: "+err.Error())
		return
	}

	status := wp.tunStatusSnapshot()
	writeJSON(w, http.StatusOK, PrivacyDiagnosticsContextResponse{
		Supported:   true,
		TunStatus:   status,
		TunSettings: settings,
	})
}
