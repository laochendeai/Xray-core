package webpanel

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseShareLinkURIStripsUTF8BOM(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI("\ufeffvless://11111111-1111-1111-1111-111111111111@example.com:443?security=tls&type=ws&host=edge.example.com&sni=edge.example.com#bom")
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	if req.Protocol != "vless" {
		t.Fatalf("expected vless protocol, got %q", req.Protocol)
	}
	if req.Address != "example.com" {
		t.Fatalf("expected address example.com, got %q", req.Address)
	}
	if req.Remark != "bom" {
		t.Fatalf("expected remark bom, got %q", req.Remark)
	}
}

func TestParseSubscriptionContentStripsUTF8BOMOnFirstLine(t *testing.T) {
	t.Parallel()

	content := "\ufeffvmess://eyJ2IjoiMiIsInBzIjoiYm9tLXZtZXNzIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMTExMTExMS0xMTExLTExMTEtMTExMS0xMTExMTExMTExMTEiLCJhaWQiOiIwIiwic2N5IjoiYXV0byIsIm5ldCI6IndzIiwidGxzIjoidGxzIiwiaG9zdCI6ImVkZ2UuZXhhbXBsZS5jb20iLCJwYXRoIjoiLyJ9\nvless://22222222-2222-2222-2222-222222222222@example.org:8443?security=tls#second"

	links, err := ParseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("ParseSubscriptionContent returned error: %v", err)
	}

	if len(links) != 2 {
		t.Fatalf("expected 2 parsed links, got %d", len(links))
	}
	if links[0].Protocol != "vmess" {
		t.Fatalf("expected first link to be vmess, got %q", links[0].Protocol)
	}
	if links[0].Remark != "bom-vmess" {
		t.Fatalf("expected first remark bom-vmess, got %q", links[0].Remark)
	}
	if links[1].Protocol != "vless" {
		t.Fatalf("expected second link to be vless, got %q", links[1].Protocol)
	}
}

func TestParseSubscriptionContentIncludesAnyTLS(t *testing.T) {
	t.Parallel()

	payload := "anytls://secret@example.cc:8443/?sni=edge.example.cc#any"
	content := base64.StdEncoding.EncodeToString([]byte(payload))

	links, err := ParseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("ParseSubscriptionContent returned error: %v", err)
	}

	found := false
	for _, link := range links {
		if link.Protocol == "anytls" {
			found = true
			if link.Address != "example.cc" {
				t.Fatalf("expected anytls address example.cc, got %q", link.Address)
			}
			if link.Port != 8443 {
				t.Fatalf("expected anytls port 8443, got %d", link.Port)
			}
			if link.Password != "secret" {
				t.Fatalf("expected anytls password secret, got %q", link.Password)
			}
		}
	}
	if !found {
		t.Fatalf("expected anytls link to be parsed")
	}
}

func TestParseShareLinkURIParsesSSV2RayPlugin(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI(`ss://bm9uZTozZTk2ZWY3ZC04NjM3LTQyYzItYWM3My03NTE4MjE5NmU3NTU=@cf.zhetengsha.eu.org:443/?plugin=v2ray-plugin%3Bobfs%3Dwebsocket%3Bobfs-hostmfvpn.fuchen.indevs.in%3Btls#plugin-ss`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	if req.Protocol != "shadowsocks" {
		t.Fatalf("expected shadowsocks protocol, got %q", req.Protocol)
	}
	if req.Type != "ws" {
		t.Fatalf("expected ws transport, got %q", req.Type)
	}
	if req.TLS != "tls" {
		t.Fatalf("expected tls transport security, got %q", req.TLS)
	}
	if req.Host != "mfvpn.fuchen.indevs.in" {
		t.Fatalf("expected plugin host to be parsed, got %q", req.Host)
	}
	if req.Remark != "plugin-ss" {
		t.Fatalf("expected remark plugin-ss, got %q", req.Remark)
	}
}

func TestParseShareLinkURIParsesHysteria2(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI(`hysteria2://b9c27a40-beaa-49d8-bb39-0511b23c0966@yindu.iptk123.com:443?insecure=1&sni=yindu.iptk123.com&alpn=h3&congestion_control=bbr&pinSHA256=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef&obfs=salamander&obfs-password=maskme#hy2`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	if req.Protocol != "hysteria2" {
		t.Fatalf("expected hysteria2 protocol, got %q", req.Protocol)
	}
	if req.Type != "hysteria" {
		t.Fatalf("expected hysteria transport, got %q", req.Type)
	}
	if req.Password != "b9c27a40-beaa-49d8-bb39-0511b23c0966" {
		t.Fatalf("expected auth token to populate password field, got %q", req.Password)
	}
	if !req.AllowInsecure {
		t.Fatal("expected allow insecure to be true")
	}
	if req.Congestion != "bbr" {
		t.Fatalf("expected congestion bbr, got %q", req.Congestion)
	}
	if req.PinSHA256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("expected pinSHA256 to be parsed, got %q", req.PinSHA256)
	}
	if req.Obfs != "salamander" {
		t.Fatalf("expected salamander obfs, got %q", req.Obfs)
	}
	if req.ObfsPassword != "maskme" {
		t.Fatalf("expected obfs password maskme, got %q", req.ObfsPassword)
	}
}

func TestParseShareLinkURIParsesAnyTLS(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI(`anytls://hunter123@any.example.com:8443/?sni=edge.example.com&insecure=1#any`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	if req.Protocol != "anytls" {
		t.Fatalf("expected anytls protocol, got %q", req.Protocol)
	}
	if req.Address != "any.example.com" {
		t.Fatalf("expected address any.example.com, got %q", req.Address)
	}
	if req.Port != 8443 {
		t.Fatalf("expected port 8443, got %d", req.Port)
	}
	if req.Password != "hunter123" {
		t.Fatalf("expected password hunter123, got %q", req.Password)
	}
	if req.AllowInsecure != true {
		t.Fatal("expected allow insecure to be true")
	}
	if req.Type != "tcp" {
		t.Fatalf("expected transport type tcp, got %q", req.Type)
	}
	if req.TLS != "tls" {
		t.Fatalf("expected tls transport, got %q", req.TLS)
	}
	if req.SNI != "edge.example.com" {
		t.Fatalf("expected sni edge.example.com, got %q", req.SNI)
	}

	defaultReq, err := ParseShareLinkURI(`anytls://secret@example.com/?sni=edge`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}
	if defaultReq.Port != 443 {
		t.Fatalf("expected default port 443, got %d", defaultReq.Port)
	}
}

func TestBuildOutboundJSONUsesTopLevelHysteriaAddress(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI(`hysteria2://b9c27a40-beaa-49d8-bb39-0511b23c0966@yindu.iptk123.com:443?insecure=1&sni=yindu.iptk123.com#hy2`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	outboundJSON, err := BuildOutboundJSON(req, "pool-active-hy2")
	if err != nil {
		t.Fatalf("BuildOutboundJSON returned error: %v", err)
	}

	var outbound map[string]any
	if err := json.Unmarshal(outboundJSON, &outbound); err != nil {
		t.Fatalf("unmarshal outbound JSON: %v", err)
	}

	settings, ok := outbound["settings"].(map[string]any)
	if !ok {
		t.Fatalf("expected settings object, got %T", outbound["settings"])
	}
	if got := settings["address"]; got != "yindu.iptk123.com" {
		t.Fatalf("expected top-level hysteria address, got %#v", got)
	}
	if got := settings["port"]; got != float64(443) {
		t.Fatalf("expected top-level hysteria port 443, got %#v", got)
	}
	if _, ok := settings["server"]; ok {
		t.Fatal("did not expect nested hysteria server object")
	}
}

func TestBuildOutboundJSONSupportsAnyTLS(t *testing.T) {
	t.Parallel()

	req, err := ParseShareLinkURI(`anytls://secret@any.example.com:8443/?sni=edge.example.com#any`)
	if err != nil {
		t.Fatalf("ParseShareLinkURI returned error: %v", err)
	}

	outboundJSON, err := BuildOutboundJSON(req, "pool-anytls")
	if err != nil {
		t.Fatalf("BuildOutboundJSON returned error: %v", err)
	}

	var outbound map[string]any
	if err := json.Unmarshal(outboundJSON, &outbound); err != nil {
		t.Fatalf("unmarshal outbound JSON: %v", err)
	}

	if pvc, ok := outbound["protocol"]; !ok || pvc != "anytls" {
		t.Fatalf("expected protocol anytls, got %#v", outbound["protocol"])
	}
	settings, ok := outbound["settings"].(map[string]any)
	if !ok {
		t.Fatalf("expected settings object, got %T", outbound["settings"])
	}
	if got := settings["address"]; got != "any.example.com" {
		t.Fatalf("expected address any.example.com, got %#v", got)
	}
	if got := settings["port"]; got != float64(8443) {
		t.Fatalf("expected port 8443, got %#v", got)
	}
	if got := settings["password"]; got != "secret" {
		t.Fatalf("expected password secret, got %#v", got)
	}
}
