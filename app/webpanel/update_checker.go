package webpanel

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	core "github.com/xtls/xray-core/core"
)

const (
	defaultReleaseFeedURL = "https://github.com/XTLS/Xray-core/releases.atom"
	defaultReleaseSource  = "XTLS/Xray-core"
)

var releaseVersionPattern = regexp.MustCompile(`v?\d+(?:\.\d+)+`)

type UpdateStatusResponse struct {
	CurrentVersion    string `json:"currentVersion"`
	LatestVersion     string `json:"latestVersion,omitempty"`
	ReleaseTitle      string `json:"releaseTitle,omitempty"`
	LatestReleaseURL  string `json:"latestReleaseUrl,omitempty"`
	LatestPublishedAt string `json:"latestPublishedAt,omitempty"`
	CheckedAt         string `json:"checkedAt,omitempty"`
	Source            string `json:"source"`
	Status            string `json:"status"`
	Message           string `json:"message,omitempty"`
	UpdateAvailable   bool   `json:"updateAvailable"`
	Stale             bool   `json:"stale"`
}

type releaseChecker struct {
	mu       sync.Mutex
	client   *http.Client
	feedURL  string
	source   string
	cacheTTL time.Duration
	now      func() time.Time
	cached   *UpdateStatusResponse
	cachedAt time.Time
}

func newReleaseChecker(client *http.Client, feedURL, source string, cacheTTL time.Duration) *releaseChecker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if feedURL == "" {
		feedURL = defaultReleaseFeedURL
	}
	if source == "" {
		source = defaultReleaseSource
	}
	if cacheTTL <= 0 {
		cacheTTL = 30 * time.Minute
	}
	return &releaseChecker{
		client:   client,
		feedURL:  feedURL,
		source:   source,
		cacheTTL: cacheTTL,
		now:      time.Now,
	}
}

func (c *releaseChecker) Check(ctx context.Context, force bool) UpdateStatusResponse {
	c.mu.Lock()
	if !force && c.cached != nil && c.now().Sub(c.cachedAt) < c.cacheTTL {
		cached := *c.cached
		c.mu.Unlock()
		cached.CurrentVersion = core.Version()
		return cached
	}
	c.mu.Unlock()

	status, err := c.fetch(ctx)
	if err == nil {
		c.mu.Lock()
		c.cached = &status
		c.cachedAt = c.now()
		c.mu.Unlock()
		return status
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil {
		stale := *c.cached
		stale.CurrentVersion = core.Version()
		stale.Status = "stale"
		stale.Stale = true
		stale.Message = fmt.Sprintf("Showing cached release info because the latest check failed: %v", err)
		return stale
	}

	return UpdateStatusResponse{
		CurrentVersion: core.Version(),
		Source:         c.source,
		Status:         "error",
		Message:        fmt.Sprintf("Failed to check latest release: %v", err),
		Stale:          false,
	}
}

func (c *releaseChecker) CachedStatus() (UpdateStatusResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached == nil {
		return UpdateStatusResponse{
			CurrentVersion:  core.Version(),
			Source:          c.source,
			Status:          "unavailable",
			Message:         "Release status has not been checked yet. Use Check Updates on the dashboard to refresh it.",
			UpdateAvailable: false,
			Stale:           false,
		}, false
	}

	cached := *c.cached
	cached.CurrentVersion = core.Version()
	if c.now().Sub(c.cachedAt) >= c.cacheTTL {
		cached.Status = "stale"
		cached.Stale = true
		if strings.TrimSpace(cached.Message) == "" {
			cached.Message = "Cached release info is older than the refresh interval. Use Check Updates on the dashboard to refresh it."
		}
	}
	return cached, true
}

func (c *releaseChecker) fetch(ctx context.Context) (UpdateStatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.feedURL, nil)
	if err != nil {
		return UpdateStatusResponse{}, err
	}
	req.Header.Set("User-Agent", "xray-webpanel/"+core.Version())
	req.Header.Set("Accept", "application/atom+xml, application/xml;q=0.9, */*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return UpdateStatusResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UpdateStatusResponse{}, fmt.Errorf("release feed returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return UpdateStatusResponse{}, err
	}

	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("parse release feed: %w", err)
	}
	if len(feed.Entries) == 0 {
		return UpdateStatusResponse{}, fmt.Errorf("release feed is empty")
	}

	entry := feed.Entries[0]
	releaseURL := entry.alternateURL()
	latestVersion := extractReleaseVersion(entry.Title, releaseURL)
	if latestVersion == "" {
		return UpdateStatusResponse{}, fmt.Errorf("could not determine latest version from release feed")
	}

	latestPublishedAt := entry.Updated
	if _, err := time.Parse(time.RFC3339, latestPublishedAt); err != nil {
		latestPublishedAt = ""
	}

	return UpdateStatusResponse{
		CurrentVersion:    core.Version(),
		LatestVersion:     latestVersion,
		ReleaseTitle:      strings.TrimSpace(entry.Title),
		LatestReleaseURL:  releaseURL,
		LatestPublishedAt: latestPublishedAt,
		CheckedAt:         c.now().UTC().Format(time.RFC3339),
		Source:            c.source,
		Status:            "ok",
		UpdateAvailable:   compareVersionStrings(latestVersion, core.Version()) > 0,
		Stale:             false,
	}, nil
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Updated string     `xml:"updated"`
	Links   []atomLink `xml:"link"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func (e atomEntry) alternateURL() string {
	for _, link := range e.Links {
		if link.Rel == "alternate" && strings.TrimSpace(link.Href) != "" {
			return strings.TrimSpace(link.Href)
		}
	}
	for _, link := range e.Links {
		if strings.TrimSpace(link.Href) != "" {
			return strings.TrimSpace(link.Href)
		}
	}
	return ""
}

func extractReleaseVersion(title, releaseURL string) string {
	if releaseURL != "" {
		if parsed, err := url.Parse(releaseURL); err == nil {
			if tag := strings.TrimSpace(path.Base(parsed.Path)); tag != "" {
				if version := normalizeVersion(tag); version != "" {
					return version
				}
			}
		}
	}

	match := releaseVersionPattern.FindString(strings.TrimSpace(title))
	return normalizeVersion(match)
}

func compareVersionStrings(left, right string) int {
	left = normalizeVersion(left)
	right = normalizeVersion(right)
	leftParts := splitVersionParts(left)
	rightParts := splitVersionParts(right)

	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for i := 0; i < maxLen; i++ {
		lv := versionPartAt(leftParts, i)
		rv := versionPartAt(rightParts, i)
		switch {
		case lv > rv:
			return 1
		case lv < rv:
			return -1
		}
	}

	return 0
}

func normalizeVersion(raw string) string {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if raw == "" || !releaseVersionPattern.MatchString(raw) {
		return ""
	}
	return raw
}

func splitVersionParts(version string) []int {
	if version == "" {
		return nil
	}
	parts := strings.Split(version, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		values = append(values, value)
	}
	return values
}

func versionPartAt(parts []int, index int) int {
	if index >= 0 && index < len(parts) {
		return parts[index]
	}
	return 0
}
