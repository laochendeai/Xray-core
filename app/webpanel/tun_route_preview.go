package webpanel

import (
	"encoding/json"
	"fmt"
	stdnet "net"
	"strconv"
	"strings"
)

type tunRoutePreviewRequest struct {
	Domain     string
	IP         string
	Port       int32
	Network    string
	SourceIP   string
	SourcePort int32
	Protocol   string
	User       string
	InboundTag string
}

type tunRoutePreviewResult struct {
	InboundTag       string `json:"inboundTag,omitempty"`
	OutboundTag      string `json:"outboundTag,omitempty"`
	BalancerTag      string `json:"balancerTag,omitempty"`
	Network          string `json:"network,omitempty"`
	User             string `json:"user,omitempty"`
	TargetDomain     string `json:"targetDomain,omitempty"`
	TargetPort       int32  `json:"targetPort,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
	MatchedRuleIndex int    `json:"matchedRuleIndex"`
}

func previewTunRoute(runtimeConfig []byte, req tunRoutePreviewRequest) (tunRoutePreviewResult, error) {
	req = normalizeTunRoutePreviewRequest(req)

	var config map[string]interface{}
	if err := json.Unmarshal(runtimeConfig, &config); err != nil {
		return tunRoutePreviewResult{}, fmt.Errorf("parse TUN runtime config: %w", err)
	}

	routing, _ := config["routing"].(map[string]interface{})
	rules, _ := routing["rules"].([]interface{})
	if len(rules) == 0 {
		return tunRoutePreviewResult{}, fmt.Errorf("TUN runtime config has no routing rules")
	}

	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}
		if !matchesTunPreviewRule(rule, req) {
			continue
		}

		result := tunRoutePreviewResult{
			InboundTag:       req.InboundTag,
			Network:          strings.ToUpper(req.Network),
			User:             req.User,
			TargetDomain:     req.Domain,
			TargetPort:       req.Port,
			Protocol:         req.Protocol,
			MatchedRuleIndex: index,
		}
		if outboundTag, ok := rule["outboundTag"].(string); ok {
			result.OutboundTag = outboundTag
		}
		if balancerTag, ok := rule["balancerTag"].(string); ok {
			result.BalancerTag = balancerTag
		}
		return result, nil
	}

	return tunRoutePreviewResult{}, fmt.Errorf("no TUN routing rule matched the request")
}

func normalizeTunRoutePreviewRequest(req tunRoutePreviewRequest) tunRoutePreviewRequest {
	req.Domain = strings.Trim(strings.ToLower(strings.TrimSpace(req.Domain)), ".")
	req.IP = strings.TrimSpace(req.IP)
	req.SourceIP = strings.TrimSpace(req.SourceIP)
	req.User = strings.TrimSpace(req.User)
	req.Protocol = strings.TrimSpace(req.Protocol)
	req.InboundTag = strings.TrimSpace(req.InboundTag)
	if req.InboundTag == "" {
		req.InboundTag = "tun-in"
	}
	req.Network = strings.ToLower(strings.TrimSpace(req.Network))
	if req.Network == "" {
		req.Network = "tcp"
	}
	if req.Port <= 0 {
		req.Port = 443
	}
	return req
}

func matchesTunPreviewRule(rule map[string]interface{}, req tunRoutePreviewRequest) bool {
	if len(rule) == 0 || tunRoutePreviewRuleHasUnsupportedMatchers(rule) {
		return false
	}
	if !matchesTunPreviewInboundTags(rule["inboundTag"], req.InboundTag) {
		return false
	}
	if !matchesTunPreviewNetwork(rule["network"], req.Network) {
		return false
	}
	if !matchesTunPreviewPort(rule["port"], req.Port) {
		return false
	}
	if !matchesTunPreviewPort(rule["sourcePort"], req.SourcePort) {
		return false
	}
	if !matchesTunPreviewDomains(rule["domain"], req.Domain) {
		return false
	}
	if !matchesTunPreviewIPs(rule["ip"], req.IP) {
		return false
	}
	if !matchesTunPreviewIPs(rule["source"], req.SourceIP) {
		return false
	}
	if !matchesTunPreviewStringField(rule["protocol"], req.Protocol) {
		return false
	}
	if !matchesTunPreviewStringField(rule["user"], req.User) {
		return false
	}
	return true
}

func tunRoutePreviewRuleHasUnsupportedMatchers(rule map[string]interface{}) bool {
	for key := range rule {
		switch key {
		case "type", "outboundTag", "balancerTag", "ruleTag", "domain", "ip", "port", "network", "inboundTag", "protocol", "user", "source", "sourcePort":
			continue
		default:
			return true
		}
	}
	return false
}

func matchesTunPreviewInboundTags(raw interface{}, inboundTag string) bool {
	tags := stringSliceFromAny(raw)
	if len(tags) == 0 {
		return true
	}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == inboundTag {
			return true
		}
	}
	return false
}

func matchesTunPreviewNetwork(raw interface{}, network string) bool {
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" || value == "<nil>" {
		return true
	}
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), network) {
			return true
		}
	}
	return false
}

func matchesTunPreviewStringField(raw interface{}, value string) bool {
	if raw == nil {
		return true
	}

	expected := stringSliceFromAny(raw)
	if len(expected) == 0 {
		trimmed := strings.TrimSpace(fmt.Sprint(raw))
		if trimmed == "" || trimmed == "<nil>" {
			return true
		}
		expected = []string{trimmed}
	}
	if strings.TrimSpace(value) == "" {
		return false
	}

	for _, candidate := range expected {
		if strings.EqualFold(strings.TrimSpace(candidate), value) {
			return true
		}
	}
	return false
}

func matchesTunPreviewPort(raw interface{}, port int32) bool {
	if raw == nil {
		return true
	}
	if port <= 0 {
		return false
	}

	contains := func(start, end int32) bool {
		return port >= start && port <= end
	}

	switch typed := raw.(type) {
	case float64:
		return int32(typed) == port
	case int:
		return int32(typed) == port
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return true
		}
		for _, segment := range strings.Split(value, ",") {
			part := strings.TrimSpace(segment)
			if part == "" {
				continue
			}
			if strings.Contains(part, "-") {
				bounds := strings.SplitN(part, "-", 2)
				start, startErr := strconv.Atoi(strings.TrimSpace(bounds[0]))
				end, endErr := strconv.Atoi(strings.TrimSpace(bounds[1]))
				if startErr == nil && endErr == nil && contains(int32(start), int32(end)) {
					return true
				}
				continue
			}
			if candidate, err := strconv.Atoi(part); err == nil && int32(candidate) == port {
				return true
			}
		}
	}

	return false
}

func matchesTunPreviewDomains(raw interface{}, domain string) bool {
	patterns := stringSliceFromAny(raw)
	if len(patterns) == 0 {
		return true
	}
	if domain == "" {
		return false
	}

	for _, pattern := range patterns {
		if matchesTunPreviewDomainRule(pattern, domain) {
			return true
		}
	}
	return false
}

func matchesTunPreviewDomainRule(pattern string, domain string) bool {
	normalizedPattern := strings.ToLower(normalizeTunDomainRule(pattern))
	normalizedDomain := strings.Trim(strings.ToLower(strings.TrimSpace(domain)), ".")
	if normalizedPattern == "" || normalizedDomain == "" {
		return false
	}

	switch {
	case strings.HasPrefix(normalizedPattern, "full:"):
		return normalizedDomain == strings.Trim(strings.TrimPrefix(normalizedPattern, "full:"), ".")
	case strings.HasPrefix(normalizedPattern, "domain:"):
		host := strings.Trim(strings.TrimPrefix(normalizedPattern, "domain:"), ".")
		return normalizedDomain == host || strings.HasSuffix(normalizedDomain, "."+host)
	case strings.HasPrefix(normalizedPattern, "keyword:"):
		return strings.Contains(normalizedDomain, strings.TrimSpace(strings.TrimPrefix(normalizedPattern, "keyword:")))
	case strings.HasPrefix(normalizedPattern, "regexp:"), strings.HasPrefix(normalizedPattern, "geosite:"):
		return false
	default:
		return normalizedDomain == strings.Trim(normalizedPattern, ".")
	}
}

func matchesTunPreviewIPs(raw interface{}, candidate string) bool {
	patterns := stringSliceFromAny(raw)
	if len(patterns) == 0 {
		return true
	}
	if strings.TrimSpace(candidate) == "" {
		return false
	}

	ip := stdnet.ParseIP(strings.TrimSpace(candidate))
	if ip == nil {
		return false
	}

	for _, pattern := range patterns {
		if matchesTunPreviewIPRule(pattern, ip) {
			return true
		}
	}
	return false
}

func matchesTunPreviewIPRule(pattern string, candidate stdnet.IP) bool {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "geoip:") {
		return false
	}
	if strings.Contains(trimmed, "/") {
		_, network, err := stdnet.ParseCIDR(trimmed)
		if err != nil {
			return false
		}
		return network.Contains(candidate)
	}
	ip := stdnet.ParseIP(trimmed)
	if ip == nil {
		return false
	}
	return ip.Equal(candidate)
}
