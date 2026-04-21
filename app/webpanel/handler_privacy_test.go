package webpanel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectPrivacyBrowserPolicyDetectsMatchingPolicy(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, privacyBrowserPolicyFileName)
	if err := os.WriteFile(policyPath, []byte(`{
  "WebRtcIPHandling": "disable_non_proxied_udp",
  "DnsOverHttpsMode": "off"
}`), 0644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	status := inspectPrivacyBrowserPolicy("linux", []privacyBrowserPolicyTarget{
		{
			Browser:    "Test Chromium",
			PolicyDirs: []string{dir},
		},
	}, privacyBrowserPolicyFileName)

	if !status.Supported {
		t.Fatalf("expected linux policy status to be supported")
	}
	if !status.Installed || !status.Configured {
		t.Fatalf("expected matching policy to be installed and configured: %+v", status)
	}
	if status.DetectedBrowsers != 1 || status.ConfiguredBrowsers != 1 || status.ConfiguredPolicyFiles != 1 {
		t.Fatalf("unexpected policy counters: %+v", status)
	}
	if len(status.Targets) != 1 || !status.Targets[0].Configured || !status.Targets[0].Paths[0].Matching {
		t.Fatalf("expected target path to be matching: %+v", status.Targets)
	}
}

func TestInspectPrivacyBrowserPolicyRejectsMismatchedPolicy(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, privacyBrowserPolicyFileName)
	if err := os.WriteFile(policyPath, []byte(`{
  "WebRtcIPHandling": "default",
  "DnsOverHttpsMode": "automatic"
}`), 0644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	status := inspectPrivacyBrowserPolicy("linux", []privacyBrowserPolicyTarget{
		{
			Browser:    "Test Chromium",
			PolicyDirs: []string{dir},
		},
	}, privacyBrowserPolicyFileName)

	if status.Installed || status.Configured {
		t.Fatalf("did not expect mismatched policy to be configured: %+v", status)
	}
	if len(status.Targets) != 1 || status.Targets[0].Configured || status.Targets[0].Paths[0].Matching {
		t.Fatalf("expected target path to be non-matching: %+v", status.Targets)
	}
}

func TestInspectPrivacyBrowserPolicyReportsUnsupportedPlatform(t *testing.T) {
	status := inspectPrivacyBrowserPolicy("darwin", defaultPrivacyBrowserPolicyTargets, privacyBrowserPolicyFileName)

	if status.Supported {
		t.Fatalf("expected non-linux policy automation to be unsupported")
	}
	if status.UnsupportedReason == "" {
		t.Fatalf("expected unsupported reason")
	}
}
