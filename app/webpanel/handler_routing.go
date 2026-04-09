package webpanel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	routerservice "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
)

type routingTestRequest struct {
	Scope      string `json:"scope"`
	Domain     string `json:"domain"`
	IP         string `json:"ip"`
	Port       int32  `json:"port"`
	Network    string `json:"network"`
	SourceIP   string `json:"sourceIP"`
	SourcePort int32  `json:"sourcePort"`
	Protocol   string `json:"protocol"`
	User       string `json:"user"`
	InboundTag string `json:"inboundTag"`
}

// handleRoutingRules handles GET /api/v1/routing/rules (list) and POST (add).
func (wp *WebPanel) handleRoutingRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wp.listRoutingRules(w, r)
	case http.MethodPost:
		wp.addRoutingRule(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleRoutingRuleByTag handles DELETE /api/v1/routing/rules/:tag.
func (wp *WebPanel) handleRoutingRuleByTag(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/api/v1/routing/rules/")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "rule tag is required")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		wp.removeRoutingRule(w, r, tag)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (wp *WebPanel) listRoutingRules(w http.ResponseWriter, r *http.Request) {
	resp, err := wp.grpcClient.Routing().ListRule(wp.grpcClient.Context(), &routerservice.ListRuleRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list routing rules: "+err.Error())
		return
	}

	rules := make([]map[string]interface{}, 0, len(resp.Rules))
	for _, rule := range resp.Rules {
		ruleInfo := map[string]interface{}{
			"tag":     rule.Tag,
			"ruleTag": rule.RuleTag,
		}
		rules = append(rules, ruleInfo)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
	})
}

func (wp *WebPanel) addRoutingRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rule json.RawMessage `json:"rule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// The Config field is a TypedMessage containing the routing rule config.
	// For simplicity, we pass the raw config through.
	addReq := &routerservice.AddRuleRequest{
		Config: &serial.TypedMessage{
			Type:  "xray.app.router.RoutingRule",
			Value: req.Rule,
		},
		ShouldAppend: true,
	}

	_, err := wp.grpcClient.Routing().AddRule(wp.grpcClient.Context(), addReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to add routing rule: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Routing rule added successfully",
	})
}

func (wp *WebPanel) removeRoutingRule(w http.ResponseWriter, r *http.Request, tag string) {
	_, err := wp.grpcClient.Routing().RemoveRule(wp.grpcClient.Context(), &routerservice.RemoveRuleRequest{
		RuleTag: tag,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove routing rule: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Routing rule removed successfully",
	})
}

// handleRoutingTest handles POST /api/v1/routing/test.
func (wp *WebPanel) handleRoutingTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var testReq routingTestRequest
	if err := json.Unmarshal(body, &testReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if strings.EqualFold(strings.TrimSpace(testReq.Scope), "tun") {
		result, err := wp.previewTunRoute(testReq)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to preview TUN route: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"result": result,
		})
		return
	}

	var network net.Network
	switch strings.ToLower(testReq.Network) {
	case "tcp":
		network = net.Network_TCP
	case "udp":
		network = net.Network_UDP
	default:
		network = net.Network_TCP
	}

	routingCtx := &routerservice.RoutingContext{
		TargetDomain: testReq.Domain,
		TargetPort:   uint32(testReq.Port),
		Network:      network,
		User:         testReq.User,
		InboundTag:   testReq.InboundTag,
	}

	if testReq.IP != "" {
		routingCtx.TargetIPs = [][]byte{net.ParseAddress(testReq.IP).IP()}
	}
	if testReq.SourceIP != "" {
		routingCtx.SourceIPs = [][]byte{net.ParseAddress(testReq.SourceIP).IP()}
	}
	if testReq.SourcePort > 0 {
		routingCtx.SourcePort = uint32(testReq.SourcePort)
	}
	if testReq.Protocol != "" {
		routingCtx.Protocol = testReq.Protocol
	}

	resp, err := wp.grpcClient.Routing().TestRoute(wp.grpcClient.Context(), &routerservice.TestRouteRequest{
		RoutingContext: routingCtx,
		PublishResult:  false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to test route: "+err.Error())
		return
	}

	result := map[string]interface{}{
		"inboundTag":  resp.InboundTag,
		"outboundTag": resp.OutboundTag,
		"network":     resp.Network.String(),
		"user":        resp.User,
	}
	if resp.TargetDomain != "" {
		result["targetDomain"] = resp.TargetDomain
	}
	if resp.TargetPort != 0 {
		result["targetPort"] = resp.TargetPort
	}
	if resp.Protocol != "" {
		result["protocol"] = resp.Protocol
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result": result,
	})
}

func (wp *WebPanel) previewTunRoute(testReq routingTestRequest) (tunRoutePreviewResult, error) {
	if wp.tunManager == nil {
		return tunRoutePreviewResult{}, fmt.Errorf("TUN manager is not configured")
	}
	if wp.subManager == nil {
		return tunRoutePreviewResult{}, fmt.Errorf("subscription manager is not configured")
	}

	settings, err := wp.tunManager.SettingsSnapshot()
	if err != nil {
		return tunRoutePreviewResult{}, err
	}
	raw, err := os.ReadFile(wp.tunManager.configPath)
	if err != nil {
		return tunRoutePreviewResult{}, err
	}
	activeNodes := wp.subManager.ListNodesByStatuses(NodeStatusActive)
	runtimeConfig, err := buildTunRuntimeConfig(raw, settings, activeNodes)
	if err != nil {
		return tunRoutePreviewResult{}, err
	}

	return previewTunRoute(runtimeConfig, tunRoutePreviewRequest{
		Domain:     testReq.Domain,
		IP:         testReq.IP,
		Port:       testReq.Port,
		Network:    testReq.Network,
		SourceIP:   testReq.SourceIP,
		SourcePort: testReq.SourcePort,
		Protocol:   testReq.Protocol,
		User:       testReq.User,
		InboundTag: testReq.InboundTag,
	})
}

// handleBalancers handles GET/PUT /api/v1/routing/balancers/:tag.
func (wp *WebPanel) handleBalancers(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/api/v1/routing/balancers/")
	if tag == "" {
		writeError(w, http.StatusBadRequest, "balancer tag is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		resp, err := wp.grpcClient.Routing().GetBalancerInfo(wp.grpcClient.Context(), &routerservice.GetBalancerInfoRequest{
			Tag: tag,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get balancer info: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"balancer": resp.Balancer,
		})

	case http.MethodPut:
		var req struct {
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}

		_, err := wp.grpcClient.Routing().OverrideBalancerTarget(wp.grpcClient.Context(), &routerservice.OverrideBalancerTargetRequest{
			BalancerTag: tag,
			Target:      req.Target,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to override balancer target: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Balancer target overridden successfully",
		})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
