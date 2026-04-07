package webpanel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mustGenerateTunTestURI(t *testing.T, address string) string {
	t.Helper()

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  address,
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	return uri
}

func TestBuildTunRuntimeConfigInjectsActivePool(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	uri := mustGenerateTunTestURI(t, "203.0.113.11")

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		RemoteDNS:      []string{"1.1.1.1"},
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{
		{
			ID:  "node-1",
			URI: uri,
		},
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	rendered := string(output)
	for _, token := range []string{
		`"tag": "tun-in"`,
		`"balancerTag": "node-pool-active"`,
		`"tag": "node-pool-active"`,
		`"selector": [`,
		`"pool-active-node-1"`,
	} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("expected runtime config to contain %q\n%s", token, rendered)
		}
	}
}

func TestBuildTunRuntimeConfigUsesLowestLatencyPolicy(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "observatory": {
    "subjectSelector": ["proxy-"],
    "probeURL": "https://latency.example.test/generate_204",
    "probeInterval": "20s"
  },
  "routing": {
    "rules": []
  }
}`)

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:   "xray0",
		MTU:             1500,
		SelectionPolicy: string(TunSelectionPolicyLowestLatency),
		ProtectCIDRs:    []string{"127.0.0.0/8"},
		ProtectDomains:  []string{"full:localhost"},
	}, []NodeRecord{
		{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.31"), AvgDelayMs: 88, TotalPings: 10},
		{ID: "node-2", URI: mustGenerateTunTestURI(t, "203.0.113.32"), AvgDelayMs: 21, TotalPings: 10},
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing := rendered["routing"].(map[string]any)
	balancers := routing["balancers"].([]any)
	balancer := balancers[0].(map[string]any)
	if balancer["fallbackTag"] != "pool-active-node-2" {
		t.Fatalf("expected fallback tag to prefer the lowest latency node, got %v", balancer["fallbackTag"])
	}

	strategy := balancer["strategy"].(map[string]any)
	if strategy["type"] != "leastping" {
		t.Fatalf("expected leastping strategy, got %v", strategy["type"])
	}

	burstObservatory := rendered["burstObservatory"].(map[string]any)
	pingConfig := burstObservatory["pingConfig"].(map[string]any)
	if pingConfig["destination"] != "https://latency.example.test/generate_204" {
		t.Fatalf("expected burst observatory destination to inherit probe URL, got %v", pingConfig["destination"])
	}
	if pingConfig["interval"] != "20s" {
		t.Fatalf("expected burst observatory interval to inherit probe interval, got %v", pingConfig["interval"])
	}
}

func TestBuildTunRuntimeConfigUsesLowestFailRateSubset(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:   "xray0",
		MTU:             1500,
		SelectionPolicy: string(TunSelectionPolicyLowestFailRate),
		ProtectCIDRs:    []string{"127.0.0.0/8"},
		ProtectDomains:  []string{"full:localhost"},
	}, []NodeRecord{
		{ID: "node-a", URI: mustGenerateTunTestURI(t, "203.0.113.41"), AvgDelayMs: 80, TotalPings: 10, FailedPings: 0},
		{ID: "node-b", URI: mustGenerateTunTestURI(t, "203.0.113.42"), AvgDelayMs: 25, TotalPings: 10, FailedPings: 1},
		{ID: "node-c", URI: mustGenerateTunTestURI(t, "203.0.113.43"), AvgDelayMs: 60, TotalPings: 6, FailedPings: 0},
		{ID: "node-d", URI: mustGenerateTunTestURI(t, "203.0.113.44"), AvgDelayMs: 10, TotalPings: 10, FailedPings: 5},
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing := rendered["routing"].(map[string]any)
	balancers := routing["balancers"].([]any)
	balancer := balancers[0].(map[string]any)
	if balancer["fallbackTag"] != "pool-active-node-c" {
		t.Fatalf("expected fallback tag to prefer the lowest fail-rate node, got %v", balancer["fallbackTag"])
	}

	selectors := balancer["selector"].([]any)
	selectorSet := make(map[string]struct{}, len(selectors))
	for _, rawSelector := range selectors {
		selectorSet[rawSelector.(string)] = struct{}{}
	}

	for _, expected := range []string{
		"pool-active-node-a",
		"pool-active-node-b",
		"pool-active-node-c",
	} {
		if _, ok := selectorSet[expected]; !ok {
			t.Fatalf("expected selector set to include %s, got %v", expected, selectorSet)
		}
	}
	if _, ok := selectorSet["pool-active-node-d"]; ok {
		t.Fatalf("expected highest fail-rate node to be excluded, got %v", selectorSet)
	}

	strategy := balancer["strategy"].(map[string]any)
	if strategy["type"] != "leastping" {
		t.Fatalf("expected leastping strategy inside the fail-rate subset, got %v", strategy["type"])
	}
}

func TestBuildTunRuntimeConfigRequiresActiveNodes(t *testing.T) {
	t.Parallel()

	_, err := buildTunRuntimeConfig([]byte(`{"outbounds":[{"tag":"direct","protocol":"freedom"}]}`), &TunFeatureSettings{
		InterfaceName: "xray0",
	}, nil)
	if err == nil {
		t.Fatal("expected error when active pool is empty")
	}
}

func TestBuildTunRuntimeConfigPlacesTunCatchAllAfterSpecificProxyRules(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" },
    { "tag": "proxy-01", "protocol": "freedom" }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "domain": ["domain:openai.com"],
        "outboundTag": "proxy-01"
      },
      {
        "type": "field",
        "network": "tcp,udp",
        "balancerTag": "auto"
      }
    ]
  }
}`)

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "203.0.113.12",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{
		{
			ID:  "node-1",
			URI: uri,
		},
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	openAIProxyIndex := -1
	tunCatchAllIndex := -1
	autoFallbackIndex := -1
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}

		if domains, ok := rule["domain"].([]any); ok && len(domains) == 1 && domains[0] == "domain:openai.com" && rule["outboundTag"] == "proxy-01" {
			openAIProxyIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["balancerTag"] == "node-pool-active" {
			tunCatchAllIndex = index
		}
		if rule["balancerTag"] == "auto" {
			autoFallbackIndex = index
		}
	}

	if openAIProxyIndex == -1 {
		t.Fatal("expected existing specific proxy rule to be preserved")
	}
	if tunCatchAllIndex == -1 {
		t.Fatal("expected tun catch-all rule to be injected")
	}
	if openAIProxyIndex >= tunCatchAllIndex {
		t.Fatalf("expected specific proxy rule before tun catch-all, got proxy=%d tun=%d", openAIProxyIndex, tunCatchAllIndex)
	}
	if autoFallbackIndex != -1 {
		t.Fatalf("expected generic auto fallback rule to be removed from tun runtime, got index=%d", autoFallbackIndex)
	}
}

func TestBuildTunRuntimeConfigDropsWideBaseDirectRulesButKeepsProtectedLocalEntries(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "domain": [
          "domain:qq.com",
          "full:leo-cy-ub.tailf0aed5.ts.net",
          "domain:tailf0aed5.ts.net",
          "full:localhost"
        ],
        "outboundTag": "direct"
      },
      {
        "type": "field",
        "ip": [
          "47.237.11.85",
          "127.0.0.0/8"
        ],
        "port": "3000",
        "outboundTag": "direct"
      }
    ]
  }
}`)

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "203.0.113.21",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{{ID: "node-1", URI: uri}})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	hasQQDirect := false
	hasLocalhostDirect := false
	hasTailscaleDirect := false
	hasRemoteIPDirect := false
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}

		if domains, ok := rule["domain"].([]any); ok {
			for _, domain := range domains {
				switch domain {
				case "domain:qq.com":
					hasQQDirect = true
				case "full:localhost":
					hasLocalhostDirect = true
				case "domain:tailf0aed5.ts.net":
					hasTailscaleDirect = true
				}
			}
		}
		if ips, ok := rule["ip"].([]any); ok {
			for _, ip := range ips {
				if ip == "47.237.11.85" {
					hasRemoteIPDirect = true
				}
			}
		}
	}

	if hasQQDirect {
		t.Fatal("expected wide direct domain rules to be stripped from tun runtime")
	}
	if hasRemoteIPDirect {
		t.Fatal("expected non-protected direct IP rules to be stripped from tun runtime")
	}
	if !hasLocalhostDirect {
		t.Fatal("expected localhost direct protection to remain in tun runtime")
	}
	if !hasTailscaleDirect {
		t.Fatal("expected local tailscale domain protection to remain in tun runtime")
	}
}

func TestBuildTunRuntimeConfigAutoTestedModeKeepsSuccessfulDirectCandidates(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "domain": [
          "domain:ifconfig.me",
          "domain:blocked.example",
          "full:localhost"
        ],
        "outboundTag": "direct"
      },
      {
        "type": "field",
        "ip": [
          "47.237.11.85",
          "203.0.113.0/24",
          "127.0.0.0/8"
        ],
        "port": "3000",
        "outboundTag": "direct"
      }
    ]
  }
}`)

	uri := mustGenerateTunTestURI(t, "203.0.113.55")
	output, err := buildTunRuntimeConfigWithDirectProbeResults(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		RouteMode:      string(TunRouteModeAutoTested),
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{{ID: "node-1", URI: uri}}, map[string]bool{
		tunDirectProbeKey("domain", "ifconfig.me", []int{443, 80}):     true,
		tunDirectProbeKey("domain", "blocked.example", []int{443, 80}): false,
		tunDirectProbeKey("ip", "47.237.11.85", []int{3000}):           true,
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	hasIfconfigDirect := false
	hasBlockedDirect := false
	hasLocalhostDirect := false
	hasTestedIPDirect := false
	hasCIDRDirect := false
	hasLocalCIDRDirect := false
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}

		if domains, ok := rule["domain"].([]any); ok {
			for _, domain := range domains {
				switch domain {
				case "domain:ifconfig.me":
					hasIfconfigDirect = true
				case "domain:blocked.example":
					hasBlockedDirect = true
				case "full:localhost":
					hasLocalhostDirect = true
				}
			}
		}

		if ips, ok := rule["ip"].([]any); ok {
			for _, ip := range ips {
				switch ip {
				case "47.237.11.85":
					hasTestedIPDirect = true
				case "203.0.113.0/24":
					hasCIDRDirect = true
				case "127.0.0.0/8":
					hasLocalCIDRDirect = true
				}
			}
		}
	}

	if !hasIfconfigDirect {
		t.Fatal("expected successful direct probe domain to remain direct")
	}
	if hasBlockedDirect {
		t.Fatal("expected failed direct probe domain to fall back to proxy")
	}
	if !hasLocalhostDirect {
		t.Fatal("expected protected localhost rule to remain direct")
	}
	if !hasTestedIPDirect {
		t.Fatal("expected successful tested IP rule to remain direct")
	}
	if hasCIDRDirect {
		t.Fatal("expected unsupported non-host CIDR to fall back to proxy")
	}
	if !hasLocalCIDRDirect {
		t.Fatal("expected protected local CIDR to remain direct")
	}
}

func TestTunManagerUpdateEditableSettingsNormalizesWildcardProtectDomains(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {}
  }
}
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	settings, err := tunManager.UpdateEditableSettings(TunEditableSettings{
		SelectionPolicy: string(TunSelectionPolicyFastest),
		RouteMode:       string(TunRouteModeStrictProxy),
		ProtectDomains:  []string{"*.example.com", "domain:example.net", ".internal.example", "*.example.com"},
	})
	if err != nil {
		t.Fatalf("update editable settings: %v", err)
	}

	expected := []string{"domain:example.com", "domain:example.net", "domain:internal.example"}
	if !reflect.DeepEqual(settings.ProtectDomains, expected) {
		t.Fatalf("expected normalized protect domains %v, got %v", expected, settings.ProtectDomains)
	}
}

func TestTunManagerEditableSettingsNormalizesWildcardProtectDomainsFromConfig(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {
      "protectDomains": ["*.example.com", ".internal.example", "full:exact.example"]
    }
  }
}
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	settings, err := tunManager.EditableSettings()
	if err != nil {
		t.Fatalf("editable settings: %v", err)
	}

	expected := []string{"domain:example.com", "domain:internal.example", "full:exact.example"}
	if !reflect.DeepEqual(settings.ProtectDomains, expected) {
		t.Fatalf("expected normalized protect domains %v, got %v", expected, settings.ProtectDomains)
	}
}

func TestTunManagerEditableSettingsPersistRemoteDNS(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {}
  }
}
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	settings, err := tunManager.UpdateEditableSettings(TunEditableSettings{
		SelectionPolicy: string(TunSelectionPolicyFastest),
		RouteMode:       string(TunRouteModeStrictProxy),
		RemoteDNS:       []string{"1.1.1.1", "8.8.8.8", "1.1.1.1"},
	})
	if err != nil {
		t.Fatalf("update editable settings: %v", err)
	}

	if !reflect.DeepEqual(settings.RemoteDNS, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Fatalf("unexpected remote dns list: %v", settings.RemoteDNS)
	}

	loaded, err := tunManager.EditableSettings()
	if err != nil {
		t.Fatalf("reload editable settings: %v", err)
	}
	if !reflect.DeepEqual(loaded.RemoteDNS, settings.RemoteDNS) {
		t.Fatalf("expected remote dns to persist, got %v", loaded.RemoteDNS)
	}
}

func TestBuildTunRuntimeConfigEnablesSniffingAndCnDirectRulesWhenAssetsExist(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	for _, assetName := range []string{"geoip.dat", "geosite.dat"} {
		if err := os.WriteFile(filepath.Join(binDir, assetName), []byte("test"), 0o644); err != nil {
			t.Fatalf("write asset %s: %v", assetName, err)
		}
	}

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "203.0.113.13",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		BinaryPath:     filepath.Join(binDir, "xray-webpanel-xray"),
		InterfaceName:  "xray0",
		MTU:            1500,
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{{ID: "node-1", URI: uri}})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	inbounds, ok := rendered["inbounds"].([]any)
	if !ok || len(inbounds) != 1 {
		t.Fatalf("expected one tun inbound, got %#v", rendered["inbounds"])
	}
	inbound, ok := inbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("expected tun inbound object, got %#v", inbounds[0])
	}
	sniffing, ok := inbound["sniffing"].(map[string]any)
	if !ok {
		t.Fatalf("expected sniffing to be enabled on tun inbound, got %#v", inbound["sniffing"])
	}
	if sniffing["enabled"] != true {
		t.Fatalf("unexpected sniffing config: %#v", sniffing)
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	hasGeositeCN := false
	hasGeoipCN := false
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}
		if domains, ok := rule["domain"].([]any); ok {
			for _, domain := range domains {
				if domain == "geosite:cn" && rule["outboundTag"] == "direct" {
					hasGeositeCN = true
				}
			}
		}
		if ips, ok := rule["ip"].([]any); ok {
			for _, ip := range ips {
				if ip == "geoip:cn" && rule["outboundTag"] == "direct" {
					hasGeoipCN = true
				}
			}
		}
	}

	if !hasGeositeCN {
		t.Fatal("expected geosite:cn direct rule when geosite.dat is available")
	}
	if !hasGeoipCN {
		t.Fatal("expected geoip:cn direct rule when geoip.dat is available")
	}
}

func TestBuildTunRuntimeConfigAddsSplitDNSAndRoutesDNSBeforeCatchAll(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	for _, assetName := range []string{"geoip.dat", "geosite.dat"} {
		if err := os.WriteFile(filepath.Join(binDir, assetName), []byte("test"), 0o644); err != nil {
			t.Fatalf("write asset %s: %v", assetName, err)
		}
	}

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "203.0.113.14",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		BinaryPath:     filepath.Join(binDir, "xray-webpanel-xray"),
		InterfaceName:  "xray0",
		MTU:            1500,
		RemoteDNS:      []string{"1.1.1.1", "8.8.8.8"},
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{{ID: "node-1", URI: uri}})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	outbounds, ok := rendered["outbounds"].([]any)
	if !ok {
		t.Fatalf("expected outbounds, got %#v", rendered["outbounds"])
	}
	hasDNSOutbound := false
	for _, rawOutbound := range outbounds {
		outbound, ok := rawOutbound.(map[string]any)
		if !ok {
			continue
		}
		if outbound["tag"] == "dns-out" && outbound["protocol"] == "dns" {
			hasDNSOutbound = true
			settings, _ := outbound["settings"].(map[string]any)
			if settings["nonIPQuery"] != "skip" {
				t.Fatalf("unexpected dns outbound settings: %#v", settings)
			}
		}
	}
	if !hasDNSOutbound {
		t.Fatal("expected dns-out outbound in runtime config")
	}

	dnsConfig, ok := rendered["dns"].(map[string]any)
	if !ok {
		t.Fatalf("expected dns config, got %#v", rendered["dns"])
	}
	if dnsConfig["queryStrategy"] != "UseIP" {
		t.Fatalf("unexpected dns query strategy: %#v", dnsConfig["queryStrategy"])
	}
	if dnsConfig["disableFallbackIfMatch"] != true {
		t.Fatalf("expected disableFallbackIfMatch=true, got %#v", dnsConfig["disableFallbackIfMatch"])
	}

	servers, ok := dnsConfig["servers"].([]any)
	if !ok {
		t.Fatalf("expected dns servers, got %#v", dnsConfig["servers"])
	}
	hasChinaDNS := false
	hasRemoteDNS := false
	hasProtectedLocalDNS := false
	for _, rawServer := range servers {
		server, ok := rawServer.(map[string]any)
		if !ok {
			continue
		}
		switch server["tag"] {
		case "dns-direct-local":
			if server["address"] != "localhost" {
				t.Fatalf("expected protected direct-domain DNS entry to use local system DNS, got %#v", server)
			}
			if domains, ok := server["domains"].([]any); ok {
				for _, rawDomain := range domains {
					if rawDomain == "full:localhost" {
						hasProtectedLocalDNS = true
					}
				}
			}
		case "dns-cn":
			if server["address"] == "tcp://223.5.5.5" {
				hasChinaDNS = true
			}
		case "dns-remote":
			if server["address"] == "tcp://1.1.1.1" {
				hasRemoteDNS = true
			}
		}
	}
	if !hasChinaDNS {
		t.Fatal("expected china split-dns server in runtime config")
	}
	if !hasProtectedLocalDNS {
		t.Fatal("expected protected direct-domain local-system DNS server in runtime config")
	}
	if !hasRemoteDNS {
		t.Fatal("expected remote split-dns server in runtime config")
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	dnsRuleIndex := -1
	protectCIDRIndex := -1
	dnsDirectLocalRouteIndex := -1
	dnsCNRouteIndex := -1
	dnsRemoteRouteIndex := -1
	tunCatchAllIndex := -1
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["port"] == "53" && rule["outboundTag"] == "dns-out" {
			dnsRuleIndex = index
		}
		if ips, ok := rule["ip"].([]any); ok && len(ips) == 1 && ips[0] == "127.0.0.0/8" && rule["outboundTag"] == "direct" {
			protectCIDRIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "dns-direct-local" && rule["outboundTag"] == "direct" {
			dnsDirectLocalRouteIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "dns-cn" && rule["outboundTag"] == "direct" {
			dnsCNRouteIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "dns-remote" && rule["balancerTag"] == "node-pool-active" {
			dnsRemoteRouteIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["balancerTag"] == "node-pool-active" {
			tunCatchAllIndex = index
		}
	}

	if dnsRuleIndex == -1 {
		t.Fatal("expected tun DNS interception rule")
	}
	if protectCIDRIndex == -1 {
		t.Fatal("expected protect CIDR direct rule")
	}
	if dnsDirectLocalRouteIndex == -1 {
		t.Fatal("expected dns-direct-local direct route")
	}
	if dnsCNRouteIndex == -1 {
		t.Fatal("expected dns-cn direct route")
	}
	if dnsRemoteRouteIndex == -1 {
		t.Fatal("expected dns-remote pool route")
	}
	if tunCatchAllIndex == -1 {
		t.Fatal("expected tun catch-all rule")
	}
	if dnsRuleIndex >= protectCIDRIndex {
		t.Fatalf("expected DNS rule before protect CIDR rule, got dns=%d protect=%d", dnsRuleIndex, protectCIDRIndex)
	}
	if dnsRuleIndex >= tunCatchAllIndex {
		t.Fatalf("expected DNS rule before tun catch-all, got dns=%d tun=%d", dnsRuleIndex, tunCatchAllIndex)
	}
}

func TestBuildTunRuntimeConfigOmitsUDP443BlockRuleAndAddsTunCatchAll(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "proxy-01", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	uri, err := GenerateShareLink(ShareLinkRequest{
		Protocol: "vmess",
		Address:  "203.0.113.15",
		Port:     443,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Remark:   "pool-active",
		TLS:      "tls",
		SNI:      "example.com",
	})
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
	}, []NodeRecord{{ID: "node-1", URI: uri}})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	outbounds, ok := rendered["outbounds"].([]any)
	if !ok {
		t.Fatalf("expected outbounds, got %#v", rendered["outbounds"])
	}

	blockOutboundCount := 0
	for _, rawOutbound := range outbounds {
		outbound, ok := rawOutbound.(map[string]any)
		if !ok {
			continue
		}
		if outbound["tag"] == "block" && outbound["protocol"] == "blackhole" {
			blockOutboundCount++
		}
	}
	if blockOutboundCount != 1 {
		t.Fatalf("expected exactly one injected block outbound, got %d", blockOutboundCount)
	}

	routing, ok := rendered["routing"].(map[string]any)
	if !ok {
		t.Fatal("expected routing section in runtime config")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		t.Fatal("expected routing rules in runtime config")
	}

	dnsRuleIndex := -1
	udp443BlockFound := false
	tunCatchAllIndex := -1
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["port"] == "53" && rule["outboundTag"] == "dns-out" {
			dnsRuleIndex = index
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["network"] == "udp" && rule["port"] == "443" && rule["outboundTag"] == "block" {
			udp443BlockFound = true
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["balancerTag"] == "node-pool-active" {
			tunCatchAllIndex = index
		}
	}

	if dnsRuleIndex == -1 {
		t.Fatal("expected tun DNS interception rule")
	}
	if udp443BlockFound {
		t.Fatal("did not expect udp/443 block rule in tun runtime")
	}
	if tunCatchAllIndex == -1 {
		t.Fatal("expected tun catch-all rule")
	}
	if dnsRuleIndex >= tunCatchAllIndex {
		t.Fatalf("expected tun catch-all rule after DNS routing, got dns=%d tun=%d", dnsRuleIndex, tunCatchAllIndex)
	}
}

func TestTunManagerInstallPrivilegeUsesPkexecInstaller(t *testing.T) {
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "runtime", "tun")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	scriptsDir := filepath.Join(tempDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	installedDir := filepath.Join(tempDir, "installed")
	installedHelper := filepath.Join(installedDir, "xray-webpanel-tun-helper")
	installedBinary := filepath.Join(installedDir, "xray-webpanel-xray")
	configPath := filepath.Join(tempDir, "config.json")

	config := map[string]any{
		"outbounds": []map[string]any{
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
		"webpanel": map[string]any{
			"tun": map[string]any{
				"stateDir":          stateDir,
				"runtimeConfigPath": filepath.Join(stateDir, "config.json"),
				"interfaceName":     "xray0",
				"remoteDns":         []string{"1.1.1.1", "8.8.8.8"},
				"useSudo":           true,
			},
		},
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	installScriptPath := filepath.Join(scriptsDir, "install-webpanel-tun-sudoers.sh")
	installScript := `#!/bin/sh
set -eu

config=""
target_user=""
xray_src=""

while [ $# -gt 0 ]; do
  case "$1" in
    --config)
      config="$2"
      shift 2
      ;;
    --user)
      target_user="$2"
      shift 2
      ;;
    --xray-src)
      xray_src="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

[ -n "$config" ] || exit 11
[ -n "$target_user" ] || exit 12
[ -n "$xray_src" ] || exit 13

helper_dst="` + installedHelper + `"
xray_dst="` + installedBinary + `"
mkdir -p "$(dirname "$helper_dst")"

cat >"$helper_dst" <<'EOF'
#!/bin/sh
action="${1:-status}"
case "$action" in
  status)
    echo "ACTION=status:stopped"
    ;;
  start)
    echo "ACTION=start:running"
    ;;
  stop)
    echo "ACTION=stop:stopped"
    ;;
  *)
    exit 1
    ;;
esac
EOF
chmod 0755 "$helper_dst"

cat >"$xray_dst" <<'EOF'
placeholder
EOF
cp "$xray_src" "$xray_dst"
chmod 0755 "$xray_dst"

python3 - "$config" "$helper_dst" "$xray_dst" <<'PY'
import json
import sys
from pathlib import Path

config_path = Path(sys.argv[1])
helper_dst = sys.argv[2]
xray_dst = sys.argv[3]

data = json.loads(config_path.read_text(encoding="utf-8"))
tun = data.setdefault("webpanel", {}).setdefault("tun", {})
tun["helperPath"] = helper_dst
tun["binaryPath"] = xray_dst
tun["useSudo"] = True
config_path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

printf 'installed for user=%s from xray=%s\n' "$target_user" "$xray_src"
`
	if err := os.WriteFile(installScriptPath, []byte(installScript), 0o755); err != nil {
		t.Fatalf("write fake install script: %v", err)
	}

	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("create temp bin: %v", err)
	}

	pkexecPath := filepath.Join(tempBin, "pkexec")
	if err := os.WriteFile(pkexecPath, []byte("#!/bin/sh\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("write fake pkexec: %v", err)
	}

	sudoPath := filepath.Join(tempBin, "sudo")
	sudoScript := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-n\" ] && [ \"$2\" = \"-l\" ]; then\n" +
		"  printf '%s\\n' \"$FAKE_SUDO_LISTING\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(sudoPath, []byte(sudoScript), 0o755); err != nil {
		t.Fatalf("write fake sudo: %v", err)
	}

	t.Setenv("PATH", tempBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_SUDO_LISTING", installedHelper+"\n"+installedBinary)

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	status := tunManager.InstallPrivilege()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "stopped" {
		t.Fatalf("expected stopped status after install, got %q", status.Status)
	}
	if !status.HelperExists {
		t.Fatal("expected helper to exist after install")
	}
	if !status.ElevationReady {
		t.Fatal("expected elevation to be ready after install")
	}
	if status.HelperPath != installedHelper {
		t.Fatalf("expected helper path %q, got %q", installedHelper, status.HelperPath)
	}
	if status.BinaryPath != installedBinary {
		t.Fatalf("expected binary path %q, got %q", installedBinary, status.BinaryPath)
	}
	if !strings.Contains(status.LastOutput, "installed for user=") {
		t.Fatalf("expected installer output in status, got %q", status.LastOutput)
	}
	if status.Message != "Privilege helper is installed" {
		t.Fatalf("expected install success message, got %q", status.Message)
	}
	if status.PrivilegeInstallRecommended {
		t.Fatal("expected privilege install recommendation to be cleared after install")
	}
	if !status.HelperCurrent {
		t.Fatal("expected helper to be current after install")
	}
	if !status.BinaryCurrent {
		t.Fatal("expected binary to be current after install")
	}
}

func TestPrepareCommandInvocationUsesInterpreterForPathScript(t *testing.T) {
	tempDir := t.TempDir()
	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("create temp bin: %v", err)
	}

	fakeCommandPath := filepath.Join(tempBin, "fake-pkexec")
	if err := os.WriteFile(fakeCommandPath, []byte("#!/bin/sh\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("write fake command: %v", err)
	}

	t.Setenv("PATH", tempBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmdName, cmdArgs := prepareCommandInvocation("fake-pkexec", []string{"--flag", "value"})
	if !strings.HasPrefix(filepath.Base(cmdName), "sh") {
		t.Fatalf("expected sh interpreter, got %q", cmdName)
	}
	if len(cmdArgs) != 3 {
		t.Fatalf("expected script path plus original args, got %v", cmdArgs)
	}
	if cmdArgs[0] != fakeCommandPath {
		t.Fatalf("expected first arg to be script path %q, got %q", fakeCommandPath, cmdArgs[0])
	}
	if cmdArgs[1] != "--flag" || cmdArgs[2] != "value" {
		t.Fatalf("expected original args to be preserved, got %v", cmdArgs)
	}
}

func TestPrepareCommandInvocationUsesInterpreterForDirectScriptPath(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "helper.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf 'ok\\n'\n"), 0o755); err != nil {
		t.Fatalf("write helper script: %v", err)
	}

	cmdName, cmdArgs := prepareCommandInvocation(scriptPath, []string{"status"})
	if !strings.HasPrefix(filepath.Base(cmdName), "sh") {
		t.Fatalf("expected sh interpreter, got %q", cmdName)
	}
	if len(cmdArgs) != 2 {
		t.Fatalf("expected script path and original arg, got %v", cmdArgs)
	}
	if cmdArgs[0] != scriptPath || cmdArgs[1] != "status" {
		t.Fatalf("expected script invocation to preserve path and arg, got %v", cmdArgs)
	}
}

func TestTunManagerStatusDetectsStalePrivilegeArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "runtime", "tun")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	scriptsDir := filepath.Join(tempDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	repoHelper := filepath.Join(scriptsDir, "webpanel-tun-helper.sh")
	if err := os.WriteFile(repoHelper, []byte("#!/bin/sh\necho repo-helper\n"), 0o755); err != nil {
		t.Fatalf("write repo helper: %v", err)
	}

	installedDir := filepath.Join(tempDir, "installed")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatalf("create installed dir: %v", err)
	}
	installedHelper := filepath.Join(installedDir, "xray-webpanel-tun-helper")
	installedHelperScript := "#!/bin/sh\n" +
		"action=\"${1:-status}\"\n" +
		"if [ \"$action\" = \"status\" ]; then\n" +
		"  echo \"ACTION=status:stopped\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(installedHelper, []byte(installedHelperScript), 0o755); err != nil {
		t.Fatalf("write installed helper: %v", err)
	}

	installedBinary := filepath.Join(installedDir, "xray-webpanel-xray")
	if err := os.WriteFile(installedBinary, []byte("#!/bin/sh\necho stale-binary\n"), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	config := map[string]any{
		"outbounds": []map[string]any{
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
		"webpanel": map[string]any{
			"tun": map[string]any{
				"helperPath":        installedHelper,
				"binaryPath":        installedBinary,
				"stateDir":          stateDir,
				"runtimeConfigPath": filepath.Join(stateDir, "config.json"),
				"interfaceName":     "xray0",
				"remoteDns":         []string{"1.1.1.1", "8.8.8.8"},
				"useSudo":           true,
			},
		},
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("create temp bin: %v", err)
	}
	sudoPath := filepath.Join(tempBin, "sudo")
	sudoScript := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-n\" ] && [ \"$2\" = \"-l\" ]; then\n" +
		"  printf '%s\\n' \"$FAKE_SUDO_LISTING\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(sudoPath, []byte(sudoScript), 0o755); err != nil {
		t.Fatalf("write fake sudo: %v", err)
	}

	t.Setenv("PATH", tempBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_SUDO_LISTING", installedHelper+"\n"+installedBinary)

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	status := tunManager.Status()
	if status == nil {
		t.Fatal("expected tun status")
	}
	if !status.HelperExists {
		t.Fatal("expected helper to exist")
	}
	if !status.ElevationReady {
		t.Fatal("expected elevation to be ready")
	}
	if status.HelperCurrent {
		t.Fatal("expected helper to be marked stale")
	}
	if status.BinaryCurrent {
		t.Fatal("expected binary to be marked stale")
	}
	if !status.PrivilegeInstallRecommended {
		t.Fatal("expected repair to be recommended")
	}
	if status.Status != "stopped" {
		t.Fatalf("expected stopped status, got %q", status.Status)
	}
}

func TestTunManagerStartBlocksWhenPrivilegeArtifactsAreStale(t *testing.T) {
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "runtime", "tun")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	scriptsDir := filepath.Join(tempDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}
	repoHelper := filepath.Join(scriptsDir, "webpanel-tun-helper.sh")
	if err := os.WriteFile(repoHelper, []byte("#!/bin/sh\necho repo-helper\n"), 0o755); err != nil {
		t.Fatalf("write repo helper: %v", err)
	}

	installedDir := filepath.Join(tempDir, "installed")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatalf("create installed dir: %v", err)
	}
	installedHelper := filepath.Join(installedDir, "xray-webpanel-tun-helper")
	installedHelperScript := "#!/bin/sh\n" +
		"action=\"${1:-status}\"\n" +
		"if [ \"$action\" = \"status\" ]; then\n" +
		"  echo \"ACTION=status:stopped\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo unexpected-action >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(installedHelper, []byte(installedHelperScript), 0o755); err != nil {
		t.Fatalf("write installed helper: %v", err)
	}

	installedBinary := filepath.Join(installedDir, "xray-webpanel-xray")
	if err := os.WriteFile(installedBinary, []byte("#!/bin/sh\necho stale-binary\n"), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	config := map[string]any{
		"outbounds": []map[string]any{
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
		"webpanel": map[string]any{
			"tun": map[string]any{
				"helperPath":        installedHelper,
				"binaryPath":        installedBinary,
				"stateDir":          stateDir,
				"runtimeConfigPath": filepath.Join(stateDir, "config.json"),
				"interfaceName":     "xray0",
				"remoteDns":         []string{"1.1.1.1", "8.8.8.8"},
				"useSudo":           true,
			},
		},
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tempBin := filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(tempBin, 0o755); err != nil {
		t.Fatalf("create temp bin: %v", err)
	}
	sudoPath := filepath.Join(tempBin, "sudo")
	sudoScript := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-n\" ] && [ \"$2\" = \"-l\" ]; then\n" +
		"  printf '%s\\n' \"$FAKE_SUDO_LISTING\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(sudoPath, []byte(sudoScript), 0o755); err != nil {
		t.Fatalf("write fake sudo: %v", err)
	}

	t.Setenv("PATH", tempBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_SUDO_LISTING", installedHelper+"\n"+installedBinary)

	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	status := tunManager.Start(nil)
	if status == nil {
		t.Fatal("expected tun status")
	}
	if status.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", status.Status)
	}
	if !status.PrivilegeInstallRecommended {
		t.Fatal("expected repair to be recommended")
	}
	if status.Message != "Install or repair the privilege helper before enabling transparent TUN mode" {
		t.Fatalf("unexpected message: %q", status.Message)
	}
}
