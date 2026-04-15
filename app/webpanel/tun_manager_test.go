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
	}, []NodeRecord{{ID: "node-1", URI: uri}})
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
	}, []NodeRecord{{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.31"), AvgDelayMs: 88, TotalPings: 10}, {ID: "node-2", URI: mustGenerateTunTestURI(t, "203.0.113.32"), AvgDelayMs: 21, TotalPings: 10}})
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
	}, []NodeRecord{{ID: "node-a", URI: mustGenerateTunTestURI(t, "203.0.113.41"), AvgDelayMs: 80, TotalPings: 10, FailedPings: 0}, {ID: "node-b", URI: mustGenerateTunTestURI(t, "203.0.113.42"), AvgDelayMs: 25, TotalPings: 10, FailedPings: 1}, {ID: "node-c", URI: mustGenerateTunTestURI(t, "203.0.113.43"), AvgDelayMs: 60, TotalPings: 6, FailedPings: 0}, {ID: "node-d", URI: mustGenerateTunTestURI(t, "203.0.113.44"), AvgDelayMs: 10, TotalPings: 10, FailedPings: 5}})
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

	for _, expected := range []string{"pool-active-node-a", "pool-active-node-b", "pool-active-node-c"} {
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

	_, err := buildTunRuntimeConfig([]byte(`{"outbounds":[{"tag":"direct","protocol":"freedom"}]}`), &TunFeatureSettings{InterfaceName: "xray0"}, nil)
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

	uri, err := GenerateShareLink(ShareLinkRequest{Protocol: "vmess", Address: "203.0.113.12", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Remark: "pool-active", TLS: "tls", SNI: "example.com"})
	if err != nil { t.Fatalf("generate share link: %v", err) }

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{InterfaceName: "xray0", MTU: 1500, ProtectCIDRs: []string{"127.0.0.0/8"}, ProtectDomains: []string{"full:localhost"}, DestinationBindings: []TunDestinationBinding{{Preset: string(TunDestinationBindingPresetOpenAI), NodeID: "node-2"}}}, []NodeRecord{{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.11")}, {ID: "node-2", URI: uri}})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("decode rendered config: %v", err)
	}

	routing := rendered["routing"].(map[string]any)
	rules := routing["rules"].([]any)
	openAIProxyIndex := -1
	tunCatchAllIndex := -1
	autoFallbackIndex := -1
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok { continue }
		if domains, ok := rule["domain"].([]any); ok && len(domains) == 1 && domains[0] == "domain:openai.com" && rule["outboundTag"] == "proxy-01" { openAIProxyIndex = index }
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["balancerTag"] == "node-pool-active" { tunCatchAllIndex = index }
		if rule["balancerTag"] == "auto" { autoFallbackIndex = index }
	}
	if openAIProxyIndex == -1 { t.Fatal("expected existing specific proxy rule to be preserved") }
	if tunCatchAllIndex == -1 { t.Fatal("expected tun catch-all rule to be injected") }
	if openAIProxyIndex >= tunCatchAllIndex { t.Fatalf("expected specific proxy rule before tun catch-all, got proxy=%d tun=%d", openAIProxyIndex, tunCatchAllIndex) }
	if autoFallbackIndex != -1 { t.Fatalf("expected generic auto fallback rule to be removed from tun runtime, got index=%d", autoFallbackIndex) }
}

func TestBuildTunRuntimeConfigPlacesDestinationBindingsBeforeTunCatchAll(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "domain": ["domain:priority.example"],
        "outboundTag": "proxy-01"
      }
    ]
  }
}`)

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{InterfaceName: "xray0", MTU: 1500, ProtectCIDRs: []string{"127.0.0.0/8"}, ProtectDomains: []string{"full:localhost"}, DestinationBindings: []TunDestinationBinding{{Preset: string(TunDestinationBindingPresetOpenAI), NodeID: "node-2"}}}, []NodeRecord{{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.11")}, {ID: "node-2", URI: mustGenerateTunTestURI(t, "203.0.113.12")}})
	if err != nil { t.Fatalf("build tun runtime config: %v", err) }

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil { t.Fatalf("decode rendered config: %v", err) }
	routing := rendered["routing"].(map[string]any)
	rules := routing["rules"].([]any)
	priorityIndex, bindingIndex, catchAllIndex := -1, -1, -1
	for index, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok { continue }
		if domains, ok := rule["domain"].([]any); ok {
			for _, domain := range domains {
				switch domain {
				case "domain:priority.example":
					priorityIndex = index
				case "domain:openai.com":
					if rule["outboundTag"] == "pool-active-node-2" { bindingIndex = index }
				}
			}
		}
		if inboundTags, ok := rule["inboundTag"].([]any); ok && len(inboundTags) == 1 && inboundTags[0] == "tun-in" && rule["balancerTag"] == "node-pool-active" { catchAllIndex = index }
	}
	if priorityIndex == -1 { t.Fatal("expected existing priority rule to be preserved") }
	if bindingIndex == -1 { t.Fatal("expected destination binding rule to be injected") }
	if catchAllIndex == -1 { t.Fatal("expected tun catch-all rule to be injected") }
	if priorityIndex >= bindingIndex { t.Fatalf("expected preserved priority rule before destination binding, got priority=%d binding=%d", priorityIndex, bindingIndex) }
	if bindingIndex >= catchAllIndex { t.Fatalf("expected destination binding before tun catch-all, got binding=%d catch-all=%d", bindingIndex, catchAllIndex) }
}

func TestBuildTunRuntimeConfigSkipsDestinationBindingsForMissingActiveNodes(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{InterfaceName: "xray0", MTU: 1500, ProtectCIDRs: []string{"127.0.0.0/8"}, ProtectDomains: []string{"full:localhost"}, DestinationBindings: []TunDestinationBinding{{Preset: string(TunDestinationBindingPresetCustom), Domains: []string{"*.example.com"}, NodeID: "missing-node"}}}, []NodeRecord{{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.21")}})
	if err != nil { t.Fatalf("build tun runtime config: %v", err) }

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil { t.Fatalf("decode rendered config: %v", err) }
	routing := rendered["routing"].(map[string]any)
	rules := routing["rules"].([]any)
	for _, rawRule := range rules {
		rule, ok := rawRule.(map[string]any)
		if !ok { continue }
		if domains, ok := rule["domain"].([]any); ok {
			for _, domain := range domains {
				if domain == "domain:example.com" { t.Fatalf("expected missing-node destination binding to be skipped, got rule %v", rule) }
			}
		}
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

	uri, err := GenerateShareLink(ShareLinkRequest{Protocol: "vmess", Address: "203.0.113.21", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Remark: "pool-active", TLS: "tls", SNI: "example.com"})
	if err != nil { t.Fatalf("generate share link: %v", err) }

	output, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{InterfaceName: "xray0", MTU: 1500, ProtectCIDRs: []string{"127.0.0.0/8"}, ProtectDomains: []string{"full:localhost"}}, []NodeRecord{{ID: "node-1", URI: uri}})
	if err != nil { t.Fatalf("build tun runtime config: %v", err) }

	var rendered map[string]any
	if err := json.Unmarshal(output, &rendered); err != nil { t.Fatalf("decode rendered config: %v", err) }
	routing := rendered["routing"].(map[string]any)
	rules := routing["rules"].([]any)
	if len(rules) < 2 { t.Fatalf("expected routing rules, got %v", rules) }
}

func TestNormalizeTunDestinationBindingDedupeAndFallbackModes(t *testing.T) {
	t.Parallel()

	bindings := normalizeTunDestinationBindings([]TunDestinationBinding{{Preset: " openai ", NodeID: " primary-1 ", FallbackNodeIDs: []string{" fallback-1 ", "fallback-1", "primary-1", ""}, SelectionMode: " primary_only "}, {Preset: "openai", NodeID: "primary-1", FallbackNodeIDs: []string{"fallback-1"}, SelectionMode: "failover_ordered"}, {Preset: "custom", Domains: []string{" *.example.com ", "full:api.example.com", "*.example.com"}, NodeID: "primary-2", FallbackNodeIDs: []string{"fallback-2", "fallback-2"}, SelectionMode: "failover_fastest"}})

	if len(bindings) != 2 {
		t.Fatalf("expected 2 normalized bindings, got %d: %#v", len(bindings), bindings)
	}
	if got := bindings[0]; got.SelectionMode != string(TunDestinationBindingSelectionModeFailoverOrdered) {
		t.Fatalf("expected primary_only with fallbacks to normalize to failover_ordered, got %#v", got)
	}
	if got := bindings[1]; got.SelectionMode != string(TunDestinationBindingSelectionModeFailoverFastest) {
		t.Fatalf("expected fastest mode to be preserved, got %#v", got)
	} else if !reflect.DeepEqual(got.Domains, []string{"domain:example.com", "full:api.example.com"}) {
		t.Fatalf("expected custom domains to normalize and dedupe, got %#v", got.Domains)
	} else if !reflect.DeepEqual(got.FallbackNodeIDs, []string{"fallback-2"}) {
		t.Fatalf("expected fallback node IDs to dedupe, got %#v", got.FallbackNodeIDs)
	}
}

func TestSelectTunDestinationBindingNodeHonorsFallbackModes(t *testing.T) {
	t.Parallel()

	active := map[string]NodeRecord{"primary": {ID: "primary", AvgDelayMs: 50, TotalPings: 10}, "fallback": {ID: "fallback", AvgDelayMs: 20, TotalPings: 10}, "fast-a": {ID: "fast-a", AvgDelayMs: 60, TotalPings: 10}, "fast-b": {ID: "fast-b", AvgDelayMs: 15, TotalPings: 10}}
	if got := selectTunDestinationBindingNode(TunDestinationBinding{NodeID: "primary", FallbackNodeIDs: []string{"fallback"}, SelectionMode: string(TunDestinationBindingSelectionModePrimaryOnly)}, active); got != "primary" { t.Fatalf("expected primary_only to choose primary, got %q", got) }
	if got := selectTunDestinationBindingNode(TunDestinationBinding{NodeID: "missing", FallbackNodeIDs: []string{"fallback"}, SelectionMode: string(TunDestinationBindingSelectionModeFailoverOrdered)}, active); got != "fallback" { t.Fatalf("expected ordered failover to choose fallback, got %q", got) }
	if got := selectTunDestinationBindingNode(TunDestinationBinding{NodeID: "missing", FallbackNodeIDs: []string{"fast-a", "fast-b"}, SelectionMode: string(TunDestinationBindingSelectionModeFailoverFastest)}, active); got != "fast-b" { t.Fatalf("expected fastest failover to choose fastest active candidate, got %q", got) }
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestTunManagerUpdateEditableSettingsPersistsAggregationScaffolding(t *testing.T) { t.Parallel(); configPath := filepath.Join(t.TempDir(), "config.json"); if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {}
  }
}
`), 0o644); err != nil { t.Fatalf("write config: %v", err) }; tunManager, err := NewTunManager(configPath); if err != nil { t.Fatalf("new tun manager: %v", err) }; settings, err := tunManager.UpdateEditableSettings(TunEditableSettings{SelectionPolicy: string(TunSelectionPolicyFastest), RouteMode: string(TunRouteModeStrictProxy), Aggregation: TunAggregationSettings{Enabled: true, Mode: "REDUNDANT_2", MaxPathsPerSession: 12, SchedulerPolicy: "single_best", RelayEndpoint: "  https://relay.example/ingress  ", Health: TunAggregationHealthSettings{MaxSessionLossPct: 9, MaxPathJitterMs: 45, RollbackOnConsecutiveFailures: 7}}}); if err != nil { t.Fatalf("update editable settings: %v", err) }; expected := TunAggregationSettings{Enabled: true, Mode: string(TunAggregationModeRedundant2), MaxPathsPerSession: 8, SchedulerPolicy: string(TunAggregationSchedulerPolicySingleBest), RelayEndpoint: "https://relay.example/ingress", Health: TunAggregationHealthSettings{MaxSessionLossPct: 9, MaxPathJitterMs: 45, RollbackOnConsecutiveFailures: 7}}; if !reflect.DeepEqual(settings.Aggregation, expected) { t.Fatalf("expected normalized aggregation settings %#v, got %#v", expected, settings.Aggregation) }; loaded, err := tunManager.EditableSettings(); if err != nil { t.Fatalf("reload editable settings: %v", err) }; if !reflect.DeepEqual(loaded.Aggregation, expected) { t.Fatalf("expected persisted aggregation settings %#v, got %#v", expected, loaded.Aggregation) }
}

func TestTunManagerEditableSettingsNormalizesWildcardProtectDomainsFromConfig(t *testing.T) { t.Parallel(); configPath := filepath.Join(t.TempDir(), "config.json"); if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {
      "protectDomains": ["*.example.com", ".internal.example", "full:exact.example"]
    }
  }
}
`), 0o644); err != nil { t.Fatalf("write config: %v", err) }; tunManager, err := NewTunManager(configPath); if err != nil { t.Fatalf("new tun manager: %v", err) }; settings, err := tunManager.EditableSettings(); if err != nil { t.Fatalf("editable settings: %v", err) }; expected := []string{"domain:example.com", "domain:internal.example", "full:exact.example"}; if !reflect.DeepEqual(settings.ProtectDomains, expected) { t.Fatalf("expected normalized protect domains %v, got %v", expected, settings.ProtectDomains) }
}

func TestTunManagerEditableSettingsPersistRemoteDNS(t *testing.T) { t.Parallel(); configPath := filepath.Join(t.TempDir(), "config.json"); if err := os.WriteFile(configPath, []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "webpanel": {
    "tun": {}
  }
}
`), 0o644); err != nil { t.Fatalf("write config: %v", err) }; tunManager, err := NewTunManager(configPath); if err != nil { t.Fatalf("new tun manager: %v", err) }; settings, err := tunManager.UpdateEditableSettings(TunEditableSettings{SelectionPolicy: string(TunSelectionPolicyFastest), RouteMode: string(TunRouteModeStrictProxy), RemoteDNS: []string{"1.1.1.1", "8.8.8.8", "1.1.1.1"}}); if err != nil { t.Fatalf("update editable settings: %v", err) }; if !reflect.DeepEqual(settings.RemoteDNS, []string{"1.1.1.1", "8.8.8.8"}) { t.Fatalf("unexpected remote dns list: %v", settings.RemoteDNS) }; loaded, err := tunManager.EditableSettings(); if err != nil { t.Fatalf("reload editable settings: %v", err) }; if !reflect.DeepEqual(loaded.RemoteDNS, settings.RemoteDNS) { t.Fatalf("expected remote dns to persist, got %v", loaded.RemoteDNS) }
}
