package webpanel

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ShareLinkRequest represents a share link generation request.
type ShareLinkRequest struct {
	Protocol  string                 `json:"protocol"`
	Address   string                 `json:"address"`
	Port      int                    `json:"port"`
	UUID      string                 `json:"uuid"`
	Password  string                 `json:"password"`
	Email     string                 `json:"email"`
	Security  string                 `json:"security"`
	Flow      string                 `json:"flow"`
	Type      string                 `json:"type"`      // Transport type: tcp, ws, grpc, etc.
	Host      string                 `json:"host"`
	Path      string                 `json:"path"`
	TLS       string                 `json:"tls"`       // tls, reality, none
	SNI       string                 `json:"sni"`
	ALPN      string                 `json:"alpn"`
	Fingerprint string              `json:"fingerprint"`
	PublicKey string                 `json:"publicKey"` // REALITY
	ShortID   string                 `json:"shortId"`   // REALITY
	SpiderX   string                 `json:"spiderX"`   // REALITY
	Remark    string                 `json:"remark"`
	Extra     map[string]string      `json:"extra"`
}

// GenerateShareLink generates a share link URI from the request.
func GenerateShareLink(req ShareLinkRequest) (string, error) {
	switch strings.ToLower(req.Protocol) {
	case "vless":
		return generateVLESSLink(req)
	case "vmess":
		return generateVMessLink(req)
	case "trojan":
		return generateTrojanLink(req)
	case "shadowsocks", "ss":
		return generateSSLink(req)
	default:
		return "", fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}
}

func generateVLESSLink(req ShareLinkRequest) (string, error) {
	if req.UUID == "" || req.Address == "" || req.Port == 0 {
		return "", fmt.Errorf("uuid, address, and port are required for VLESS")
	}

	params := url.Values{}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.Security != "" {
		params.Set("encryption", req.Security)
	} else {
		params.Set("encryption", "none")
	}
	if req.TLS != "" && req.TLS != "none" {
		params.Set("security", req.TLS)
	}
	if req.Flow != "" {
		params.Set("flow", req.Flow)
	}
	if req.SNI != "" {
		params.Set("sni", req.SNI)
	}
	if req.ALPN != "" {
		params.Set("alpn", req.ALPN)
	}
	if req.Fingerprint != "" {
		params.Set("fp", req.Fingerprint)
	}
	if req.Host != "" {
		params.Set("host", req.Host)
	}
	if req.Path != "" {
		params.Set("path", req.Path)
	}
	if req.PublicKey != "" {
		params.Set("pbk", req.PublicKey)
	}
	if req.ShortID != "" {
		params.Set("sid", req.ShortID)
	}
	if req.SpiderX != "" {
		params.Set("spx", req.SpiderX)
	}
	for k, v := range req.Extra {
		params.Set(k, v)
	}

	remark := req.Remark
	if remark == "" {
		remark = req.Email
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		req.UUID, req.Address, req.Port,
		params.Encode(),
		url.PathEscape(remark),
	), nil
}

func generateVMessLink(req ShareLinkRequest) (string, error) {
	if req.UUID == "" || req.Address == "" || req.Port == 0 {
		return "", fmt.Errorf("uuid, address, and port are required for VMess")
	}

	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   req.Remark,
		"add":  req.Address,
		"port": fmt.Sprintf("%d", req.Port),
		"id":   req.UUID,
		"aid":  "0",
		"scy":  "auto",
		"net":  req.Type,
		"type": "none",
		"host": req.Host,
		"path": req.Path,
		"tls":  req.TLS,
		"sni":  req.SNI,
		"alpn": req.ALPN,
		"fp":   req.Fingerprint,
	}

	if req.Security != "" {
		vmessConfig["scy"] = req.Security
	}
	if vmessConfig["ps"] == "" {
		vmessConfig["ps"] = req.Email
	}

	jsonBytes, err := json.Marshal(vmessConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMess config: %w", err)
	}

	return "vmess://" + base64.StdEncoding.EncodeToString(jsonBytes), nil
}

func generateTrojanLink(req ShareLinkRequest) (string, error) {
	password := req.Password
	if password == "" {
		password = req.UUID
	}
	if password == "" || req.Address == "" || req.Port == 0 {
		return "", fmt.Errorf("password/uuid, address, and port are required for Trojan")
	}

	params := url.Values{}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.TLS != "" && req.TLS != "none" {
		params.Set("security", req.TLS)
	}
	if req.SNI != "" {
		params.Set("sni", req.SNI)
	}
	if req.ALPN != "" {
		params.Set("alpn", req.ALPN)
	}
	if req.Fingerprint != "" {
		params.Set("fp", req.Fingerprint)
	}
	if req.Host != "" {
		params.Set("host", req.Host)
	}
	if req.Path != "" {
		params.Set("path", req.Path)
	}
	for k, v := range req.Extra {
		params.Set(k, v)
	}

	remark := req.Remark
	if remark == "" {
		remark = req.Email
	}

	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		url.PathEscape(password), req.Address, req.Port,
		params.Encode(),
		url.PathEscape(remark),
	), nil
}

func generateSSLink(req ShareLinkRequest) (string, error) {
	if req.Password == "" || req.Address == "" || req.Port == 0 {
		return "", fmt.Errorf("password, address, and port are required for Shadowsocks")
	}

	method := req.Security
	if method == "" {
		method = "aes-256-gcm"
	}

	userInfo := base64.URLEncoding.EncodeToString([]byte(method + ":" + req.Password))

	remark := req.Remark
	if remark == "" {
		remark = req.Email
	}

	return fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo, req.Address, req.Port,
		url.PathEscape(remark),
	), nil
}

// GenerateSubscriptionLinks generates base64-encoded subscription content from config.
func GenerateSubscriptionLinks(configData json.RawMessage) (string, error) {
	var config struct {
		Inbounds []struct {
			Protocol string          `json:"protocol"`
			Port     int             `json:"port"`
			Tag      string          `json:"tag"`
			Listen   string          `json:"listen"`
			Settings json.RawMessage `json:"settings"`
			StreamSettings struct {
				Network  string `json:"network"`
				Security string `json:"security"`
				WSSettings struct {
					Path    string            `json:"path"`
					Headers map[string]string `json:"headers"`
				} `json:"wsSettings"`
				TLSSettings struct {
					ServerName string   `json:"serverName"`
					ALPN       []string `json:"alpn"`
				} `json:"tlsSettings"`
			} `json:"streamSettings"`
		} `json:"inbounds"`
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	var links []string

	for _, ib := range config.Inbounds {
		protocol := strings.ToLower(ib.Protocol)
		if protocol != "vless" && protocol != "vmess" && protocol != "trojan" && protocol != "shadowsocks" {
			continue
		}

		// Parse settings to get users
		var settings struct {
			Clients []struct {
				ID       string `json:"id"`
				Email    string `json:"email"`
				Password string `json:"password"`
				Flow     string `json:"flow"`
				Security string `json:"security"`
			} `json:"clients"`
			Password string `json:"password"`
			Method   string `json:"method"`
		}
		if err := json.Unmarshal(ib.Settings, &settings); err != nil {
			continue
		}

		address := ib.Listen
		if address == "" || address == "0.0.0.0" || address == "::" {
			address = "YOUR_SERVER_IP"
		}

		for _, client := range settings.Clients {
			req := ShareLinkRequest{
				Protocol: protocol,
				Address:  address,
				Port:     ib.Port,
				UUID:     client.ID,
				Password: client.Password,
				Email:    client.Email,
				Flow:     client.Flow,
				Security: client.Security,
				Type:     ib.StreamSettings.Network,
				TLS:      ib.StreamSettings.Security,
				Host:     ib.StreamSettings.WSSettings.Headers["Host"],
				Path:     ib.StreamSettings.WSSettings.Path,
				SNI:      ib.StreamSettings.TLSSettings.ServerName,
				Remark:   client.Email,
			}
			if len(ib.StreamSettings.TLSSettings.ALPN) > 0 {
				req.ALPN = strings.Join(ib.StreamSettings.TLSSettings.ALPN, ",")
			}

			link, err := GenerateShareLink(req)
			if err == nil {
				links = append(links, link)
			}
		}
	}

	content := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}
