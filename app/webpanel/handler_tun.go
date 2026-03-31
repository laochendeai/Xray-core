package webpanel

import (
	"encoding/json"
	"net/http"
)

func (wp *WebPanel) handleTunStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	writeJSON(w, http.StatusOK, wp.tunStatusSnapshot())
}

func (wp *WebPanel) handleTunSettings(w http.ResponseWriter, r *http.Request) {
	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		settings, err := wp.tunManager.EditableSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load TUN settings: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		var req TunEditableSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		settings, err := wp.tunManager.UpdateEditableSettings(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save TUN settings: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, settings)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) handleTunStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	writeTunStatusResponse(w, wp.startTransparentMode())
}

func (wp *WebPanel) handleTunStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	writeTunStatusResponse(w, wp.restoreClean(
		MachineReasonOperatorRestoreClean,
		MachineReasonOperatorRestoreClean,
		EventActorOperator,
		"restore clean requested from transparent mode toggle",
	))
}

func (wp *WebPanel) handleTunRestoreClean(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	writeTunStatusResponse(w, wp.restoreClean(
		MachineReasonOperatorRestoreClean,
		MachineReasonOperatorRestoreClean,
		EventActorOperator,
		"restore clean requested explicitly",
	))
}

func (wp *WebPanel) handleTunToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	if wp.tunStatusSnapshot().Running {
		writeTunStatusResponse(w, wp.restoreClean(
			MachineReasonOperatorRestoreClean,
			MachineReasonOperatorRestoreClean,
			EventActorOperator,
			"restore clean requested from toggle",
		))
		return
	}
	writeTunStatusResponse(w, wp.startTransparentMode())
}

func (wp *WebPanel) handleTunInstallPrivilege(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if wp.tunManager == nil {
		writeError(w, http.StatusServiceUnavailable, "TUN manager is not configured")
		return
	}

	if ok := wp.enforceTunAccess(w, r); !ok {
		return
	}

	writeTunStatusResponse(w, wp.decorateTunStatus(wp.tunManager.InstallPrivilege()))
}

func (wp *WebPanel) enforceTunAccess(w http.ResponseWriter, r *http.Request) bool {
	allowed, settings, err := wp.tunManager.IsRequestAllowed(r.RemoteAddr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to evaluate TUN access: "+err.Error())
		return false
	}
	if allowed {
		return true
	}

	message := "TUN control is limited to local browser requests for safety"
	if settings != nil && settings.AllowRemote {
		message = "TUN control is blocked"
	}
	writeError(w, http.StatusForbidden, message)
	return false
}

func writeTunStatusResponse(w http.ResponseWriter, status *TunStatus) {
	code := http.StatusOK
	if status == nil {
		code = http.StatusInternalServerError
	} else {
		switch status.Status {
		case "blocked":
			code = http.StatusConflict
		case "error", "unavailable":
			code = http.StatusInternalServerError
		}
	}
	writeJSON(w, code, status)
}
