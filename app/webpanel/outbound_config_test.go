package webpanel

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/common/serial"
	anytls "github.com/xtls/xray-core/proxy/anytls"
	hysteriaproxy "github.com/xtls/xray-core/proxy/hysteria"
	shadowsocks2022 "github.com/xtls/xray-core/proxy/shadowsocks_2022"
	internettransport "github.com/xtls/xray-core/transport/internet"
	salamandermask "github.com/xtls/xray-core/transport/internet/finalmask/salamander"
	hysteriatransport "github.com/xtls/xray-core/transport/internet/hysteria"
	realitytransport "github.com/xtls/xray-core/transport/internet/reality"
	tlstransport "github.com/xtls/xray-core/transport/internet/tls"
)

func TestDecodeOutboundHandlerConfigSupportsStandardJSON(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
	  "tag": "direct-test",
	  "protocol": "freedom",
	  "settings": {}
	}`)

	config, err := decodeOutboundHandlerConfig(raw)
	if err != nil {
		t.Fatalf("decode outbound config: %v", err)
	}
	if config.Tag != "direct-test" {
		t.Fatalf("expected tag %q, got %q", "direct-test", config.Tag)
	}
	if config.SenderSettings == nil {
		t.Fatal("expected sender settings to be populated")
	}
	if config.ProxySettings == nil {
		t.Fatal("expected proxy settings to be populated")
	}
}

func TestDecodeStandardProxySettingsSupportsAnyTLS(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"address":"any.example.com","port":8443,"password":"secret"}`)
	config, err := decodeStandardProxySettings("anytls", raw)
	if err != nil {
		t.Fatalf("decode standard anytls settings: %v", err)
	}

	proxyConfig, ok := config.(*anytls.Config)
	if !ok {
		t.Fatalf("expected anytls config, got %T", config)
	}
	if proxyConfig.Address != "any.example.com" {
		t.Fatalf("expected address any.example.com, got %q", proxyConfig.Address)
	}
	if proxyConfig.Port != 8443 {
		t.Fatalf("expected port 8443, got %d", proxyConfig.Port)
	}
	if proxyConfig.Password != "secret" {
		t.Fatalf("expected password secret, got %q", proxyConfig.Password)
	}
}

func TestBuildOutboundHandlerConfigFromLinkBuildsTLSWSStream(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`vless://11111111-1111-1111-1111-111111111111@8.223.63.150:443?type=ws&path=/&host=dyj0.q78.eduin.indevs.in&tls=true#ws-tls`)
	if err != nil {
		t.Fatalf("parse share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-link-test")
	if err != nil {
		t.Fatalf("build outbound handler config from link: %v", err)
	}
	if config.Tag != "pool-link-test" {
		t.Fatalf("expected tag %q, got %q", "pool-link-test", config.Tag)
	}

	sender, err := typedMessageAs[*proxyman.SenderConfig](config.SenderSettings)
	if err != nil {
		t.Fatalf("decode sender settings: %v", err)
	}
	if sender.StreamSettings == nil {
		t.Fatal("expected stream settings to be populated")
	}
	if got := sender.StreamSettings.ProtocolName; got != "websocket" {
		t.Fatalf("expected websocket stream, got %q", got)
	}
	if len(sender.StreamSettings.SecuritySettings) != 1 {
		t.Fatalf("expected one TLS security setting, got %d", len(sender.StreamSettings.SecuritySettings))
	}

	tlsConfig, err := typedMessageAs[*tlstransport.Config](sender.StreamSettings.SecuritySettings[0])
	if err != nil {
		t.Fatalf("decode tls settings: %v", err)
	}
	if tlsConfig.ServerName != "dyj0.q78.eduin.indevs.in" {
		t.Fatalf("expected tls server name to backfill from host, got %q", tlsConfig.ServerName)
	}
}

func TestBuildOutboundHandlerConfigFromLinkBuildsRealityStream(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`vless://726dd73e-e06b-4665-a2c6-38a2277c499b@tw1.miyazono-kaori.com:443?type=tcp&security=reality&fp=chrome&pb=KJgODVMTQcncG_l1ZCDhzn8gEgMU2YRl2Yw2je6moWY&sid=c0d856ec#reality`)
	if err != nil {
		t.Fatalf("parse reality share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-reality-test")
	if err != nil {
		t.Fatalf("build outbound handler config from reality link: %v", err)
	}

	sender, err := typedMessageAs[*proxyman.SenderConfig](config.SenderSettings)
	if err != nil {
		t.Fatalf("decode sender settings: %v", err)
	}
	if sender.StreamSettings == nil {
		t.Fatal("expected stream settings to be populated")
	}
	if got := sender.StreamSettings.ProtocolName; got != "tcp" {
		t.Fatalf("expected tcp stream, got %q", got)
	}
	if len(sender.StreamSettings.SecuritySettings) != 1 {
		t.Fatalf("expected one REALITY security setting, got %d", len(sender.StreamSettings.SecuritySettings))
	}

	realityConfig, err := typedMessageAs[*realitytransport.Config](sender.StreamSettings.SecuritySettings[0])
	if err != nil {
		t.Fatalf("decode reality settings: %v", err)
	}
	if realityConfig.Fingerprint != "chrome" {
		t.Fatalf("expected fingerprint %q, got %q", "chrome", realityConfig.Fingerprint)
	}
	if len(realityConfig.PublicKey) != 32 {
		t.Fatalf("expected 32-byte public key, got %d bytes", len(realityConfig.PublicKey))
	}
	if got := len(realityConfig.ShortId); got != 8 {
		t.Fatalf("expected 8-byte short id buffer, got %d", got)
	}
	if got := len(realityConfig.SpiderY); got != 10 {
		t.Fatalf("expected 10-element spiderY buffer, got %d", got)
	}
	if realityConfig.SpiderX != "/" {
		t.Fatalf("expected default spiderX '/', got %q", realityConfig.SpiderX)
	}
	if realityConfig.ServerName != "" {
		t.Fatalf("expected empty reality server name when link omits sni/host, got %q", realityConfig.ServerName)
	}
}

func TestBuildOutboundHandlerConfigFromLinkParsesRealitySpiderXRanges(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`vless://726dd73e-e06b-4665-a2c6-38a2277c499b@tw1.miyazono-kaori.com:443?type=tcp&security=reality&fp=chrome&pb=KJgODVMTQcncG_l1ZCDhzn8gEgMU2YRl2Yw2je6moWY&sid=c0d856ec&spx=%2Fportal%3Fp%3D10-20%26c%3D2%26t%3D3-4%26i%3D5-6%26r%3D7-8#reality-spider`)
	if err != nil {
		t.Fatalf("parse reality share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-reality-spider-test")
	if err != nil {
		t.Fatalf("build outbound handler config from reality link: %v", err)
	}

	sender, err := typedMessageAs[*proxyman.SenderConfig](config.SenderSettings)
	if err != nil {
		t.Fatalf("decode sender settings: %v", err)
	}

	realityConfig, err := typedMessageAs[*realitytransport.Config](sender.StreamSettings.SecuritySettings[0])
	if err != nil {
		t.Fatalf("decode reality settings: %v", err)
	}

	if realityConfig.SpiderX != "/portal" {
		t.Fatalf("expected spiderX '/portal', got %q", realityConfig.SpiderX)
	}

	expectedSpiderY := []int64{10, 20, 2, 2, 3, 4, 5, 6, 7, 8}
	if !reflect.DeepEqual(realityConfig.SpiderY, expectedSpiderY) {
		t.Fatalf("unexpected spiderY: got %v want %v", realityConfig.SpiderY, expectedSpiderY)
	}
}

func TestBuildOutboundHandlerConfigFromLinkBuildsHysteria2Stream(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`hysteria2://b9c27a40-beaa-49d8-bb39-0511b23c0966@yindu.iptk123.com:443?insecure=1&sni=yindu.iptk123.com&alpn=h3&congestion_control=bbr&pinSHA256=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef&obfs=salamander&obfs-password=maskme#hy2`)
	if err != nil {
		t.Fatalf("parse hysteria2 share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-hy2-test")
	if err != nil {
		t.Fatalf("build outbound handler config from hysteria2 link: %v", err)
	}

	sender, err := typedMessageAs[*proxyman.SenderConfig](config.SenderSettings)
	if err != nil {
		t.Fatalf("decode sender settings: %v", err)
	}
	if sender.StreamSettings == nil {
		t.Fatal("expected stream settings to be populated")
	}
	if got := sender.StreamSettings.ProtocolName; got != "hysteria" {
		t.Fatalf("expected hysteria stream, got %q", got)
	}
	if sender.StreamSettings.QuicParams == nil || sender.StreamSettings.QuicParams.Congestion != "bbr" {
		t.Fatalf("expected quic congestion bbr, got %+v", sender.StreamSettings.QuicParams)
	}
	if len(sender.StreamSettings.TransportSettings) != 1 {
		t.Fatalf("expected one transport setting, got %d", len(sender.StreamSettings.TransportSettings))
	}

	hysteriaTransportConfig, err := typedMessageAs[*hysteriatransport.Config](sender.StreamSettings.TransportSettings[0].Settings)
	if err != nil {
		t.Fatalf("decode hysteria transport settings: %v", err)
	}
	if hysteriaTransportConfig.Version != 2 {
		t.Fatalf("expected hysteria transport version 2, got %d", hysteriaTransportConfig.Version)
	}
	if hysteriaTransportConfig.Auth != "b9c27a40-beaa-49d8-bb39-0511b23c0966" {
		t.Fatalf("expected hysteria auth to be preserved, got %q", hysteriaTransportConfig.Auth)
	}

	tlsConfig, err := typedMessageAs[*tlstransport.Config](sender.StreamSettings.SecuritySettings[0])
	if err != nil {
		t.Fatalf("decode tls settings: %v", err)
	}
	if !tlsConfig.AllowInsecure {
		t.Fatal("expected allow insecure to be propagated")
	}
	if tlsConfig.ServerName != "yindu.iptk123.com" {
		t.Fatalf("expected hysteria tls server name to use sni, got %q", tlsConfig.ServerName)
	}
	if len(tlsConfig.PinnedPeerCertSha256) != 1 {
		t.Fatalf("expected one pinned cert hash, got %d", len(tlsConfig.PinnedPeerCertSha256))
	}
	if got := hex.EncodeToString(tlsConfig.PinnedPeerCertSha256[0]); got != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("expected pinned cert hash to be preserved, got %q", got)
	}
	if len(sender.StreamSettings.Udpmasks) != 1 {
		t.Fatalf("expected one udp mask, got %d", len(sender.StreamSettings.Udpmasks))
	}
	maskConfig, err := typedMessageAs[*salamandermask.Config](sender.StreamSettings.Udpmasks[0])
	if err != nil {
		t.Fatalf("decode salamander udp mask: %v", err)
	}
	if maskConfig.Password != "maskme" {
		t.Fatalf("expected salamander password maskme, got %q", maskConfig.Password)
	}

	proxyConfig, err := typedMessageAs[*hysteriaproxy.ClientConfig](config.ProxySettings)
	if err != nil {
		t.Fatalf("decode hysteria proxy settings: %v", err)
	}
	if proxyConfig.Version != 2 {
		t.Fatalf("expected hysteria proxy version 2, got %d", proxyConfig.Version)
	}
}

func TestBuildOutboundHandlerConfigFromLinkBuildsShadowsocks2022Proxy(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`ss://2022-blake3-aes-256-gcm:VJE3om53Iz5t3ugNa57KYNW79ZBrgxCtMXfMw%2BRKrEc%3D%3AAAvo%2BBsgpvvR5lJsdQU32vrMXL7ZAbZJZfETB7bddhQ%3D@toy4lkdzy0c.22b74943-12ad-47f4-b705-f2defb6ffea0.org:13952#ss2022`)
	if err != nil {
		t.Fatalf("parse ss2022 share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-ss2022-test")
	if err != nil {
		t.Fatalf("build outbound handler config from ss2022 link: %v", err)
	}

	proxyConfig, err := typedMessageAs[*shadowsocks2022.ClientConfig](config.ProxySettings)
	if err != nil {
		t.Fatalf("decode ss2022 proxy settings: %v", err)
	}
	if proxyConfig.Method != "2022-blake3-aes-256-gcm" {
		t.Fatalf("expected ss2022 method to be preserved, got %q", proxyConfig.Method)
	}
	if proxyConfig.Port != 13952 {
		t.Fatalf("expected ss2022 port 13952, got %d", proxyConfig.Port)
	}
}

func TestBuildOutboundHandlerConfigFromLinkBuildsShadowsocksPluginWebSocketStream(t *testing.T) {
	t.Parallel()

	link, err := ParseShareLinkURI(`ss://bm9uZTozZTk2ZWY3ZC04NjM3LTQyYzItYWM3My03NTE4MjE5NmU3NTU=@cf.zhetengsha.eu.org:443/?plugin=v2ray-plugin%3Bobfs%3Dwebsocket%3Bobfs-hostmfvpn.fuchen.indevs.in%3Btls#plugin-ss`)
	if err != nil {
		t.Fatalf("parse plugin ss share link: %v", err)
	}

	config, err := buildOutboundHandlerConfigFromLink(link, "pool-ss-plugin-test")
	if err != nil {
		t.Fatalf("build outbound handler config from plugin ss link: %v", err)
	}

	sender, err := typedMessageAs[*proxyman.SenderConfig](config.SenderSettings)
	if err != nil {
		t.Fatalf("decode sender settings: %v", err)
	}
	if sender.StreamSettings == nil {
		t.Fatal("expected stream settings to be populated")
	}
	if got := sender.StreamSettings.ProtocolName; got != "websocket" {
		t.Fatalf("expected websocket stream, got %q", got)
	}

	tlsConfig, err := typedMessageAs[*tlstransport.Config](sender.StreamSettings.SecuritySettings[0])
	if err != nil {
		t.Fatalf("decode tls settings: %v", err)
	}
	if tlsConfig.ServerName != "mfvpn.fuchen.indevs.in" {
		t.Fatalf("expected plugin host to backfill tls server name, got %q", tlsConfig.ServerName)
	}

	transportConfig := sender.StreamSettings.TransportSettings[0]
	if transportConfig.ProtocolName != "websocket" {
		t.Fatalf("expected websocket transport config, got %q", transportConfig.ProtocolName)
	}
	if _, err := typedMessageAs[*internettransport.TransportConfig](nil); err != nil {
		t.Fatalf("typed message helper sanity check failed: %v", err)
	}
}

func typedMessageAs[T any](tm *serial.TypedMessage) (T, error) {
	var zero T
	if tm == nil {
		return zero, nil
	}
	instance, err := tm.GetInstance()
	if err != nil {
		return zero, err
	}

	value, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected typed message instance %T", instance)
	}
	return value, nil
}
