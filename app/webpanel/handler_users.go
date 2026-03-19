package webpanel

import (
	"encoding/json"
	"net/http"
	"strings"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	statsservice "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
)

// handleUsers handles /api/v1/users/ endpoint for cross-inbound user management.
func (wp *WebPanel) handleUsers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")

	switch r.Method {
	case http.MethodGet:
		wp.listAllUsers(w, r)
	case http.MethodDelete:
		if path == "" {
			writeError(w, http.StatusBadRequest, "user email is required")
			return
		}
		wp.deleteUser(w, r, path)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleInboundUsers handles user operations within a specific inbound.
func (wp *WebPanel) handleInboundUsers(w http.ResponseWriter, r *http.Request, tag, email string) {
	switch r.Method {
	case http.MethodGet:
		wp.getInboundUsers(w, r, tag, email)
	case http.MethodPost:
		wp.addInboundUser(w, r, tag)
	case http.MethodDelete:
		if email == "" {
			writeError(w, http.StatusBadRequest, "user email is required for deletion")
			return
		}
		wp.removeInboundUser(w, r, tag, email)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) listAllUsers(w http.ResponseWriter, r *http.Request) {
	// List inbounds first
	ibResp, err := wp.grpcClient.Handler().ListInbounds(wp.grpcClient.Context(), &handlerservice.ListInboundsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list inbounds: "+err.Error())
		return
	}

	// Get online users
	onlineResp, err := wp.grpcClient.Stats().GetAllOnlineUsers(wp.grpcClient.Context(), &statsservice.GetAllOnlineUsersRequest{})
	onlineUsers := make(map[string]bool)
	if err == nil && onlineResp != nil {
		for _, u := range onlineResp.Users {
			onlineUsers[u] = true
		}
	}

	var allUsers []map[string]interface{}

	for _, ib := range ibResp.Inbounds {
		// Get users for this inbound
		userResp, err := wp.grpcClient.Handler().GetInboundUsers(wp.grpcClient.Context(), &handlerservice.GetInboundUserRequest{
			Tag: ib.Tag,
		})
		if err != nil {
			continue
		}

		for _, user := range userResp.Users {
			userInfo := map[string]interface{}{
				"email":       user.Email,
				"level":       user.Level,
				"inboundTag":  ib.Tag,
				"online":      onlineUsers[user.Email],
			}

			// Get user traffic stats
			upStat, _ := wp.grpcClient.Stats().GetStats(wp.grpcClient.Context(), &statsservice.GetStatsRequest{
				Name: "user>>>" + user.Email + ">>>traffic>>>uplink",
			})
			if upStat != nil && upStat.Stat != nil {
				userInfo["uplink"] = upStat.Stat.Value
			}

			downStat, _ := wp.grpcClient.Stats().GetStats(wp.grpcClient.Context(), &statsservice.GetStatsRequest{
				Name: "user>>>" + user.Email + ">>>traffic>>>downlink",
			})
			if downStat != nil && downStat.Stat != nil {
				userInfo["downlink"] = downStat.Stat.Value
			}

			// Decode protocol-specific account info
			if user.Account != nil {
				userInfo["accountType"] = user.Account.Type
			}

			allUsers = append(allUsers, userInfo)
		}
	}

	if allUsers == nil {
		allUsers = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": allUsers,
	})
}

func (wp *WebPanel) getInboundUsers(w http.ResponseWriter, r *http.Request, tag, email string) {
	resp, err := wp.grpcClient.Handler().GetInboundUsers(wp.grpcClient.Context(), &handlerservice.GetInboundUserRequest{
		Tag:   tag,
		Email: email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get inbound users: "+err.Error())
		return
	}

	users := make([]map[string]interface{}, 0, len(resp.Users))
	for _, user := range resp.Users {
		u := map[string]interface{}{
			"email": user.Email,
			"level": user.Level,
		}
		if user.Account != nil {
			u["accountType"] = user.Account.Type
		}
		users = append(users, u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

func (wp *WebPanel) addInboundUser(w http.ResponseWriter, r *http.Request, tag string) {
	var req struct {
		Email       string          `json:"email"`
		Level       uint32          `json:"level"`
		AccountType string          `json:"accountType"`
		Account     json.RawMessage `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	user := &protocol.User{
		Email: req.Email,
		Level: req.Level,
	}

	if req.Account != nil && req.AccountType != "" {
		user.Account = &serial.TypedMessage{
			Type:  req.AccountType,
			Value: req.Account,
		}
	}

	_, err := wp.grpcClient.Handler().AlterInbound(wp.grpcClient.Context(), &handlerservice.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&handlerservice.AddUserOperation{
			User: user,
		}),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "User added successfully",
	})
}

func (wp *WebPanel) removeInboundUser(w http.ResponseWriter, r *http.Request, tag, email string) {
	_, err := wp.grpcClient.Handler().AlterInbound(wp.grpcClient.Context(), &handlerservice.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&handlerservice.RemoveUserOperation{
			Email: email,
		}),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "User removed successfully",
	})
}

func (wp *WebPanel) deleteUser(w http.ResponseWriter, r *http.Request, email string) {
	// Remove user from all inbounds
	ibResp, err := wp.grpcClient.Handler().ListInbounds(wp.grpcClient.Context(), &handlerservice.ListInboundsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list inbounds: "+err.Error())
		return
	}

	removed := 0
	for _, ib := range ibResp.Inbounds {
		_, err := wp.grpcClient.Handler().AlterInbound(wp.grpcClient.Context(), &handlerservice.AlterInboundRequest{
			Tag: ib.Tag,
			Operation: serial.ToTypedMessage(&handlerservice.RemoveUserOperation{
				Email: email,
			}),
		})
		if err == nil {
			removed++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "User removed from inbounds",
		"removedCount": removed,
	})
}
