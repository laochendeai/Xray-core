package webpanel

import "testing"

func TestPreviewTunRouteMatchesDestinationBinding(t *testing.T) {
	t.Parallel()

	baseConfig := []byte(`{
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ],
  "routing": {
    "rules": []
  }
}`)

	runtimeConfig, err := buildTunRuntimeConfig(baseConfig, &TunFeatureSettings{
		InterfaceName:  "xray0",
		MTU:            1500,
		ProtectCIDRs:   []string{"127.0.0.0/8"},
		ProtectDomains: []string{"full:localhost"},
		DestinationBindings: []TunDestinationBinding{
			{Preset: string(TunDestinationBindingPresetOpenAI), NodeID: "node-2"},
		},
	}, []NodeRecord{
		{ID: "node-1", URI: mustGenerateTunTestURI(t, "203.0.113.31")},
		{ID: "node-2", URI: mustGenerateTunTestURI(t, "203.0.113.32")},
	})
	if err != nil {
		t.Fatalf("build tun runtime config: %v", err)
	}

	result, err := previewTunRoute(runtimeConfig, tunRoutePreviewRequest{
		Domain:  "api.openai.com",
		Port:    443,
		Network: "tcp",
	})
	if err != nil {
		t.Fatalf("preview tun route: %v", err)
	}

	if result.OutboundTag != "pool-active-node-2" {
		t.Fatalf("expected destination binding to select node-2, got %+v", result)
	}
	if result.InboundTag != "tun-in" {
		t.Fatalf("expected tun preview inbound tag to default to tun-in, got %+v", result)
	}
}
