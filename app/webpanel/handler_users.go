package webpanel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	statsservice "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// handleUsers handles /api/v1/users/ endpoint for cross-inbound user management.
func (wp *WebPanel) handleUsers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	parts := strings.SplitN(path, "/", 3)
	email := strings.TrimSpace(parts[0])

	if len(parts) >= 2 && parts[1] == "reset-traffic" {
		if email == "" {
			writeError(w, http.StatusBadRequest, "user email is required")
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		wp.resetUserTraffic(w, r, email)
		return
	}

	switch r.Method {
	case http.MethodGet:
		wp.listAllUsers(w, r)
	case http.MethodDelete:
		if email == "" {
			writeError(w, http.StatusBadRequest, "user email is required")
			return
		}
		wp.deleteUser(w, r, email)
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
	case http.MethodPut:
		if email == "" {
			writeError(w, http.StatusBadRequest, "user email is required for update")
			return
		}
		wp.updateInboundUser(w, r, tag, email)
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
				"email":      user.Email,
				"level":      user.Level,
				"inboundTag": ib.Tag,
				"online":     onlineUsers[user.Email],
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
				userInfo["protocol"] = messageProtocolLabel(user.Account.Type)
				if _, account, err := encodeJSONTypedMessage(user.Account); err == nil {
					userInfo["account"] = account
				}
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
		u, err := editableUserPayload(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to decode inbound user: "+err.Error())
			return
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

	user, err := buildProtocolUser(req, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user config: "+err.Error())
		return
	}

	_, err = wp.grpcClient.Handler().AlterInbound(wp.grpcClient.Context(), &handlerservice.AlterInboundRequest{
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

func (wp *WebPanel) updateInboundUser(w http.ResponseWriter, r *http.Request, tag, email string) {
	var req inboundUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	currentUser, err := wp.getInboundUser(tag, email)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	nextUser, err := buildProtocolUser(req, email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user config: "+err.Error())
		return
	}

	ctx := wp.grpcClient.Context()
	if _, err := wp.grpcClient.Handler().AlterInbound(ctx, &handlerservice.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&handlerservice.RemoveUserOperation{
			Email: email,
		}),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove user before update: "+err.Error())
		return
	}

	if _, err := wp.grpcClient.Handler().AlterInbound(ctx, &handlerservice.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&handlerservice.AddUserOperation{
			User: nextUser,
		}),
	}); err != nil {
		if _, rollbackErr := wp.grpcClient.Handler().AlterInbound(ctx, &handlerservice.AlterInboundRequest{
			Tag: tag,
			Operation: serial.ToTypedMessage(&handlerservice.AddUserOperation{
				User: currentUser,
			}),
		}); rollbackErr != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error()+"; rollback failed: "+rollbackErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update user: "+err.Error())
		return
	}

	userPayload, err := editableUserPayload(nextUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to encode updated user: "+err.Error())
		return
	}
	userPayload["inboundTag"] = tag

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"user":    userPayload,
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

func (wp *WebPanel) resetUserTraffic(w http.ResponseWriter, r *http.Request, email string) {
	uplink, err := wp.resetUserCounter("user>>>" + email + ">>>traffic>>>uplink")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset uplink traffic: "+err.Error())
		return
	}

	downlink, err := wp.resetUserCounter("user>>>" + email + ">>>traffic>>>downlink")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset downlink traffic: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "User traffic reset successfully",
		"email":    email,
		"uplink":   uplink,
		"downlink": downlink,
	})
}

type inboundUserRequest struct {
	Email       string          `json:"email"`
	Level       uint32          `json:"level"`
	AccountType string          `json:"accountType"`
	Account     json.RawMessage `json:"account"`
}

func buildProtocolUser(req inboundUserRequest, fallbackEmail string) (*protocol.User, error) {
	email := strings.TrimSpace(req.Email)
	if email == "" {
		email = strings.TrimSpace(fallbackEmail)
	}
	if email == "" {
		return nil, fmt.Errorf("user email is required")
	}

	user := &protocol.User{
		Email: email,
		Level: req.Level,
	}

	if strings.TrimSpace(req.AccountType) != "" {
		account, err := decodeJSONTypedMessage(req.AccountType, req.Account)
		if err != nil {
			return nil, err
		}
		user.Account = account
	}

	return user, nil
}

func editableUserPayload(user *protocol.User) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"email": user.Email,
		"level": user.Level,
	}

	if user.Account != nil {
		payload["accountType"] = user.Account.Type
		payload["protocol"] = messageProtocolLabel(user.Account.Type)
		if _, account, err := encodeJSONTypedMessage(user.Account); err != nil {
			return nil, err
		} else {
			payload["account"] = account
		}
	}

	return payload, nil
}

func (wp *WebPanel) getInboundUser(tag, email string) (*protocol.User, error) {
	resp, err := wp.grpcClient.Handler().GetInboundUsers(wp.grpcClient.Context(), &handlerservice.GetInboundUserRequest{
		Tag:   tag,
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("get inbound user: %w", err)
	}
	if len(resp.Users) == 0 || resp.Users[0] == nil || strings.TrimSpace(resp.Users[0].Email) == "" {
		return nil, fmt.Errorf("user %q not found in inbound %q", email, tag)
	}
	return resp.Users[0], nil
}

func (wp *WebPanel) resetUserCounter(name string) (int64, error) {
	resp, err := wp.grpcClient.Stats().GetStats(wp.grpcClient.Context(), &statsservice.GetStatsRequest{
		Name:   name,
		Reset_: true,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, nil
		}
		return 0, err
	}
	if resp == nil || resp.Stat == nil {
		return 0, nil
	}
	return resp.Stat.Value, nil
}
