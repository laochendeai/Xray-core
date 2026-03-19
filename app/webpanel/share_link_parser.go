package webpanel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseShareLinkURI parses a share link URI into a ShareLinkRequest.
func ParseShareLinkURI(uri string) (*ShareLinkRequest, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, fmt.Errorf("empty URI")
	}

	idx := strings.Index(uri, "://")
	if idx < 0 {
		return nil, fmt.Errorf("invalid URI scheme: %s", uri)
	}
	scheme := strings.ToLower(uri[:idx])

	switch scheme {
	case "vless":
		return parseVLESSURI(uri)
	case "vmess":
		return parseVMessURI(uri)
	case "trojan":
		return parseTrojanURI(uri)
	case "ss":
		return parseSSURI(uri)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", scheme)
	}
}

// ParseSubscriptionContent decodes base64 subscription content and parses each line.
func ParseSubscriptionContent(raw string) ([]*ShareLinkRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty subscription content")
	}

	// Try base64 decode
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			// Maybe it's already plain text (lines of URIs)
			decoded = []byte(raw)
		}
	}

	lines := strings.Split(strings.TrimSpace(string(decoded)), "\n")
	var results []*ShareLinkRequest

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		req, err := ParseShareLinkURI(line)
		if err != nil {
			continue // skip unparseable lines
		}
		results = append(results, req)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid share links found in subscription content")
	}
	return results, nil
}

// parseVLESSURI parses vless://uuid@host:port?params#remark
func parseVLESSURI(uri string) (*ShareLinkRequest, error) {
	// Remove scheme
	body := uri[len("vless://"):]

	// Extract remark from fragment
	remark := ""
	if idx := strings.LastIndex(body, "#"); idx >= 0 {
		remark, _ = url.PathUnescape(body[idx+1:])
		body = body[:idx]
	}

	// Split user@host:port?params
	atIdx := strings.Index(body, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("invalid VLESS URI: missing @")
	}
	uuid := body[:atIdx]
	rest := body[atIdx+1:]

	// Split host:port and query
	queryStr := ""
	if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
		queryStr = rest[qIdx+1:]
		rest = rest[:qIdx]
	}

	host, portStr, err := splitHostPort(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid VLESS URI host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid VLESS URI port: %w", err)
	}

	params, _ := url.ParseQuery(queryStr)

	req := &ShareLinkRequest{
		Protocol:    "vless",
		UUID:        uuid,
		Address:     host,
		Port:        port,
		Remark:      remark,
		Type:        params.Get("type"),
		Security:    params.Get("encryption"),
		TLS:         params.Get("security"),
		Flow:        params.Get("flow"),
		SNI:         params.Get("sni"),
		ALPN:        params.Get("alpn"),
		Fingerprint: params.Get("fp"),
		Host:        params.Get("host"),
		Path:        params.Get("path"),
		PublicKey:   params.Get("pbk"),
		ShortID:     params.Get("sid"),
		SpiderX:     params.Get("spx"),
	}

	if req.Type == "" {
		req.Type = "tcp"
	}

	return req, nil
}

// parseVMessURI parses vmess://base64(json)
func parseVMessURI(uri string) (*ShareLinkRequest, error) {
	body := uri[len("vmess://"):]

	decoded, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(body)
		if err != nil {
			return nil, fmt.Errorf("invalid VMess URI: base64 decode failed: %w", err)
		}
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return nil, fmt.Errorf("invalid VMess URI: JSON parse failed: %w", err)
	}

	port := 0
	switch v := cfg["port"].(type) {
	case string:
		port, _ = strconv.Atoi(v)
	case float64:
		port = int(v)
	}

	req := &ShareLinkRequest{
		Protocol:    "vmess",
		UUID:        getStr(cfg, "id"),
		Address:     getStr(cfg, "add"),
		Port:        port,
		Remark:      getStr(cfg, "ps"),
		Security:    getStr(cfg, "scy"),
		Type:        getStr(cfg, "net"),
		Host:        getStr(cfg, "host"),
		Path:        getStr(cfg, "path"),
		TLS:         getStr(cfg, "tls"),
		SNI:         getStr(cfg, "sni"),
		ALPN:        getStr(cfg, "alpn"),
		Fingerprint: getStr(cfg, "fp"),
	}

	if req.Type == "" {
		req.Type = "tcp"
	}
	if req.Security == "" {
		req.Security = "auto"
	}

	return req, nil
}

// parseTrojanURI parses trojan://password@host:port?params#remark
func parseTrojanURI(uri string) (*ShareLinkRequest, error) {
	body := uri[len("trojan://"):]

	remark := ""
	if idx := strings.LastIndex(body, "#"); idx >= 0 {
		remark, _ = url.PathUnescape(body[idx+1:])
		body = body[:idx]
	}

	atIdx := strings.Index(body, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("invalid Trojan URI: missing @")
	}
	password, _ := url.PathUnescape(body[:atIdx])
	rest := body[atIdx+1:]

	queryStr := ""
	if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
		queryStr = rest[qIdx+1:]
		rest = rest[:qIdx]
	}

	host, portStr, err := splitHostPort(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid Trojan URI host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Trojan URI port: %w", err)
	}

	params, _ := url.ParseQuery(queryStr)

	req := &ShareLinkRequest{
		Protocol:    "trojan",
		Password:    password,
		Address:     host,
		Port:        port,
		Remark:      remark,
		Type:        params.Get("type"),
		TLS:         params.Get("security"),
		SNI:         params.Get("sni"),
		ALPN:        params.Get("alpn"),
		Fingerprint: params.Get("fp"),
		Host:        params.Get("host"),
		Path:        params.Get("path"),
		PublicKey:   params.Get("pbk"),
		ShortID:     params.Get("sid"),
	}

	if req.Type == "" {
		req.Type = "tcp"
	}
	if req.TLS == "" {
		req.TLS = "tls"
	}

	return req, nil
}

// parseSSURI parses ss://base64(method:password)@host:port#remark
func parseSSURI(uri string) (*ShareLinkRequest, error) {
	body := uri[len("ss://"):]

	remark := ""
	if idx := strings.LastIndex(body, "#"); idx >= 0 {
		remark, _ = url.PathUnescape(body[idx+1:])
		body = body[:idx]
	}

	var method, password, host, portStr string

	if atIdx := strings.LastIndex(body, "@"); atIdx >= 0 {
		// SIP002 format: ss://base64(method:password)@host:port
		userInfo := body[:atIdx]
		serverPart := body[atIdx+1:]

		decoded, err := base64.URLEncoding.DecodeString(userInfo)
		if err != nil {
			decoded, err = base64.RawURLEncoding.DecodeString(userInfo)
			if err != nil {
				decoded, err = base64.StdEncoding.DecodeString(userInfo)
				if err != nil {
					decoded, err = base64.RawStdEncoding.DecodeString(userInfo)
					if err != nil {
						return nil, fmt.Errorf("invalid SS URI: base64 decode failed: %w", err)
					}
				}
			}
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid SS URI: expected method:password")
		}
		method = parts[0]
		password = parts[1]

		var splitErr error
		host, portStr, splitErr = splitHostPort(serverPart)
		if splitErr != nil {
			return nil, fmt.Errorf("invalid SS URI host:port: %w", splitErr)
		}
	} else {
		// Legacy format: ss://base64(method:password@host:port)
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(body)
			if err != nil {
				return nil, fmt.Errorf("invalid SS URI: base64 decode failed: %w", err)
			}
		}
		s := string(decoded)
		atIdx := strings.LastIndex(s, "@")
		if atIdx < 0 {
			return nil, fmt.Errorf("invalid SS URI: missing @")
		}
		parts := strings.SplitN(s[:atIdx], ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid SS URI: expected method:password")
		}
		method = parts[0]
		password = parts[1]
		var splitErr error
		host, portStr, splitErr = splitHostPort(s[atIdx+1:])
		if splitErr != nil {
			return nil, fmt.Errorf("invalid SS URI host:port: %w", splitErr)
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SS URI port: %w", err)
	}

	return &ShareLinkRequest{
		Protocol: "shadowsocks",
		Security: method,
		Password: password,
		Address:  host,
		Port:     port,
		Remark:   remark,
	}, nil
}

// BuildOutboundJSON builds a complete Xray outbound JSON config from a ShareLinkRequest.
func BuildOutboundJSON(req *ShareLinkRequest, tag string) (json.RawMessage, error) {
	outbound := map[string]interface{}{
		"tag": tag,
	}

	switch strings.ToLower(req.Protocol) {
	case "vless":
		outbound["protocol"] = "vless"
		outbound["settings"] = buildVLESSSettings(req)
	case "vmess":
		outbound["protocol"] = "vmess"
		outbound["settings"] = buildVMessSettings(req)
	case "trojan":
		outbound["protocol"] = "trojan"
		outbound["settings"] = buildTrojanSettings(req)
	case "shadowsocks", "ss":
		outbound["protocol"] = "shadowsocks"
		outbound["settings"] = buildSSSettings(req)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}

	if stream := buildStreamSettings(req); stream != nil {
		outbound["streamSettings"] = stream
	}

	data, err := json.Marshal(outbound)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbound config: %w", err)
	}
	return json.RawMessage(data), nil
}

func buildVLESSSettings(req *ShareLinkRequest) map[string]interface{} {
	vnext := map[string]interface{}{
		"address": req.Address,
		"port":    req.Port,
		"users": []map[string]interface{}{
			{
				"id":         req.UUID,
				"encryption": "none",
				"flow":       req.Flow,
			},
		},
	}
	return map[string]interface{}{
		"vnext": []map[string]interface{}{vnext},
	}
}

func buildVMessSettings(req *ShareLinkRequest) map[string]interface{} {
	security := req.Security
	if security == "" {
		security = "auto"
	}
	vnext := map[string]interface{}{
		"address": req.Address,
		"port":    req.Port,
		"users": []map[string]interface{}{
			{
				"id":       req.UUID,
				"alterId":  0,
				"security": security,
			},
		},
	}
	return map[string]interface{}{
		"vnext": []map[string]interface{}{vnext},
	}
}

func buildTrojanSettings(req *ShareLinkRequest) map[string]interface{} {
	password := req.Password
	if password == "" {
		password = req.UUID
	}
	server := map[string]interface{}{
		"address":  req.Address,
		"port":     req.Port,
		"password": password,
	}
	return map[string]interface{}{
		"servers": []map[string]interface{}{server},
	}
}

func buildSSSettings(req *ShareLinkRequest) map[string]interface{} {
	method := req.Security
	if method == "" {
		method = "aes-256-gcm"
	}
	server := map[string]interface{}{
		"address":  req.Address,
		"port":     req.Port,
		"method":   method,
		"password": req.Password,
	}
	return map[string]interface{}{
		"servers": []map[string]interface{}{server},
	}
}

func buildStreamSettings(req *ShareLinkRequest) map[string]interface{} {
	stream := map[string]interface{}{}

	network := req.Type
	if network == "" {
		network = "tcp"
	}
	stream["network"] = network

	// Transport settings
	switch network {
	case "ws":
		ws := map[string]interface{}{}
		if req.Path != "" {
			ws["path"] = req.Path
		}
		if req.Host != "" {
			ws["headers"] = map[string]string{"Host": req.Host}
		}
		stream["wsSettings"] = ws
	case "grpc":
		grpc := map[string]interface{}{}
		if req.Path != "" {
			grpc["serviceName"] = req.Path
		}
		stream["grpcSettings"] = grpc
	case "h2", "http":
		h2 := map[string]interface{}{}
		if req.Path != "" {
			h2["path"] = req.Path
		}
		if req.Host != "" {
			h2["host"] = []string{req.Host}
		}
		stream["httpSettings"] = h2
	case "kcp", "mkcp":
		kcp := map[string]interface{}{}
		if req.Path != "" {
			kcp["seed"] = req.Path
		}
		stream["kcpSettings"] = kcp
	case "quic":
		quic := map[string]interface{}{}
		if req.Path != "" {
			quic["key"] = req.Path
		}
		if req.Host != "" {
			quic["security"] = req.Host
		}
		stream["quicSettings"] = quic
	case "httpupgrade":
		hu := map[string]interface{}{}
		if req.Path != "" {
			hu["path"] = req.Path
		}
		if req.Host != "" {
			hu["host"] = req.Host
		}
		stream["httpupgradeSettings"] = hu
	case "splithttp":
		sh := map[string]interface{}{}
		if req.Path != "" {
			sh["path"] = req.Path
		}
		if req.Host != "" {
			sh["host"] = req.Host
		}
		stream["splithttpSettings"] = sh
	}

	// TLS / REALITY settings
	switch req.TLS {
	case "tls":
		stream["security"] = "tls"
		tls := map[string]interface{}{}
		if req.SNI != "" {
			tls["serverName"] = req.SNI
		}
		if req.ALPN != "" {
			tls["alpn"] = strings.Split(req.ALPN, ",")
		}
		if req.Fingerprint != "" {
			tls["fingerprint"] = req.Fingerprint
		}
		stream["tlsSettings"] = tls
	case "reality":
		stream["security"] = "reality"
		reality := map[string]interface{}{}
		if req.SNI != "" {
			reality["serverName"] = req.SNI
		}
		if req.Fingerprint != "" {
			reality["fingerprint"] = req.Fingerprint
		}
		if req.PublicKey != "" {
			reality["publicKey"] = req.PublicKey
		}
		if req.ShortID != "" {
			reality["shortId"] = req.ShortID
		}
		if req.SpiderX != "" {
			reality["spiderX"] = req.SpiderX
		}
		stream["realitySettings"] = reality
	}

	return stream
}

// splitHostPort splits host:port, handling IPv6 brackets.
func splitHostPort(s string) (host, port string, err error) {
	if strings.HasPrefix(s, "[") {
		// IPv6: [host]:port
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", fmt.Errorf("missing closing bracket in IPv6 address")
		}
		host = s[1:end]
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") {
			return "", "", fmt.Errorf("missing port after IPv6 address")
		}
		port = rest[1:]
		return host, port, nil
	}

	// Check for multiple colons (IPv6 without brackets, unusual but handle it)
	lastColon := strings.LastIndex(s, ":")
	if lastColon < 0 {
		return "", "", fmt.Errorf("missing port in address: %s", s)
	}
	return s[:lastColon], s[lastColon+1:], nil
}

func getStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}
