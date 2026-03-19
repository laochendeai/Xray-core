package webpanel

import (
	"encoding/json"
	"io"
	"net/http"
)

// handleConfig handles GET/PUT /api/v1/config.
func (wp *WebPanel) handleConfig(w http.ResponseWriter, r *http.Request) {
	configPath := wp.config.ConfigPath
	if configPath == "" {
		writeError(w, http.StatusBadRequest, "Config path not set in webpanel configuration")
		return
	}

	cfm := NewConfigFileManager(configPath)

	switch r.Method {
	case http.MethodGet:
		data, err := cfm.ReadConfig()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read config: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"config": data,
		})

	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Failed to read request body")
			return
		}

		var req struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}

		if err := cfm.WriteConfig(req.Config); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to write config: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Config saved successfully",
		})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfigReload handles POST /api/v1/config/reload.
func (wp *WebPanel) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Note: In a real implementation, this would restart the Xray instance.
	// For now, we signal that a restart is needed.
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Config reload requested. Please restart Xray for changes to take effect.",
	})
}

// handleConfigValidate handles POST /api/v1/config/validate.
func (wp *WebPanel) handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req struct {
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	configPath := wp.config.ConfigPath
	if configPath == "" {
		writeError(w, http.StatusBadRequest, "Config path not set")
		return
	}

	cfm := NewConfigFileManager(configPath)
	if err := cfm.ValidateConfig(req.Config); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"message": "Config is valid",
	})
}

// handleConfigBackups handles GET/POST /api/v1/config/backups.
func (wp *WebPanel) handleConfigBackups(w http.ResponseWriter, r *http.Request) {
	configPath := wp.config.ConfigPath
	if configPath == "" {
		writeError(w, http.StatusBadRequest, "Config path not set")
		return
	}

	cfm := NewConfigFileManager(configPath)

	switch r.Method {
	case http.MethodGet:
		backups, err := cfm.ListBackups()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to list backups: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"backups": backups,
		})

	case http.MethodPost:
		var req struct {
			Action string `json:"action"` // "create" or "restore"
			Name   string `json:"name"`   // backup name for restore
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}

		switch req.Action {
		case "create":
			if err := cfm.CreateBackup(); err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to create backup: "+err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"message": "Backup created successfully",
			})

		case "restore":
			if req.Name == "" {
				writeError(w, http.StatusBadRequest, "Backup name is required for restore")
				return
			}
			if err := cfm.RestoreBackup(req.Name); err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to restore backup: "+err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"message": "Backup restored successfully",
			})

		default:
			writeError(w, http.StatusBadRequest, "Invalid action. Use 'create' or 'restore'")
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
