package webpanel

import (
	"context"
	"fmt"
	"io"
	stdnet "net"
	"net/http"
	"strings"
	"time"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/utils"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport/internet/tagged"
)

type NodeExitIPStatus string

const (
	NodeExitIPStatusUnknown   NodeExitIPStatus = "unknown"
	NodeExitIPStatusAvailable NodeExitIPStatus = "available"
	NodeExitIPStatusError     NodeExitIPStatus = "error"
)

type nodeExitIPProbeResult struct {
	IP        string
	Source    string
	CheckedAt time.Time
	Error     string
}

type nodeExitIPProbeFunc func(ctx context.Context, dispatcher routing.Dispatcher, tag string) nodeExitIPProbeResult

const nodeExitIPProbeTimeout = 5 * time.Second

var nodeExitIPProbeEndpoints = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://ifconfig.me/ip",
}

func defaultNodeExitIPProber(ctx context.Context, dispatcher routing.Dispatcher, tag string) nodeExitIPProbeResult {
	if ctx == nil {
		ctx = context.Background()
	}

	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(reqCtx context.Context, network, addr string) (xnet.Conn, error) {
			dest, err := xnet.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			return tagged.Dialer(ctx, dispatcher, dest, tag)
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   nodeExitIPProbeTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	lastError := ""
	lastSource := ""
	for _, endpoint := range nodeExitIPProbeEndpoints {
		lastSource = endpoint
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastError = err.Error()
			continue
		}
		req.Header.Set("User-Agent", utils.ChromeUA)

		resp, err := client.Do(req)
		if err != nil {
			lastError = err.Error()
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		if readErr != nil {
			lastError = readErr.Error()
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastError = fmt.Sprintf("%s returned HTTP %d", endpoint, resp.StatusCode)
			continue
		}

		candidate := strings.TrimSpace(string(body))
		ip := stdnet.ParseIP(candidate)
		if ip == nil {
			lastError = fmt.Sprintf("%s returned a non-IP body", endpoint)
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			candidate = ip4.String()
		} else {
			candidate = ip.String()
		}

		return nodeExitIPProbeResult{
			IP:        candidate,
			Source:    endpoint,
			CheckedAt: time.Now().UTC(),
		}
	}

	if lastError == "" {
		lastError = "all exit-IP probe endpoints failed"
	}
	return nodeExitIPProbeResult{
		Source:    lastSource,
		CheckedAt: time.Now().UTC(),
		Error:     lastError,
	}
}
