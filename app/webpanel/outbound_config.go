package webpanel

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	shadowaead2022 "github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	"github.com/xtls/xray-core/app/proxyman"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	core "github.com/xtls/xray-core/core"
	anytls "github.com/xtls/xray-core/proxy/anytls"
	blackhole "github.com/xtls/xray-core/proxy/blackhole"
	freedom "github.com/xtls/xray-core/proxy/freedom"
	hysteriaoutbound "github.com/xtls/xray-core/proxy/hysteria"
	shadowsocks "github.com/xtls/xray-core/proxy/shadowsocks"
	shadowsocks2022 "github.com/xtls/xray-core/proxy/shadowsocks_2022"
	trojan "github.com/xtls/xray-core/proxy/trojan"
	vless "github.com/xtls/xray-core/proxy/vless"
	vlessoutbound "github.com/xtls/xray-core/proxy/vless/outbound"
	vmess "github.com/xtls/xray-core/proxy/vmess"
	vmessoutbound "github.com/xtls/xray-core/proxy/vmess/outbound"
	"github.com/xtls/xray-core/transport/internet"
	salamandermask "github.com/xtls/xray-core/transport/internet/finalmask/salamander"
	grpctransport "github.com/xtls/xray-core/transport/internet/grpc"
	httpupgradetransport "github.com/xtls/xray-core/transport/internet/httpupgrade"
	hysteriatransport "github.com/xtls/xray-core/transport/internet/hysteria"
	realitytransport "github.com/xtls/xray-core/transport/internet/reality"
	splithttptransport "github.com/xtls/xray-core/transport/internet/splithttp"
	tlstransport "github.com/xtls/xray-core/transport/internet/tls"
	websockettransport "github.com/xtls/xray-core/transport/internet/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type standardOutboundConfig struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings"`
}

type standardStreamSettings struct {
	Network             string                       `json:"network"`
	Security            string                       `json:"security"`
	UDPMasks            []standardUDPMaskSettings    `json:"udpmasks"`
	WSSettings          *standardWebSocketSettings   `json:"wsSettings"`
	GRPCSettings        *standardGRPCSettings        `json:"grpcSettings"`
	HTTPUpgradeSettings *standardHTTPUpgradeSettings `json:"httpupgradeSettings"`
	SplitHTTPSettings   *standardSplitHTTPSettings   `json:"splithttpSettings"`
	HysteriaSettings    *standardHysteriaSettings    `json:"hysteriaSettings"`
	QuicParams          *standardQuicParams          `json:"quicParams"`
	TLSSettings         *standardTLSSettings         `json:"tlsSettings"`
	RealitySettings     *standardRealitySettings     `json:"realitySettings"`
}

type standardWebSocketSettings struct {
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

type standardGRPCSettings struct {
	Authority   string `json:"authority"`
	ServiceName string `json:"serviceName"`
}

type standardHTTPUpgradeSettings struct {
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

type standardSplitHTTPSettings struct {
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Mode    string            `json:"mode"`
	Headers map[string]string `json:"headers"`
}

type standardTLSSettings struct {
	ServerName           string   `json:"serverName"`
	ALPN                 []string `json:"alpn"`
	Fingerprint          string   `json:"fingerprint"`
	AllowInsecure        bool     `json:"allowInsecure"`
	PinnedPeerCertSha256 string   `json:"pinnedPeerCertSha256"`
}

type standardRealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	ShortID     string `json:"shortId"`
	SpiderX     string `json:"spiderX"`
}

type standardHysteriaSettings struct {
	Version int32  `json:"version"`
	Auth    string `json:"auth"`
}

type standardQuicParams struct {
	Congestion string `json:"congestion"`
	BrutalUp   uint64 `json:"brutalUp"`
	BrutalDown uint64 `json:"brutalDown"`
}

type standardUDPMaskSettings struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}

type standardVNextSettings struct {
	VNext []standardServerEndpoint `json:"vnext"`
}

type standardServerEndpoint struct {
	Address string         `json:"address"`
	Port    int            `json:"port"`
	Users   []standardUser `json:"users"`
}

type standardUser struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Level      uint32 `json:"level"`
	Encryption string `json:"encryption"`
	Flow       string `json:"flow"`
	Security   string `json:"security"`
}

type standardTrojanSettings struct {
	Servers []standardTrojanServer `json:"servers"`
}

type standardTrojanServer struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Level    uint32 `json:"level"`
}

type standardShadowsocksSettings struct {
	Servers []standardShadowsocksServer `json:"servers"`
}

type standardShadowsocksServer struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Level    uint32 `json:"level"`
}

type standardHysteriaProxySettings struct {
	Version int32                 `json:"version"`
	Address string                `json:"address"`
	Port    int                   `json:"port"`
	Server  standardServerAddress `json:"server"`
}

type standardAnyTLSSettings struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Password string `json:"password"`
}

type standardServerAddress struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type standardBlackholeSettings struct {
	Response *struct {
		Type string `json:"type"`
	} `json:"response"`
}

func decodeOutboundHandlerConfig(raw json.RawMessage) (*core.OutboundHandlerConfig, error) {
	if outboundConfig, ok, err := decodeEditableOutboundHandlerConfig(raw); ok || err != nil {
		return outboundConfig, err
	}

	var outboundConfig core.OutboundHandlerConfig
	if err := protojson.Unmarshal(raw, &outboundConfig); err == nil {
		return &outboundConfig, nil
	}

	return decodeStandardOutboundHandlerConfig(raw)
}

func buildOutboundHandlerConfigFromLink(link *ShareLinkRequest, tag string) (*core.OutboundHandlerConfig, error) {
	outboundJSON, err := BuildOutboundJSON(link, tag)
	if err != nil {
		return nil, err
	}

	return decodeStandardOutboundHandlerConfig(outboundJSON)
}

func decodeStandardOutboundHandlerConfig(raw json.RawMessage) (*core.OutboundHandlerConfig, error) {
	var outbound standardOutboundConfig
	if err := json.Unmarshal(raw, &outbound); err != nil {
		return nil, fmt.Errorf("invalid outbound config: %w", err)
	}
	if strings.TrimSpace(outbound.Tag) == "" {
		return nil, fmt.Errorf("outbound tag is required")
	}
	if strings.TrimSpace(outbound.Protocol) == "" {
		return nil, fmt.Errorf("outbound protocol is required")
	}

	senderSettings, err := decodeStandardSenderSettings(outbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	proxySettings, err := decodeStandardProxySettings(outbound.Protocol, outbound.Settings)
	if err != nil {
		return nil, err
	}

	return &core.OutboundHandlerConfig{
		Tag:            outbound.Tag,
		SenderSettings: serial.ToTypedMessage(senderSettings),
		ProxySettings:  serial.ToTypedMessage(proxySettings),
	}, nil
}

func decodeStandardSenderSettings(raw json.RawMessage) (*proxyman.SenderConfig, error) {
	senderSettings := &proxyman.SenderConfig{}
	if isEmptyRawJSON(raw) {
		return senderSettings, nil
	}

	var streamSettings standardStreamSettings
	if err := json.Unmarshal(raw, &streamSettings); err != nil {
		return nil, fmt.Errorf("invalid outbound stream settings: %w", err)
	}

	streamConfig, err := buildStandardStreamConfig(&streamSettings)
	if err != nil {
		return nil, err
	}
	if streamConfig != nil {
		senderSettings.StreamSettings = streamConfig
	}

	return senderSettings, nil
}

func decodeStandardProxySettings(protocolName string, raw json.RawMessage) (proto.Message, error) {
	switch strings.ToLower(strings.TrimSpace(protocolName)) {
	case "freedom":
		config := &freedom.Config{}
		if !isEmptyRawJSON(raw) {
			if err := json.Unmarshal(raw, config); err != nil {
				return nil, fmt.Errorf("invalid freedom settings: %w", err)
			}
		}
		return config, nil
	case "blackhole":
		var settings standardBlackholeSettings
		if !isEmptyRawJSON(raw) {
			if err := json.Unmarshal(raw, &settings); err != nil {
				return nil, fmt.Errorf("invalid blackhole settings: %w", err)
			}
		}
		response := serial.ToTypedMessage(&blackhole.NoneResponse{})
		if settings.Response != nil && strings.EqualFold(settings.Response.Type, "http") {
			response = serial.ToTypedMessage(&blackhole.HTTPResponse{})
		}
		return &blackhole.Config{Response: response}, nil
	case "vless":
		return buildStandardVLESSProxySettings(raw)
	case "vmess":
		return buildStandardVMessProxySettings(raw)
	case "trojan":
		return buildStandardTrojanProxySettings(raw)
	case "shadowsocks", "ss":
		return buildStandardShadowsocksProxySettings(raw)
	case "hysteria", "hysteria2":
		return buildStandardHysteriaProxySettings(raw)
	case "anytls":
		return buildStandardAnyTLSProxySettings(raw)
	default:
		return nil, fmt.Errorf("unsupported standard outbound protocol: %s", protocolName)
	}
}

func buildStandardVLESSProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardVNextSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("invalid vless settings: %w", err)
	}
	server, err := singleServerEndpoint(settings.VNext, "vless")
	if err != nil {
		return nil, err
	}
	user, err := singleUser(server.Users, "vless")
	if err != nil {
		return nil, err
	}

	return &vlessoutbound.Config{
		Vnext: buildServerEndpoint(server.Address, server.Port, &protocol.User{
			Level: user.Level,
			Email: user.Email,
			Account: serial.ToTypedMessage(&vless.Account{
				Id:         user.ID,
				Encryption: normalizedVLESSEncryption(user.Encryption),
				Flow:       user.Flow,
			}),
		}),
	}, nil
}

func buildStandardVMessProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardVNextSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("invalid vmess settings: %w", err)
	}
	server, err := singleServerEndpoint(settings.VNext, "vmess")
	if err != nil {
		return nil, err
	}
	user, err := singleUser(server.Users, "vmess")
	if err != nil {
		return nil, err
	}

	return &vmessoutbound.Config{
		Receiver: buildServerEndpoint(server.Address, server.Port, &protocol.User{
			Level: user.Level,
			Email: user.Email,
			Account: serial.ToTypedMessage(&vmess.Account{
				Id: user.ID,
				SecuritySettings: &protocol.SecurityConfig{
					Type: vmessSecurityTypeFromString(user.Security),
				},
			}),
		}),
	}, nil
}

func buildStandardTrojanProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardTrojanSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("invalid trojan settings: %w", err)
	}
	if len(settings.Servers) != 1 {
		return nil, fmt.Errorf("trojan settings must contain exactly one server")
	}
	server := settings.Servers[0]

	return &trojan.ClientConfig{
		Server: buildServerEndpoint(server.Address, server.Port, &protocol.User{
			Level: server.Level,
			Email: server.Email,
			Account: serial.ToTypedMessage(&trojan.Account{
				Password: server.Password,
			}),
		}),
	}, nil
}

func buildStandardShadowsocksProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardShadowsocksSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("invalid shadowsocks settings: %w", err)
	}
	if len(settings.Servers) != 1 {
		return nil, fmt.Errorf("shadowsocks settings must contain exactly one server")
	}
	server := settings.Servers[0]
	if isShadowsocks2022Method(server.Method) {
		return &shadowsocks2022.ClientConfig{
			Address: xnet.NewIPOrDomain(xnet.ParseAddress(server.Address)),
			Port:    uint32(server.Port),
			Method:  strings.TrimSpace(server.Method),
			Key:     server.Password,
		}, nil
	}
	cipherType, err := shadowsocksCipherTypeFromString(server.Method)
	if err != nil {
		return nil, err
	}

	return &shadowsocks.ClientConfig{
		Server: buildServerEndpoint(server.Address, server.Port, &protocol.User{
			Level: server.Level,
			Email: server.Email,
			Account: serial.ToTypedMessage(&shadowsocks.Account{
				Password:   server.Password,
				CipherType: cipherType,
			}),
		}),
	}, nil
}

func buildStandardHysteriaProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardHysteriaProxySettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("invalid hysteria settings: %w", err)
	}

	version := settings.Version
	if version == 0 {
		version = 2
	}

	address := strings.TrimSpace(settings.Address)
	port := settings.Port
	if address == "" {
		address = strings.TrimSpace(settings.Server.Address)
	}
	if port == 0 {
		port = settings.Server.Port
	}

	if address == "" {
		return nil, fmt.Errorf("hysteria server address is required")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("hysteria server port is invalid: %d", port)
	}

	return &hysteriaoutbound.ClientConfig{
		Version: version,
		Server:  buildServerEndpoint(address, port, nil),
	}, nil
}

func buildStandardAnyTLSProxySettings(raw json.RawMessage) (proto.Message, error) {
	var settings standardAnyTLSSettings
	if !isEmptyRawJSON(raw) {
		if err := json.Unmarshal(raw, &settings); err != nil {
			return nil, fmt.Errorf("invalid anytls settings: %w", err)
		}
	}

	address := strings.TrimSpace(settings.Address)
	if address == "" {
		return nil, fmt.Errorf("anytls server address is required")
	}

	port := settings.Port
	if port == 0 {
		port = 443
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("anytls server port is invalid: %d", port)
	}

	return &anytls.Config{
		Address:  address,
		Port:     uint32(port),
		Password: settings.Password,
	}, nil
}

func buildStandardStreamConfig(settings *standardStreamSettings) (*internet.StreamConfig, error) {
	if settings == nil {
		return nil, nil
	}

	protocolName, err := standardTransportProtocolName(settings.Network)
	if err != nil {
		return nil, err
	}

	streamConfig := &internet.StreamConfig{
		ProtocolName: protocolName,
	}
	if settings.QuicParams != nil {
		streamConfig.QuicParams = buildStandardQuicParams(settings.QuicParams)
	}

	switch protocolName {
	case "websocket":
		transportSettings := &websockettransport.Config{}
		if settings.WSSettings != nil {
			transportSettings.Path = settings.WSSettings.Path
			transportSettings.Host = settings.WSSettings.Host
			transportSettings.Header = cloneStringMap(settings.WSSettings.Headers)
			if transportSettings.Host == "" && transportSettings.Header != nil {
				transportSettings.Host = transportSettings.Header["Host"]
				delete(transportSettings.Header, "Host")
			}
		}
		streamConfig.TransportSettings = append(streamConfig.TransportSettings, &internet.TransportConfig{
			ProtocolName: protocolName,
			Settings:     serial.ToTypedMessage(transportSettings),
		})
	case "grpc":
		transportSettings := &grpctransport.Config{}
		if settings.GRPCSettings != nil {
			transportSettings.Authority = settings.GRPCSettings.Authority
			transportSettings.ServiceName = settings.GRPCSettings.ServiceName
		}
		streamConfig.TransportSettings = append(streamConfig.TransportSettings, &internet.TransportConfig{
			ProtocolName: protocolName,
			Settings:     serial.ToTypedMessage(transportSettings),
		})
	case "httpupgrade":
		transportSettings := &httpupgradetransport.Config{}
		if settings.HTTPUpgradeSettings != nil {
			transportSettings.Host = settings.HTTPUpgradeSettings.Host
			transportSettings.Path = settings.HTTPUpgradeSettings.Path
			transportSettings.Header = cloneStringMap(settings.HTTPUpgradeSettings.Headers)
		}
		streamConfig.TransportSettings = append(streamConfig.TransportSettings, &internet.TransportConfig{
			ProtocolName: protocolName,
			Settings:     serial.ToTypedMessage(transportSettings),
		})
	case "splithttp":
		transportSettings := &splithttptransport.Config{}
		if settings.SplitHTTPSettings != nil {
			transportSettings.Host = settings.SplitHTTPSettings.Host
			transportSettings.Path = settings.SplitHTTPSettings.Path
			transportSettings.Mode = settings.SplitHTTPSettings.Mode
			transportSettings.Headers = cloneStringMap(settings.SplitHTTPSettings.Headers)
		}
		streamConfig.TransportSettings = append(streamConfig.TransportSettings, &internet.TransportConfig{
			ProtocolName: protocolName,
			Settings:     serial.ToTypedMessage(transportSettings),
		})
	case "hysteria":
		transportSettings := &hysteriatransport.Config{}
		if settings.HysteriaSettings != nil {
			transportSettings.Version = settings.HysteriaSettings.Version
			transportSettings.Auth = settings.HysteriaSettings.Auth
		}
		if transportSettings.Version == 0 {
			transportSettings.Version = 2
		}
		streamConfig.TransportSettings = append(streamConfig.TransportSettings, &internet.TransportConfig{
			ProtocolName: protocolName,
			Settings:     serial.ToTypedMessage(transportSettings),
		})
	}

	for _, mask := range settings.UDPMasks {
		switch strings.ToLower(strings.TrimSpace(mask.Type)) {
		case "":
			continue
		case "salamander":
			streamConfig.Udpmasks = append(streamConfig.Udpmasks, serial.ToTypedMessage(&salamandermask.Config{
				Password: mask.Password,
			}))
		default:
			return nil, fmt.Errorf("unsupported outbound udp mask type: %s", mask.Type)
		}
	}

	switch normalizeTransportSecurityValue(settings.Security, "") {
	case "", "none":
		return streamConfig, nil
	case "tls":
		securitySettings := &tlstransport.Config{}
		if settings.TLSSettings != nil {
			securitySettings.ServerName = settings.TLSSettings.ServerName
			securitySettings.NextProtocol = append([]string(nil), settings.TLSSettings.ALPN...)
			securitySettings.Fingerprint = strings.ToLower(strings.TrimSpace(settings.TLSSettings.Fingerprint))
			securitySettings.AllowInsecure = settings.TLSSettings.AllowInsecure
			pins, err := decodePinnedPeerCertSHA256List(settings.TLSSettings.PinnedPeerCertSha256)
			if err != nil {
				return nil, err
			}
			securitySettings.PinnedPeerCertSha256 = pins
		}
		tm := serial.ToTypedMessage(securitySettings)
		streamConfig.SecurityType = tm.Type
		streamConfig.SecuritySettings = append(streamConfig.SecuritySettings, tm)
		return streamConfig, nil
	case "reality":
		securitySettings, err := buildStandardRealitySecuritySettings(settings.RealitySettings)
		if err != nil {
			return nil, err
		}
		tm := serial.ToTypedMessage(securitySettings)
		streamConfig.SecurityType = tm.Type
		streamConfig.SecuritySettings = append(streamConfig.SecuritySettings, tm)
		return streamConfig, nil
	default:
		return nil, fmt.Errorf("unsupported outbound stream security: %s", settings.Security)
	}
}

func decodePinnedPeerCertSHA256List(raw string) ([][]byte, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	hashes := make([][]byte, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.NewReplacer(":", "", "-", "", " ", "", "\t", "", "\n", "", "\r", "").Replace(strings.TrimSpace(part))
		if cleaned == "" {
			continue
		}
		hashValue, err := hex.DecodeString(cleaned)
		if err != nil {
			return nil, fmt.Errorf("invalid pinnedPeerCertSha256 value %q: %w", part, err)
		}
		hashes = append(hashes, hashValue)
	}
	return hashes, nil
}

func buildStandardRealitySecuritySettings(settings *standardRealitySettings) (*realitytransport.Config, error) {
	if settings == nil {
		return nil, fmt.Errorf("realitySettings are required when stream security is reality")
	}

	publicKey, err := base64.RawURLEncoding.DecodeString(settings.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid reality public key: %w", err)
	}

	shortID, err := decodeRealityShortID(settings.ShortID)
	if err != nil {
		return nil, err
	}

	spiderX := settings.SpiderX
	if spiderX == "" {
		spiderX = "/"
	}
	parsedSpiderX, spiderY, err := normalizeRealitySpiderX(spiderX)
	if err != nil {
		return nil, err
	}

	return &realitytransport.Config{
		Fingerprint: strings.ToLower(strings.TrimSpace(settings.Fingerprint)),
		ServerName:  settings.ServerName,
		PublicKey:   publicKey,
		ShortId:     shortID,
		SpiderX:     parsedSpiderX,
		SpiderY:     spiderY,
	}, nil
}

func normalizeRealitySpiderX(raw string) (string, []int64, error) {
	if raw == "" {
		raw = "/"
	}
	if !strings.HasPrefix(raw, "/") {
		return "", nil, fmt.Errorf("invalid reality spiderX: %q", raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", nil, fmt.Errorf("invalid reality spiderX: %w", err)
	}

	spiderY := make([]int64, 10)
	query := parsed.Query()
	parseRange := func(param string, index int) error {
		value := strings.TrimSpace(query.Get(param))
		if value == "" {
			return nil
		}

		parts := strings.Split(value, "-")
		switch len(parts) {
		case 1:
			parsedValue, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid reality spiderX %q value %q: %w", param, value, err)
			}
			spiderY[index] = parsedValue
			spiderY[index+1] = parsedValue
		case 2:
			start, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid reality spiderX %q value %q: %w", param, value, err)
			}
			end, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid reality spiderX %q value %q: %w", param, value, err)
			}
			spiderY[index] = start
			spiderY[index+1] = end
		default:
			return fmt.Errorf("invalid reality spiderX %q range %q", param, value)
		}

		query.Del(param)
		return nil
	}

	if err := parseRange("p", 0); err != nil {
		return "", nil, err
	}
	if err := parseRange("c", 2); err != nil {
		return "", nil, err
	}
	if err := parseRange("t", 4); err != nil {
		return "", nil, err
	}
	if err := parseRange("i", 6); err != nil {
		return "", nil, err
	}
	if err := parseRange("r", 8); err != nil {
		return "", nil, err
	}

	parsed.RawQuery = query.Encode()
	return parsed.String(), spiderY, nil
}

func buildServerEndpoint(address string, port int, user *protocol.User) *protocol.ServerEndpoint {
	return &protocol.ServerEndpoint{
		Address: xnet.NewIPOrDomain(xnet.ParseAddress(address)),
		Port:    uint32(port),
		User:    user,
	}
}

func singleServerEndpoint(servers []standardServerEndpoint, protocolName string) (standardServerEndpoint, error) {
	if len(servers) != 1 {
		return standardServerEndpoint{}, fmt.Errorf("%s settings must contain exactly one server", protocolName)
	}
	server := servers[0]
	if strings.TrimSpace(server.Address) == "" {
		return standardServerEndpoint{}, fmt.Errorf("%s server address is required", protocolName)
	}
	if server.Port <= 0 || server.Port > 65535 {
		return standardServerEndpoint{}, fmt.Errorf("%s server port is invalid: %d", protocolName, server.Port)
	}
	return server, nil
}

func singleUser(users []standardUser, protocolName string) (standardUser, error) {
	if len(users) != 1 {
		return standardUser{}, fmt.Errorf("%s settings must contain exactly one user", protocolName)
	}
	user := users[0]
	if strings.TrimSpace(user.ID) == "" {
		return standardUser{}, fmt.Errorf("%s user id is required", protocolName)
	}
	return user, nil
}

func standardTransportProtocolName(network string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "", "raw", "tcp":
		return "tcp", nil
	case "ws", "websocket":
		return "websocket", nil
	case "grpc":
		return "grpc", nil
	case "httpupgrade":
		return "httpupgrade", nil
	case "xhttp", "splithttp":
		return "splithttp", nil
	case "hysteria":
		return "hysteria", nil
	case "h2", "h3", "http":
		return "", fmt.Errorf("removed outbound stream network: %s", network)
	case "kcp", "mkcp", "quic":
		return "", fmt.Errorf("unsupported outbound stream network: %s", network)
	default:
		return "", fmt.Errorf("unknown outbound stream network: %s", network)
	}
}

func normalizedVLESSEncryption(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "none"
}

func vmessSecurityTypeFromString(value string) protocol.SecurityType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "aes-128-gcm":
		return protocol.SecurityType_AES128_GCM
	case "chacha20-poly1305":
		return protocol.SecurityType_CHACHA20_POLY1305
	case "none":
		return protocol.SecurityType_NONE
	case "zero":
		return protocol.SecurityType_ZERO
	case "auto", "":
		return protocol.SecurityType_AUTO
	default:
		return protocol.SecurityType_AUTO
	}
}

func shadowsocksCipherTypeFromString(value string) (shadowsocks.CipherType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "aes-128-gcm", "aead_aes_128_gcm":
		return shadowsocks.CipherType_AES_128_GCM, nil
	case "aes-256-gcm", "aead_aes_256_gcm", "":
		return shadowsocks.CipherType_AES_256_GCM, nil
	case "chacha20-poly1305", "aead_chacha20_poly1305", "chacha20-ietf-poly1305":
		return shadowsocks.CipherType_CHACHA20_POLY1305, nil
	case "xchacha20-poly1305", "aead_xchacha20_poly1305", "xchacha20-ietf-poly1305":
		return shadowsocks.CipherType_XCHACHA20_POLY1305, nil
	case "none", "plain":
		return shadowsocks.CipherType_NONE, nil
	default:
		return shadowsocks.CipherType_UNKNOWN, fmt.Errorf("unknown shadowsocks cipher method: %s", value)
	}
}

func buildStandardQuicParams(settings *standardQuicParams) *internet.QuicParams {
	if settings == nil {
		return nil
	}
	return &internet.QuicParams{
		Congestion: strings.ToLower(strings.TrimSpace(settings.Congestion)),
		BrutalUp:   settings.BrutalUp,
		BrutalDown: settings.BrutalDown,
	}
}

func isShadowsocks2022Method(value string) bool {
	method := strings.TrimSpace(value)
	for _, candidate := range shadowaead2022.List {
		if strings.EqualFold(method, candidate) {
			return true
		}
	}
	return false
}

func decodeRealityShortID(value string) ([]byte, error) {
	shortID := make([]byte, 8)
	value = strings.TrimSpace(value)
	if value == "" {
		return shortID, nil
	}
	if len(value) > 16 {
		return nil, fmt.Errorf("invalid reality short id: %s", value)
	}
	if _, err := hex.Decode(shortID, []byte(value)); err != nil {
		return nil, fmt.Errorf("invalid reality short id: %w", err)
	}
	return shortID, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func isEmptyRawJSON(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
