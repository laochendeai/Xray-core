package webpanel

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePreviewTunTestConfig(t *testing.T, helperPath string, runtimeConfigPath string, destinationBindings string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	config := `{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  },
  "webpanel": {
    "tun": {
      "helperPath": "` + helperPath + `",
      "runtimeConfigPath": "` + runtimeConfigPath + `",
      "interfaceName": "xray0",
      "mtu": 1500,
      "useSudo": false,
      "protectDomains": ["full:localhost"],
      "protectCidrs": ["127.0.0.0/8"],
      "destinationBindings": ` + destinationBindings + `
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writePreviewTunHelper(t *testing.T, action string) string {
	t.Helper()

	helperPath := filepath.Join(t.TempDir(), "webpanel-tun-helper.sh")
	content := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  status)\n" +
		"    echo \"ACTION=status:" + action + "\"\n" +
		"    ;;\n" +
		"  *)\n" +
		"    echo \"ACTION=$1:" + action + "\"\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(helperPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	return helperPath
}

func TestWebPanelPreviewTunRouteUsesRunningRuntimeConfig(t *testing.T) {
	t.Parallel()

	helperPath := writePreviewTunHelper(t, "running")
	runtimeConfigPath := filepath.Join(t.TempDir(), "runtime.json")
	runtimeConfig := `{
  "routing": {
    "rules": [
      {
        "type": "field",
        "domain": ["domain:api.openai.com"],
        "outboundTag": "pool-active-runtime-node"
      },
      {
        "type": "field",
        "inboundTag": ["tun-in"],
        "balancerTag": "node-pool-active"
      }
    ]
  }
}`
	if err := os.WriteFile(runtimeConfigPath, []byte(runtimeConfig), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	configPath := writePreviewTunTestConfig(t, helperPath, runtimeConfigPath, `[
        {"preset":"openai","nodeId":"different-node"}
      ]`)
	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	wp := &WebPanel{tunManager: tunManager}
	result, err := wp.previewTunRoute(routingTestRequest{
		Scope:      "tun",
		Domain:     "api.openai.com",
		Port:       443,
		Network:    "tcp",
		InboundTag: "tun-in",
	})
	if err != nil {
		t.Fatalf("preview tun route: %v", err)
	}

	if result.OutboundTag != "pool-active-runtime-node" {
		t.Fatalf("expected running runtime config to win, got %+v", result)
	}
}

func TestWebPanelPreviewTunRouteUsesEligibleNodesWhenStopped(t *testing.T) {
	t.Parallel()

	helperPath := writePreviewTunHelper(t, "stopped")
	runtimeConfigPath := filepath.Join(t.TempDir(), "runtime.json")
	configPath := writePreviewTunTestConfig(t, helperPath, runtimeConfigPath, `[
        {"preset":"openai","nodeId":"target-node"}
      ]`)
	tunManager, err := NewTunManager(configPath)
	if err != nil {
		t.Fatalf("new tun manager: %v", err)
	}

	now := time.Now()
	wp := &WebPanel{
		tunManager: tunManager,
		subManager: &SubscriptionManager{
			state: &NodePoolState{
				ValidationConfig: defaultValidationConfig(),
				Nodes: []NodeRecord{
					{
						ID:               "target-node",
						Status:           NodeStatusActive,
						URI:              mustGenerateTunTestURI(t, "203.0.113.10"),
						Protocol:         "vmess",
						AvgDelayMs:       120,
						ConsecutiveFails: 1,
						LastCheckedAt:    &now,
					},
					{
						ID:               "fallback-node",
						Status:           NodeStatusActive,
						URI:              mustGenerateTunTestURI(t, "203.0.113.11"),
						Protocol:         "vmess",
						AvgDelayMs:       80,
						ConsecutiveFails: 0,
						LastCheckedAt:    &now,
					},
				},
			},
		},
	}

	result, err := wp.previewTunRoute(routingTestRequest{
		Scope:      "tun",
		Domain:     "api.openai.com",
		Port:       443,
		Network:    "tcp",
		InboundTag: "tun-in",
	})
	if err != nil {
		t.Fatalf("preview tun route: %v", err)
	}

	if result.OutboundTag != "" {
		t.Fatalf("expected ineligible binding target to be skipped, got %+v", result)
	}
	if result.BalancerTag != "node-pool-active" {
		t.Fatalf("expected catch-all balancer when target node is not TUN-eligible, got %+v", result)
	}
}
