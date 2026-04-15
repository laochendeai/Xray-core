package webpanel

import (
	"testing"
	"time"
)

func TestClassifyNodeIntelligenceMarksHostingExitSuspicious(t *testing.T) {
	t.Parallel()

	result := classifyNodeIntelligence(
		NodeRecord{
			ExitIPStatus:     NodeExitIPStatusAvailable,
			ExitIP:           "203.0.113.10",
			TotalPings:       24,
			FailedPings:      1,
			AvgDelayMs:       180,
			ConsecutiveFails: 0,
		},
		defaultValidationConfig(),
		nodeIPConnectionLookupResult{
			ASN:       14061,
			Org:       "DigitalOcean, LLC",
			ISP:       "DigitalOcean, LLC",
			Domain:    "digitalocean.com",
			CheckedAt: time.Now().UTC(),
		},
	)

	if result.NetworkType != NodeNetworkTypeDatacenterLikely {
		t.Fatalf("expected datacenter-likely network type, got %q", result.NetworkType)
	}
	if result.Cleanliness != CleanlinessSuspicious {
		t.Fatalf("expected suspicious cleanliness, got %q", result.Cleanliness)
	}
	if result.NetworkConfidence != NodeIntelligenceConfidenceHigh {
		t.Fatalf("expected high network confidence, got %q", result.NetworkConfidence)
	}
	if result.CleanlinessReason != nodeIntelligenceReasonDatacenterExit {
		t.Fatalf("expected datacenter cleanliness reason, got %q", result.CleanlinessReason)
	}
}

func TestClassifyNodeIntelligenceMarksResidentialStableExitTrusted(t *testing.T) {
	t.Parallel()

	cfg := defaultValidationConfig()
	cfg.MinSamples = 10
	cfg.MaxFailRate = 0.30
	cfg.MaxAvgDelayMs = 1000

	result := classifyNodeIntelligence(
		NodeRecord{
			ExitIPStatus:     NodeExitIPStatusAvailable,
			ExitIP:           "198.51.100.8",
			TotalPings:       20,
			FailedPings:      1,
			AvgDelayMs:       220,
			ConsecutiveFails: 0,
		},
		cfg,
		nodeIPConnectionLookupResult{
			ASN:       7922,
			Org:       "Comcast Cable Communications, LLC",
			ISP:       "Comcast Cable Communications, LLC",
			Domain:    "comcast.net",
			CheckedAt: time.Now().UTC(),
		},
	)

	if result.NetworkType != NodeNetworkTypeResidentialLikely {
		t.Fatalf("expected residential-likely network type, got %q", result.NetworkType)
	}
	if result.Cleanliness != CleanlinessTrusted {
		t.Fatalf("expected trusted cleanliness, got %q", result.Cleanliness)
	}
	if result.CleanlinessReason != nodeIntelligenceReasonResidentialStableExit {
		t.Fatalf("expected residential stable reason, got %q", result.CleanlinessReason)
	}
}

func TestClassifyNodeIntelligenceMarksISPLikeExitUnknown(t *testing.T) {
	t.Parallel()

	cfg := defaultValidationConfig()
	cfg.MinSamples = 10
	cfg.MaxFailRate = 0.30
	cfg.MaxAvgDelayMs = 1000

	result := classifyNodeIntelligence(
		NodeRecord{
			ExitIPStatus:     NodeExitIPStatusAvailable,
			ExitIP:           "198.51.100.20",
			TotalPings:       20,
			FailedPings:      0,
			AvgDelayMs:       110,
			ConsecutiveFails: 0,
		},
		cfg,
		nodeIPConnectionLookupResult{
			ASN:       64512,
			Org:       "Example Telecom Communications",
			ISP:       "Example Telecom Communications",
			Domain:    "example.net",
			CheckedAt: time.Now().UTC(),
		},
	)

	if result.NetworkType != NodeNetworkTypeISPLikely {
		t.Fatalf("expected isp-likely network type, got %q", result.NetworkType)
	}
	if result.Cleanliness != CleanlinessUnknown {
		t.Fatalf("expected unknown cleanliness, got %q", result.Cleanliness)
	}
	if result.NetworkReason != nodeIntelligenceReasonISPKeywordMatch {
		t.Fatalf("expected isp keyword reason, got %q", result.NetworkReason)
	}
	if result.CleanlinessReason != nodeIntelligenceReasonInsufficientSignal {
		t.Fatalf("expected insufficient signal cleanliness reason, got %q", result.CleanlinessReason)
	}
}

func TestClassifyNodeIntelligenceKeepsLowSignalUnknown(t *testing.T) {
	t.Parallel()

	result := classifyNodeIntelligence(
		NodeRecord{
			ExitIPStatus:     NodeExitIPStatusAvailable,
			ExitIP:           "198.51.100.20",
			TotalPings:       3,
			FailedPings:      0,
			AvgDelayMs:       110,
			ConsecutiveFails: 0,
		},
		defaultValidationConfig(),
		nodeIPConnectionLookupResult{
			ASN:       64512,
			Org:       "Example Networks",
			ISP:       "Example Networks",
			Domain:    "example.net",
			CheckedAt: time.Now().UTC(),
		},
	)

	if result.NetworkType != NodeNetworkTypeUnknown {
		t.Fatalf("expected unknown network type, got %q", result.NetworkType)
	}
	if result.Cleanliness != CleanlinessUnknown {
		t.Fatalf("expected unknown cleanliness, got %q", result.Cleanliness)
	}
	if result.CleanlinessReason != nodeIntelligenceReasonInsufficientSignal {
		t.Fatalf("expected insufficient signal reason, got %q", result.CleanlinessReason)
	}
}

func TestClassifyNodeIntelligenceMarksHighFailRateSuspicious(t *testing.T) {
	t.Parallel()

	cfg := defaultValidationConfig()
	cfg.MinSamples = 10

	result := classifyNodeIntelligence(
		NodeRecord{
			ExitIPStatus:     NodeExitIPStatusAvailable,
			ExitIP:           "198.51.100.30",
			TotalPings:       20,
			FailedPings:      14,
			AvgDelayMs:       0,
			ConsecutiveFails: 6,
		},
		cfg,
		nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     "lookup timeout",
		},
	)

	if result.Cleanliness != CleanlinessSuspicious {
		t.Fatalf("expected suspicious cleanliness, got %q", result.Cleanliness)
	}
	if result.CleanlinessReason != nodeIntelligenceReasonProbeFailureRateHigh {
		t.Fatalf("expected probe failure reason, got %q", result.CleanlinessReason)
	}
	if result.NetworkType != NodeNetworkTypeUnknown {
		t.Fatalf("expected unknown network type on lookup failure, got %q", result.NetworkType)
	}
}
