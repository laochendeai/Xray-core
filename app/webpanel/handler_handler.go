package webpanel

import (
	"encoding/json"
	"net/http"
	"strings"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/serial"
)

// handleInbounds handles GET /api/v1/inbounds (list) and POST /api/v1/inbounds (add).
func (wp *WebPanel) handleInbounds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wp.listInbounds(w, r)
	case http.MethodPost:
		wp.addInbound(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleInboundByTag handles operations on a specific inbound: GET, DELETE, and user operations.
func (wp *WebPanel) handleInboundByTag(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/inbounds/")
	parts := strings.SplitN(path, "/", 3)
	tag := parts[0]

	if tag == "" {
		writeError(w, http.StatusBadRequest, "inbound tag is required")
		return
	}

	// Check if this is a user operation: /api/v1/inbounds/:tag/users[/:email]
	if len(parts) >= 2 && parts[1] == "users" {
		email := ""
		if len(parts) >= 3 {
			email = parts[2]
		}
		wp.handleInboundUsers(w, r, tag, email)
		return
	}

	switch r.Method {
	case http.MethodGet:
		wp.getInbound(w, r, tag)
	case http.MethodPut:
		wp.updateInbound(w, r, tag)
	case http.MethodDelete:
		wp.removeInbound(w, r, tag)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) listInbounds(w http.ResponseWriter, r *http.Request) {
	resp, err := wp.grpcClient.Handler().ListInbounds(wp.grpcClient.Context(), &handlerservice.ListInboundsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list inbounds: "+err.Error())
		return
	}

	inbounds := make([]map[string]interface{}, 0, len(resp.Inbounds))
	for _, ib := range resp.Inbounds {
		inbound := map[string]interface{}{
			"tag": ib.Tag,
		}

		if ib.ReceiverSettings != nil {
			inbound["receiverType"] = ib.ReceiverSettings.Type
			inbound["receiverSettings"] = decodeTypedMessage(ib.ReceiverSettings)
		}
		if ib.ProxySettings != nil {
			inbound["protocol"] = messageProtocolLabel(ib.ProxySettings.Type)
			inbound["proxyType"] = ib.ProxySettings.Type
			inbound["proxySettings"] = decodeTypedMessage(ib.ProxySettings)
		}

		inbounds = append(inbounds, inbound)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inbounds": inbounds,
	})
}

func (wp *WebPanel) addInbound(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Inbound json.RawMessage `json:"inbound"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	inboundConfig, err := decodeInboundHandlerConfig(req.Inbound)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid inbound config: "+err.Error())
		return
	}

	_, err = wp.grpcClient.Handler().AddInbound(wp.grpcClient.Context(), &handlerservice.AddInboundRequest{
		Inbound: inboundConfig,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add inbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Inbound added successfully",
	})
}

func (wp *WebPanel) getInbound(w http.ResponseWriter, r *http.Request, tag string) {
	inboundConfig, err := findInboundConfig(wp.grpcClient.Handler(), tag)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	editable, err := editableInboundFromCore(inboundConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to decode inbound config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inbound": editable,
	})
}

func (wp *WebPanel) updateInbound(w http.ResponseWriter, r *http.Request, tag string) {
	var req struct {
		Inbound json.RawMessage `json:"inbound"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	inboundConfig, err := decodeInboundHandlerConfig(req.Inbound)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid inbound config: "+err.Error())
		return
	}
	if inboundConfig.Tag != tag {
		writeError(w, http.StatusBadRequest, "Inbound tag cannot be changed during edit")
		return
	}

	if err := wp.replaceInboundConfig(tag, inboundConfig); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update inbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Inbound updated successfully",
	})
}

func (wp *WebPanel) removeInbound(w http.ResponseWriter, r *http.Request, tag string) {
	_, err := wp.grpcClient.Handler().RemoveInbound(wp.grpcClient.Context(), &handlerservice.RemoveInboundRequest{
		Tag: tag,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove inbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Inbound removed successfully",
	})
}

// handleOutbounds handles GET /api/v1/outbounds (list) and POST /api/v1/outbounds (add).
func (wp *WebPanel) handleOutbounds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wp.listOutbounds(w, r)
	case http.MethodPost:
		wp.addOutbound(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleOutboundByTag handles DELETE /api/v1/outbounds/:tag.
func (wp *WebPanel) handleOutboundByTag(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/api/v1/outbounds/")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "outbound tag is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		wp.getOutbound(w, r, tag)
	case http.MethodPut:
		wp.updateOutbound(w, r, tag)
	case http.MethodDelete:
		wp.removeOutbound(w, r, tag)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) listOutbounds(w http.ResponseWriter, r *http.Request) {
	resp, err := wp.grpcClient.Handler().ListOutbounds(wp.grpcClient.Context(), &handlerservice.ListOutboundsRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list outbounds: "+err.Error())
		return
	}

	outbounds := make([]map[string]interface{}, 0, len(resp.Outbounds))
	for _, ob := range resp.Outbounds {
		outbound := map[string]interface{}{
			"tag": ob.Tag,
		}

		if ob.SenderSettings != nil {
			outbound["senderType"] = ob.SenderSettings.Type
			outbound["senderSettings"] = decodeTypedMessage(ob.SenderSettings)
		}
		if ob.ProxySettings != nil {
			outbound["protocol"] = messageProtocolLabel(ob.ProxySettings.Type)
			outbound["proxyType"] = ob.ProxySettings.Type
			outbound["proxySettings"] = decodeTypedMessage(ob.ProxySettings)
		}
		if ob.Comment != "" {
			outbound["comment"] = ob.Comment
		}

		outbounds = append(outbounds, outbound)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"outbounds": outbounds,
	})
}

func (wp *WebPanel) getOutbound(w http.ResponseWriter, r *http.Request, tag string) {
	outboundConfig, err := findOutboundConfig(wp.grpcClient.Handler(), tag)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	editable, err := editableOutboundFromCore(outboundConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to decode outbound config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"outbound": editable,
	})
}

func (wp *WebPanel) addOutbound(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Outbound json.RawMessage `json:"outbound"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	outboundConfig, err := decodeOutboundHandlerConfig(req.Outbound)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid outbound config: "+err.Error())
		return
	}

	_, err = wp.grpcClient.Handler().AddOutbound(wp.grpcClient.Context(), &handlerservice.AddOutboundRequest{
		Outbound: outboundConfig,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add outbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Outbound added successfully",
	})
}

func (wp *WebPanel) updateOutbound(w http.ResponseWriter, r *http.Request, tag string) {
	var req struct {
		Outbound json.RawMessage `json:"outbound"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	outboundConfig, err := decodeOutboundHandlerConfig(req.Outbound)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid outbound config: "+err.Error())
		return
	}
	if outboundConfig.Tag != tag {
		writeError(w, http.StatusBadRequest, "Outbound tag cannot be changed during edit")
		return
	}

	if err := wp.replaceOutboundConfig(tag, outboundConfig); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update outbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Outbound updated successfully",
	})
}

func (wp *WebPanel) removeOutbound(w http.ResponseWriter, r *http.Request, tag string) {
	_, err := wp.grpcClient.Handler().RemoveOutbound(wp.grpcClient.Context(), &handlerservice.RemoveOutboundRequest{
		Tag: tag,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove outbound: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Outbound removed successfully",
	})
}

// decodeTypedMessage converts a TypedMessage to a JSON-friendly representation.
func decodeTypedMessage(tm *serial.TypedMessage) map[string]interface{} {
	if tm == nil {
		return nil
	}
	messageType, value, err := encodeJSONTypedMessage(tm)
	if err == nil {
		return map[string]interface{}{
			"type":  messageType,
			"value": value,
		}
	}
	return map[string]interface{}{
		"type":  tm.Type,
		"value": tm.Value,
	}
}
