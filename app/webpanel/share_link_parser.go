package webpanel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdnet "net"
	"net/url"
	"strconv"
	"strings"
)

const utf8BOM = "\ufeff"

// ParseShareLinkURI parses a share link URI into a ShareLinkRequest.
func ParseShareLinkURI(uri string) (*ShareLinkRequest, error) {
	uri = strings.TrimSpace(trimUTF8BOM(uri))
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
	case "hysteria", "hysteria2", "hy2":
		return parseHysteria2URI(uri)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", scheme)
	}
}

// ParseSubscriptionContent decodes base64 subscription content and parses each line.
func ParseSubscriptionContent(raw string) ([]*ShareLinkRequest, error) {
	raw = strings.TrimSpace(trimUTF8BOM(raw))
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

	content := strings.TrimSpace(trimUTF8BOM(string(decoded)))
	lines := strings.Split(content, "\n")
	var results []*ShareLinkRequest

	for _, line := range lines {
		line = strings.TrimSpace(trimUTF8BOM(line))
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

func trimUTF8BOM(value string) string {
	return strings.TrimPrefix(value, utf8BOM)
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
		Protocol:      "vless",
		UUID:          uuid,
		Address:       host,
		Port:          port,
		Remark:        remark,
		Type:          params.Get("type"),
		Security:      params.Get("encryption"),
		TLS:           normalizeTransportSecurityValue(params.Get("security"), params.Get("tls")),
		AllowInsecure: parseBoolishParam(params.Get("skip-cert-verify")) || parseBoolishParam(params.Get("allowInsecure")),
		Flow:          params.Get("flow"),
		SNI:           params.Get("sni"),
		ALPN:          params.Get("alpn"),
		Fingerprint:   params.Get("fp"),
		Host:          params.Get("host"),
		Path:          params.Get("path"),
		PublicKey:     firstNonEmpty(params.Get("pbk"), params.Get("pb")),
		ShortID:       params.Get("sid"),
		SpiderX:       params.Get("spx"),
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
		Protocol:      "trojan",
		Password:      password,
		Address:       host,
		Port:          port,
		Remark:        remark,
		Type:          params.Get("type"),
		TLS:           normalizeTransportSecurityValue(params.Get("security"), params.Get("tls")),
		AllowInsecure: parseBoolishParam(params.Get("skip-cert-verify")) || parseBoolishParam(params.Get("allowInsecure")),
		SNI:           params.Get("sni"),
		ALPN:          params.Get("alpn"),
		Fingerprint:   params.Get("fp"),
		Host:          params.Get("host"),
		Path:          params.Get("path"),
		PublicKey:     firstNonEmpty(params.Get("pbk"), params.Get("pb")),
		ShortID:       params.Get("sid"),
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

	queryStr := ""
	if qIdx := strings.Index(body, "?"); qIdx >= 0 {
		queryStr = body[qIdx+1:]
		body = strings.TrimSuffix(body[:qIdx], "/")
	}

	var method, password, host, portStr string

	if atIdx := strings.LastIndex(body, "@"); atIdx >= 0 {
		userInfo := body[:atIdx]
		serverPart := strings.TrimSuffix(body[atIdx+1:], "/")

		var parseErr error
		method, password, parseErr = parseSSUserInfo(userInfo)
		if parseErr != nil {
			return nil, parseErr
		}

		var splitErr error
		host, portStr, splitErr = splitHostPort(serverPart)
		if splitErr != nil {
			return nil, fmt.Errorf("invalid SS URI host:port: %w", splitErr)
		}
	} else {
		// Legacy format: ss://base64(method:password@host:port)
		s, err := decodeBase64Text(body)
		if err != nil {
			return nil, fmt.Errorf("invalid SS URI: base64 decode failed: %w", err)
		}
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

	req := &ShareLinkRequest{
		Protocol: "shadowsocks",
		Security: method,
		Password: password,
		Address:  host,
		Port:     port,
		Remark:   remark,
	}

	params, _ := url.ParseQuery(queryStr)
	if plugin := params.Get("plugin"); plugin != "" {
		if err := applyV2RayPluginToSS(req, plugin); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func parseHysteria2URI(uri string) (*ShareLinkRequest, error) {
	schemeIdx := strings.Index(uri, "://")
	if schemeIdx < 0 {
		return nil, fmt.Errorf("invalid Hysteria2 URI: missing scheme")
	}

	body := uri[schemeIdx+3:]
	remark := ""
	if idx := strings.LastIndex(body, "#"); idx >= 0 {
		remark, _ = url.PathUnescape(body[idx+1:])
		body = body[:idx]
	}

	atIdx := strings.Index(body, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("invalid Hysteria2 URI: missing @")
	}

	auth, _ := url.PathUnescape(body[:atIdx])
	rest := body[atIdx+1:]

	queryStr := ""
	if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
		queryStr = rest[qIdx+1:]
		rest = rest[:qIdx]
	}

	host, portStr, err := splitHostPort(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid Hysteria2 URI host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Hysteria2 URI port: %w", err)
	}

	params, _ := url.ParseQuery(queryStr)
	req := &ShareLinkRequest{
		Protocol:      "hysteria2",
		Version:       2,
		Password:      auth,
		Address:       host,
		Port:          port,
		Type:          "hysteria",
		TLS:           "tls",
		AllowInsecure: parseBoolishParam(params.Get("insecure")) || parseBoolishParam(params.Get("allow_insecure")) || parseBoolishParam(params.Get("allowInsecure")) || parseBoolishParam(params.Get("skip-cert-verify")),
		SNI:           params.Get("sni"),
		ALPN:          params.Get("alpn"),
		Congestion:    params.Get("congestion_control"),
		PinSHA256:     firstNonEmpty(params.Get("pinSHA256"), params.Get("pin_sha256"), params.Get("pinsha256")),
		Obfs:          firstNonEmpty(params.Get("obfs"), params.Get("obfs_type")),
		ObfsPassword:  firstNonEmpty(params.Get("obfs-password"), params.Get("obfs_password")),
		Remark:        remark,
	}

	if req.Obfs != "" {
		req.Obfs = strings.ToLower(strings.TrimSpace(req.Obfs))
		if req.Obfs != "salamander" {
			return nil, fmt.Errorf("unsupported Hysteria2 obfs mode: %s", req.Obfs)
		}
		if req.ObfsPassword == "" {
			return nil, fmt.Errorf("invalid Hysteria2 URI: obfs-password is required when obfs is set")
		}
	}

	if udpRelayMode := params.Get("udp_relay_mode"); udpRelayMode != "" {
		req.Extra = map[string]string{"udp_relay_mode": udpRelayMode}
	}

	return req, nil
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
	case "hysteria", "hysteria2":
		outbound["protocol"] = "hysteria"
		outbound["settings"] = buildHysteriaProxySettings(req)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}

	stream := buildStreamSettings(req)
	if stream == nil {
		stream = map[string]interface{}{}
	}
	outbound["streamSettings"] = stream

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

func buildHysteriaProxySettings(req *ShareLinkRequest) map[string]interface{} {
	version := req.Version
	if version == 0 {
		version = 2
	}

	return map[string]interface{}{
		"version": version,
		// infra/conf.HysteriaClientConfig expects address/port at the top level.
		// Emitting a nested server object causes config loading to panic.
		"address": req.Address,
		"port":    req.Port,
	}
}

func buildStreamSettings(req *ShareLinkRequest) map[string]interface{} {
	stream := map[string]interface{}{}

	network := req.Type
	if network == "" {
		if strings.EqualFold(req.Protocol, "hysteria") || strings.EqualFold(req.Protocol, "hysteria2") {
			network = "hysteria"
		} else {
			network = "tcp"
		}
	}
	stream["network"] = network
	security := normalizeTransportSecurityValue(req.TLS, "")
	if security == "" && (strings.EqualFold(req.Protocol, "hysteria") || strings.EqualFold(req.Protocol, "hysteria2")) {
		security = "tls"
	}
	tlsServerName := effectiveTLSServerName(req)
	realityServerName := effectiveRealityServerName(req)

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
	case "hysteria":
		hysteria := map[string]interface{}{
			"version": hysteriaVersion(req),
		}
		if req.Password != "" {
			hysteria["auth"] = req.Password
		}
		stream["hysteriaSettings"] = hysteria
		if req.Congestion != "" {
			stream["quicParams"] = map[string]interface{}{
				"congestion": req.Congestion,
			}
		}
		if req.Obfs == "salamander" && req.ObfsPassword != "" {
			stream["udpmasks"] = []map[string]interface{}{
				{
					"type":     "salamander",
					"password": req.ObfsPassword,
				},
			}
		}
	}

	// TLS / REALITY settings
	switch security {
	case "tls":
		stream["security"] = "tls"
		tls := map[string]interface{}{}
		if tlsServerName != "" {
			tls["serverName"] = tlsServerName
		}
		if req.ALPN != "" {
			tls["alpn"] = strings.Split(req.ALPN, ",")
		}
		if req.Fingerprint != "" {
			tls["fingerprint"] = req.Fingerprint
		}
		if req.AllowInsecure {
			tls["allowInsecure"] = true
		}
		if req.PinSHA256 != "" {
			tls["pinnedPeerCertSha256"] = req.PinSHA256
		}
		stream["tlsSettings"] = tls
	case "reality":
		stream["security"] = "reality"
		reality := map[string]interface{}{}
		if realityServerName != "" {
			reality["serverName"] = realityServerName
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

func parseSSUserInfo(userInfo string) (string, string, error) {
	plainCandidate := userInfo
	if strings.Contains(userInfo, ":") || strings.Contains(strings.ToLower(userInfo), "%3a") {
		unescaped, err := url.QueryUnescape(userInfo)
		if err == nil {
			plainCandidate = unescaped
		} else if unescaped, err = url.PathUnescape(userInfo); err == nil {
			plainCandidate = unescaped
		}
	} else {
		decoded, err := decodeBase64Text(userInfo)
		if err == nil {
			plainCandidate = decoded
		}
	}

	parts := strings.SplitN(plainCandidate, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid SS URI: expected method:password")
	}
	return parts[0], parts[1], nil
}

func decodeBase64Text(value string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(value)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(value)
				if err != nil {
					return "", err
				}
			}
		}
	}
	return string(decoded), nil
}

func applyV2RayPluginToSS(req *ShareLinkRequest, plugin string) error {
	req.Plugin = plugin

	parts := strings.Split(plugin, ";")
	if len(parts) == 0 {
		return nil
	}

	pluginName := strings.ToLower(strings.TrimSpace(parts[0]))
	if pluginName != "" && pluginName != "v2ray-plugin" {
		return fmt.Errorf("unsupported SS plugin: %s", parts[0])
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		lower := strings.ToLower(part)
		switch {
		case lower == "", lower == "tls":
			if lower == "tls" {
				req.TLS = "tls"
			}
		case strings.HasPrefix(lower, "obfs="):
			obfs := strings.TrimSpace(strings.TrimPrefix(lower, "obfs="))
			if obfs != "websocket" && obfs != "ws" {
				return fmt.Errorf("unsupported SS plugin obfs mode: %s", part)
			}
			req.Type = "ws"
		case strings.HasPrefix(lower, "obfs-host="):
			req.Host = part[len("obfs-host="):]
		case strings.HasPrefix(lower, "obfs-host"):
			req.Host = strings.TrimPrefix(part[len("obfs-host"):], "=")
		case strings.HasPrefix(lower, "host="):
			req.Host = part[len("host="):]
		case strings.HasPrefix(lower, "path="):
			req.Path = part[len("path="):]
		}
	}

	if req.Type == "ws" && req.Path == "" {
		req.Path = "/"
	}
	return nil
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

func normalizeTransportSecurityValue(securityValue, tlsValue string) string {
	switch strings.ToLower(strings.TrimSpace(securityValue)) {
	case "tls", "reality", "none", "":
		if securityValue != "" {
			return strings.ToLower(strings.TrimSpace(securityValue))
		}
	}

	switch strings.ToLower(strings.TrimSpace(tlsValue)) {
	case "tls", "true", "1":
		return "tls"
	case "reality":
		return "reality"
	case "none", "false", "0":
		return "none"
	}

	switch strings.ToLower(strings.TrimSpace(securityValue)) {
	case "tls", "reality", "none":
		return strings.ToLower(strings.TrimSpace(securityValue))
	default:
		return ""
	}
}

func parseBoolishParam(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func effectiveTLSServerName(req *ShareLinkRequest) string {
	if req == nil {
		return ""
	}
	if req.SNI != "" {
		return req.SNI
	}
	if req.Host != "" {
		return req.Host
	}
	if ip := stdnet.ParseIP(strings.Trim(req.Address, "[]")); ip == nil {
		return req.Address
	}
	return ""
}

func effectiveRealityServerName(req *ShareLinkRequest) string {
	if req == nil {
		return ""
	}
	if req.SNI != "" {
		return req.SNI
	}
	if req.Host != "" {
		return req.Host
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func hysteriaVersion(req *ShareLinkRequest) int {
	if req == nil || req.Version == 0 {
		return 2
	}
	return req.Version
}
