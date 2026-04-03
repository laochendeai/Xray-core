package webpanel

import (
	"context"
	"net/http"
	"sync"
	"time"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/utils"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport/internet/tagged"
)

// ProbeResult holds the result of a single node probe.
type ProbeResult struct {
	Tag     string
	Success bool
	DelayMs int64
}

// ProbeCallback is called after each probe round with results.
type ProbeCallback func(results []ProbeResult)

// NodeProberSnapshot is a read-only snapshot used by diagnostics surfaces.
type NodeProberSnapshot struct {
	Running     bool   `json:"running"`
	ProbeURL    string `json:"probeUrl"`
	IntervalSec int    `json:"intervalSec"`
	TagCount    int    `json:"tagCount"`
}

// NodeProber performs periodic health checks on nodes via their outbound tags.
type NodeProber struct {
	mu         sync.RWMutex
	baseCtx    context.Context
	ctx        context.Context
	cancel     context.CancelFunc
	dispatcher routing.Dispatcher
	probeURL   string
	interval   time.Duration
	timeout    time.Duration
	tags       map[string]struct{} // tags to probe
	callback   ProbeCallback
	running    bool
}

// NewNodeProber creates a new NodeProber.
func NewNodeProber(baseCtx context.Context, dispatcher routing.Dispatcher, probeURL string, intervalSec int, callback ProbeCallback) *NodeProber {
	if probeURL == "" {
		probeURL = "https://www.gstatic.com/generate_204"
	}
	if intervalSec <= 0 {
		intervalSec = 60
	}
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	return &NodeProber{
		baseCtx:    baseCtx,
		dispatcher: dispatcher,
		probeURL:   probeURL,
		interval:   time.Duration(intervalSec) * time.Second,
		timeout:    5 * time.Second,
		tags:       make(map[string]struct{}),
		callback:   callback,
	}
}

// Start begins the periodic probing loop.
func (p *NodeProber) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.ctx, p.cancel = context.WithCancel(p.baseCtx)
	p.running = true
	p.mu.Unlock()

	go p.loop()
}

// Stop stops the probing loop.
func (p *NodeProber) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	p.running = false
}

// UpdateConfig updates probe URL and interval.
func (p *NodeProber) UpdateConfig(probeURL string, intervalSec int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if probeURL != "" {
		p.probeURL = probeURL
	}
	if intervalSec > 0 {
		p.interval = time.Duration(intervalSec) * time.Second
	}
}

// AddTag adds a tag to be probed.
func (p *NodeProber) AddTag(tag string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tags[tag] = struct{}{}
}

// RemoveTag removes a tag from probing.
func (p *NodeProber) RemoveTag(tag string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.tags, tag)
}

// GetTags returns all tags being probed.
func (p *NodeProber) GetTags() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tags := make([]string, 0, len(p.tags))
	for t := range p.tags {
		tags = append(tags, t)
	}
	return tags
}

// Snapshot returns the current diagnostics state of the prober.
func (p *NodeProber) Snapshot() NodeProberSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	intervalSec := 0
	if p.interval > 0 {
		intervalSec = int(p.interval / time.Second)
	}

	return NodeProberSnapshot{
		Running:     p.running,
		ProbeURL:    p.probeURL,
		IntervalSec: intervalSec,
		TagCount:    len(p.tags),
	}
}

func (p *NodeProber) loop() {
	// Initial probe after a short delay
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-timer.C:
			p.probeAll()
			p.mu.RLock()
			interval := p.interval
			p.mu.RUnlock()
			timer.Reset(interval)
		}
	}
}

func (p *NodeProber) probeAll() {
	p.mu.RLock()
	tags := make([]string, 0, len(p.tags))
	for t := range p.tags {
		tags = append(tags, t)
	}
	probeURL := p.probeURL
	p.mu.RUnlock()

	if len(tags) == 0 {
		return
	}

	results := make([]ProbeResult, len(tags))
	var wg sync.WaitGroup

	for i, tag := range tags {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			delayMs, err := p.probeNode(t, probeURL)
			results[idx] = ProbeResult{
				Tag:     t,
				Success: err == nil,
				DelayMs: delayMs,
			}
		}(i, tag)
	}

	wg.Wait()

	if p.callback != nil {
		p.callback(results)
	}
}

// probeNode sends an HTTP request through the specified outbound tag and measures delay.
func (p *NodeProber) probeNode(tag string, probeURL string) (int64, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (xnet.Conn, error) {
			dest, err := xnet.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			return tagged.Dialer(p.ctx, p.dispatcher, dest, tag)
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   p.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodHead, probeURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", utils.ChromeUA)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		errors.LogDebug(context.Background(), "probe failed for tag ", tag, ": ", err.Error())
		return 0, err
	}
	resp.Body.Close()

	return time.Since(start).Milliseconds(), nil
}
