package webpanel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core "github.com/xtls/xray-core/core"
)

func TestReleaseCheckerDetectsAvailableUpdate(t *testing.T) {
	t.Parallel()

	latestVersion := bumpedPatchVersion(t, core.Version())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = io.WriteString(w, sampleReleaseFeed(serverURL(r), latestVersion, "2026-03-28T19:44:23Z"))
	}))
	defer server.Close()

	checker := newReleaseChecker(server.Client(), server.URL, "test/source", time.Hour)
	checker.now = func() time.Time { return time.Date(2026, 4, 3, 3, 0, 0, 0, time.UTC) }

	status := checker.Check(context.Background(), true)
	if status.Status != "ok" {
		t.Fatalf("expected ok status, got %q (%s)", status.Status, status.Message)
	}
	if !status.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
	if status.LatestVersion != latestVersion {
		t.Fatalf("expected latest version %q, got %q", latestVersion, status.LatestVersion)
	}
	if status.Source != "test/source" {
		t.Fatalf("expected source to round-trip, got %q", status.Source)
	}
	if status.LatestReleaseURL == "" {
		t.Fatal("expected release URL")
	}
}

func TestReleaseCheckerUsesCacheUntilTTLExpires(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = io.WriteString(w, sampleReleaseFeed(serverURL(r), core.Version(), "2026-03-28T19:44:23Z"))
	}))
	defer server.Close()

	now := time.Date(2026, 4, 3, 3, 0, 0, 0, time.UTC)
	checker := newReleaseChecker(server.Client(), server.URL, "test/source", 30*time.Minute)
	checker.now = func() time.Time { return now }

	first := checker.Check(context.Background(), false)
	second := checker.Check(context.Background(), false)

	if first.Status != "ok" || second.Status != "ok" {
		t.Fatalf("expected ok statuses, got %q and %q", first.Status, second.Status)
	}
	if requests != 1 {
		t.Fatalf("expected one request due to cache, got %d", requests)
	}
}

func TestReleaseCheckerReturnsStaleCacheWhenRefreshFails(t *testing.T) {
	t.Parallel()

	latestVersion := bumpedPatchVersion(t, core.Version())
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = io.WriteString(w, sampleReleaseFeed(serverURL(r), latestVersion, "2026-03-28T19:44:23Z"))
			return
		}
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	checker := newReleaseChecker(server.Client(), server.URL, "test/source", time.Hour)
	checker.now = func() time.Time { return time.Date(2026, 4, 3, 3, 0, 0, 0, time.UTC) }

	okStatus := checker.Check(context.Background(), true)
	if okStatus.Status != "ok" {
		t.Fatalf("expected first status ok, got %q", okStatus.Status)
	}

	staleStatus := checker.Check(context.Background(), true)
	if staleStatus.Status != "stale" {
		t.Fatalf("expected stale status, got %q", staleStatus.Status)
	}
	if !staleStatus.Stale {
		t.Fatal("expected stale flag to be true")
	}
	if staleStatus.LatestVersion != latestVersion {
		t.Fatalf("expected stale latest version %q, got %q", latestVersion, staleStatus.LatestVersion)
	}
	if !strings.Contains(staleStatus.Message, "latest check failed") {
		t.Fatalf("expected stale message to explain refresh failure, got %q", staleStatus.Message)
	}
}

func sampleReleaseFeed(feedURL, version, updated string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <updated>%s</updated>
    <link rel="alternate" type="text/html" href="%s/releases/tag/v%s"/>
    <title>Xray-core v%s</title>
  </entry>
</feed>`, updated, feedURL, version, version)
}

func bumpedPatchVersion(t *testing.T, version string) string {
	t.Helper()

	parts := splitVersionParts(version)
	if len(parts) == 0 {
		t.Fatalf("unexpected version format: %q", version)
	}
	parts[len(parts)-1]++
	stringParts := make([]string, 0, len(parts))
	for _, part := range parts {
		stringParts = append(stringParts, fmt.Sprintf("%d", part))
	}
	return strings.Join(stringParts, ".")
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
