package webpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type NodeNetworkType string
type NodeIntelligenceConfidence string

const (
	NodeNetworkTypeUnknown            NodeNetworkType            = "unknown"
	NodeNetworkTypeResidentialLikely  NodeNetworkType            = "residential_likely"
	NodeNetworkTypeISPLikely          NodeNetworkType            = "isp_likely"
	NodeNetworkTypeDatacenterLikely   NodeNetworkType            = "datacenter_likely"
	NodeIntelligenceConfidenceUnknown NodeIntelligenceConfidence = "unknown"
	NodeIntelligenceConfidenceLow     NodeIntelligenceConfidence = "low"
	NodeIntelligenceConfidenceMedium  NodeIntelligenceConfidence = "medium"
	NodeIntelligenceConfidenceHigh    NodeIntelligenceConfidence = "high"
)

const (
	nodeIntelligenceReasonExitIPUnavailable     = "exit_ip_unavailable"
	nodeIntelligenceReasonLookupFailed          = "lookup_failed"
	nodeIntelligenceReasonHostingKeywordMatch   = "hosting_keyword_match"
	nodeIntelligenceReasonISPKeywordMatch       = "isp_keyword_match"
	nodeIntelligenceReasonResidentialKeyword    = "residential_keyword_match"
	nodeIntelligenceReasonProbeFailureRateHigh  = "probe_failure_rate_high"
	nodeIntelligenceReasonResidentialStableExit = "residential_stable_exit"
	nodeIntelligenceReasonDatacenterExit        = "datacenter_exit_network"
	nodeIntelligenceReasonInsufficientSignal    = "insufficient_signal"
)

type nodeIPConnectionLookupResult struct {
	ASN       int
	Org       string
	ISP       string
	Domain    string
	CheckedAt time.Time
	Error     string
}

type nodeIPConnectionLookupFunc func(ctx context.Context, ip string) nodeIPConnectionLookupResult

type nodeIntelligenceResult struct {
	ExitIP                string
	Cleanliness           CleanlinessStatus
	CleanlinessConfidence NodeIntelligenceConfidence
	CleanlinessReason     string
	CleanlinessDetail     string
	NetworkType           NodeNetworkType
	NetworkConfidence     NodeIntelligenceConfidence
	NetworkReason         string
	NetworkDetail         string
	CheckedAt             time.Time
	Error                 string
}

const nodeIntelligenceLookupTimeout = 5 * time.Second

var hostingProviderKeywords = []string{
	"amazon", "amazonaws", "aws", "azure", "microsoft azure", "microsoft corporation",
	"digitalocean", "digital ocean", "linode", "akamai", "google cloud", "google llc", "gcp",
	"vultr", "hetzner", "ovh", "ovhcloud", "oracle cloud", "cloudflare",
	"alibaba cloud", "tencent cloud", "choopa", "leaseweb", "scaleway", "contabo",
	"racknerd", "pq.hosting", "hostinger", "greenhost", "cloud", "hosting", "colo", "vps",
	"server", "datacenter", "compute",
}

var residentialProviderKeywords = []string{
	"xfinity", "comcast", "spectrum", "charter", "verizon fios", "centurylink",
	"cox", "rogers", "shaw", "telus", "virgin media",
}

var ispProviderKeywords = []string{
	"telecom", "communications", "broadband", "wireless", "mobile", "fiber", "fibre",
	"cable", "internet service", "internet services", "isp", "verizon", "at&t", "att ",
	"telefonica", "vodafone", "orange", "china telecom", "china unicom", "china mobile",
	"bt",
}

func hasWholeWord(corpus string, word string) bool {
	for _, token := range strings.FieldsFunc(corpus, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= '0' && r <= '9':
			return false
		default:
			return true
		}
	}) {
		if token == word {
			return true
		}
	}
	return false
}

func firstMatchingKeyword(corpus string, keywords []string) string {
	if strings.TrimSpace(corpus) == "" {
		return ""
	}
	for _, keyword := range keywords {
		if strings.Contains(keyword, " ") {
			if strings.Contains(corpus, keyword) {
				return keyword
			}
			continue
		}
		if hasWholeWord(corpus, keyword) {
			return keyword
		}
	}
	return ""
}

func defaultNodeIPConnectionLookup(ctx context.Context, ip string) nodeIPConnectionLookupResult {
	if ctx == nil {
		ctx = context.Background()
	}

	client := &http.Client{Timeout: nodeIntelligenceLookupTimeout}
	endpoint := fmt.Sprintf("https://ipwho.is/%s?fields=ip,success,connection,message", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     err.Error(),
		}
	}
	req.Header.Set("User-Agent", "xray-webpanel-node-intelligence/1")

	resp, err := client.Do(req)
	if err != nil {
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     err.Error(),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     fmt.Sprintf("ipwho.is returned HTTP %d", resp.StatusCode),
		}
	}

	var decoded struct {
		Success    bool   `json:"success"`
		Message    string `json:"message"`
		Connection struct {
			ASN    int    `json:"asn"`
			Org    string `json:"org"`
			ISP    string `json:"isp"`
			Domain string `json:"domain"`
		} `json:"connection"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     err.Error(),
		}
	}
	if !decoded.Success {
		message := strings.TrimSpace(decoded.Message)
		if message == "" {
			message = "ipwho.is returned success=false"
		}
		return nodeIPConnectionLookupResult{
			CheckedAt: time.Now().UTC(),
			Error:     message,
		}
	}

	return nodeIPConnectionLookupResult{
		ASN:       decoded.Connection.ASN,
		Org:       decoded.Connection.Org,
		ISP:       decoded.Connection.ISP,
		Domain:    decoded.Connection.Domain,
		CheckedAt: time.Now().UTC(),
	}
}

func classifyNodeIntelligence(node NodeRecord, cfg ValidationConfig, lookup nodeIPConnectionLookupResult) nodeIntelligenceResult {
	checkedAt := lookup.CheckedAt
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}

	result := nodeIntelligenceResult{
		ExitIP:                node.ExitIP,
		Cleanliness:           CleanlinessUnknown,
		CleanlinessConfidence: NodeIntelligenceConfidenceUnknown,
		CleanlinessReason:     nodeIntelligenceReasonInsufficientSignal,
		CleanlinessDetail:     "no high-confidence cleanliness signal was available",
		NetworkType:           NodeNetworkTypeUnknown,
		NetworkConfidence:     NodeIntelligenceConfidenceUnknown,
		NetworkReason:         nodeIntelligenceReasonInsufficientSignal,
		NetworkDetail:         "connection metadata did not match a high-confidence access or hosting pattern",
		CheckedAt:             checkedAt,
		Error:                 lookup.Error,
	}

	if node.ExitIPStatus != NodeExitIPStatusAvailable || strings.TrimSpace(node.ExitIP) == "" {
		result.CleanlinessReason = nodeIntelligenceReasonExitIPUnavailable
		result.CleanlinessDetail = "the node does not currently have a usable exit IP result"
		result.NetworkReason = nodeIntelligenceReasonExitIPUnavailable
		result.NetworkDetail = "the node does not currently have a usable exit IP result"
		return result
	}

	result.NetworkType, result.NetworkConfidence, result.NetworkReason, result.NetworkDetail = classifyNodeNetworkType(lookup)

	if isNodeProbeFailureSuspicious(node, cfg) {
		result.Cleanliness = CleanlinessSuspicious
		result.CleanlinessConfidence = NodeIntelligenceConfidenceMedium
		result.CleanlinessReason = nodeIntelligenceReasonProbeFailureRateHigh
		result.CleanlinessDetail = describeNodeProbeFailure(node, cfg)
		return result
	}

	if lookup.Error != "" {
		result.CleanlinessReason = nodeIntelligenceReasonLookupFailed
		result.CleanlinessDetail = "the exit IP lookup failed before a higher-confidence cleanliness verdict could be assigned"
		return result
	}

	switch result.NetworkType {
	case NodeNetworkTypeDatacenterLikely:
		result.Cleanliness = CleanlinessSuspicious
		result.CleanlinessConfidence = maxNodeConfidence(result.NetworkConfidence, NodeIntelligenceConfidenceMedium)
		result.CleanlinessReason = nodeIntelligenceReasonDatacenterExit
		result.CleanlinessDetail = "the exit network looks like a hosting or cloud provider instead of an access ISP"
	case NodeNetworkTypeResidentialLikely:
		if isNodeProbeStable(node, cfg) {
			result.Cleanliness = CleanlinessTrusted
			result.CleanlinessConfidence = result.NetworkConfidence
			result.CleanlinessReason = nodeIntelligenceReasonResidentialStableExit
			result.CleanlinessDetail = "the exit network looks like a residential broadband ISP and the node's probe history is stable"
			return result
		}
		result.CleanlinessReason = nodeIntelligenceReasonInsufficientSignal
		result.CleanlinessDetail = "the exit network looks residential, but probe history is still too weak to promote cleanliness confidence"
	case NodeNetworkTypeISPLikely:
		result.CleanlinessReason = nodeIntelligenceReasonInsufficientSignal
		result.CleanlinessDetail = "the exit network looks like an access ISP, but not strongly enough to treat it like residential"
	default:
		result.CleanlinessReason = nodeIntelligenceReasonInsufficientSignal
		result.CleanlinessDetail = "the node did not produce enough network-identity signal for a reliable cleanliness verdict"
	}

	return result
}

func classifyNodeNetworkType(lookup nodeIPConnectionLookupResult) (NodeNetworkType, NodeIntelligenceConfidence, string, string) {
	if lookup.Error != "" {
		return NodeNetworkTypeUnknown, NodeIntelligenceConfidenceUnknown, nodeIntelligenceReasonLookupFailed, "the exit IP lookup failed before a network type could be inferred"
	}

	corpusParts := []string{
		strings.ToLower(strings.TrimSpace(lookup.Org)),
		strings.ToLower(strings.TrimSpace(lookup.ISP)),
		strings.ToLower(strings.TrimSpace(lookup.Domain)),
	}
	corpus := strings.Join(corpusParts, " ")

	if keyword := firstMatchingKeyword(corpus, hostingProviderKeywords); keyword != "" {
		return NodeNetworkTypeDatacenterLikely, NodeIntelligenceConfidenceHigh, nodeIntelligenceReasonHostingKeywordMatch, fmt.Sprintf("connection metadata matched hosting keyword %q (org=%q isp=%q domain=%q)", keyword, lookup.Org, lookup.ISP, lookup.Domain)
	}
	if keyword := firstMatchingKeyword(corpus, residentialProviderKeywords); keyword != "" {
		return NodeNetworkTypeResidentialLikely, NodeIntelligenceConfidenceMedium, nodeIntelligenceReasonResidentialKeyword, fmt.Sprintf("connection metadata matched residential keyword %q (org=%q isp=%q domain=%q)", keyword, lookup.Org, lookup.ISP, lookup.Domain)
	}
	if keyword := firstMatchingKeyword(corpus, ispProviderKeywords); keyword != "" {
		return NodeNetworkTypeISPLikely, NodeIntelligenceConfidenceMedium, nodeIntelligenceReasonISPKeywordMatch, fmt.Sprintf("connection metadata matched ISP keyword %q (org=%q isp=%q domain=%q)", keyword, lookup.Org, lookup.ISP, lookup.Domain)
	}

	return NodeNetworkTypeUnknown, NodeIntelligenceConfidenceLow, nodeIntelligenceReasonInsufficientSignal, fmt.Sprintf("connection metadata stayed inconclusive (org=%q isp=%q domain=%q asn=%d)", lookup.Org, lookup.ISP, lookup.Domain, lookup.ASN)
}

func isNodeProbeFailureSuspicious(node NodeRecord, cfg ValidationConfig) bool {
	if node.TotalPings < cfg.MinSamples {
		return false
	}
	if node.ConsecutiveFails >= cfg.DemoteAfterFails && cfg.DemoteAfterFails > 0 {
		return true
	}
	return nodeProbeFailRate(node) >= 0.5
}

func isNodeProbeStable(node NodeRecord, cfg ValidationConfig) bool {
	if node.TotalPings < cfg.MinSamples || node.AvgDelayMs <= 0 {
		return false
	}
	if nodeProbeFailRate(node) > cfg.MaxFailRate {
		return false
	}
	if cfg.MaxAvgDelayMs > 0 && node.AvgDelayMs > cfg.MaxAvgDelayMs {
		return false
	}
	return node.ConsecutiveFails == 0
}

func describeNodeProbeFailure(node NodeRecord, cfg ValidationConfig) string {
	failRate := nodeProbeFailRate(node)
	if node.ConsecutiveFails >= cfg.DemoteAfterFails && cfg.DemoteAfterFails > 0 {
		return fmt.Sprintf("consecutive probe failures reached %d with threshold %d", node.ConsecutiveFails, cfg.DemoteAfterFails)
	}
	return fmt.Sprintf("probe failure rate %.2f exceeded the suspicious threshold after %d samples", failRate, node.TotalPings)
}

func nodeProbeFailRate(node NodeRecord) float64 {
	if node.TotalPings <= 0 {
		return 0
	}
	return float64(node.FailedPings) / float64(node.TotalPings)
}

func maxNodeConfidence(values ...NodeIntelligenceConfidence) NodeIntelligenceConfidence {
	best := NodeIntelligenceConfidenceUnknown
	bestRank := -1
	for _, value := range values {
		rank := nodeConfidenceRank(value)
		if rank > bestRank {
			best = value
			bestRank = rank
		}
	}
	return best
}

func nodeConfidenceRank(value NodeIntelligenceConfidence) int {
	switch value {
	case NodeIntelligenceConfidenceHigh:
		return 3
	case NodeIntelligenceConfidenceMedium:
		return 2
	case NodeIntelligenceConfidenceLow:
		return 1
	default:
		return 0
	}
}
