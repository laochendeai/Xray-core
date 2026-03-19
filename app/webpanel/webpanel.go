package webpanel

import (
	"context"
	"crypto/tls"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	core "github.com/xtls/xray-core/core"
)

// WebPanel is the main feature that runs the web management panel.
type WebPanel struct {
	sync.Mutex
	config     *Config
	server     *http.Server
	listener   net.Listener
	grpcClient *GRPCClient
	auth       *AuthManager
	instance   *core.Instance
}

// NewWebPanel creates a new WebPanel instance from config.
func NewWebPanel(ctx context.Context, config *Config) (*WebPanel, error) {
	wp := &WebPanel{
		config: config,
	}

	s := core.FromContext(ctx)
	if s != nil {
		wp.instance = s
	}

	return wp, nil
}

func (wp *WebPanel) Type() interface{} {
	return (*WebPanel)(nil)
}

func (wp *WebPanel) Start() error {
	wp.Lock()
	defer wp.Unlock()

	listen := wp.config.Listen
	if listen == "" {
		listen = "127.0.0.1:9527"
	}

	apiEndpoint := wp.config.ApiEndpoint
	if apiEndpoint == "" {
		apiEndpoint = "127.0.0.1:10085"
	}

	username := wp.config.Username
	if username == "" {
		username = "admin"
	}

	password := wp.config.Password
	if password == "" {
		password = "admin123"
	}

	jwtSecret := wp.config.JwtSecret
	if jwtSecret == "" {
		jwtSecret = "xray-webpanel-secret"
	}

	// Initialize gRPC client
	var err error
	wp.grpcClient, err = NewGRPCClient(apiEndpoint)
	if err != nil {
		return errors.New("failed to create gRPC client").Base(err)
	}

	// Initialize auth manager
	wp.auth = NewAuthManager(username, password, jwtSecret)

	// Build HTTP mux
	mux := http.NewServeMux()
	wp.registerRoutes(mux)

	wp.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	l, err := net.Listen("tcp", listen)
	if err != nil {
		return errors.New("failed to listen on ", listen).Base(err)
	}
	wp.listener = l

	// Use TLS if configured
	if wp.config.CertFile != "" && wp.config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(wp.config.CertFile, wp.config.KeyFile)
		if err != nil {
			return errors.New("failed to load TLS certificate").Base(err)
		}
		wp.listener = tls.NewListener(l, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
	}

	errors.LogInfo(context.Background(), "Web panel listening on ", listen)

	go func() {
		if err := wp.server.Serve(wp.listener); err != nil && err != http.ErrServerClosed {
			errors.LogErrorInner(context.Background(), err, "failed to start web panel server")
		}
	}()

	return nil
}

func (wp *WebPanel) Close() error {
	wp.Lock()
	defer wp.Unlock()

	if wp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		wp.server.Shutdown(ctx)
		wp.server = nil
	}
	if wp.grpcClient != nil {
		wp.grpcClient.Close()
		wp.grpcClient = nil
	}
	return nil
}

func (wp *WebPanel) registerRoutes(mux *http.ServeMux) {
	// Auth
	mux.HandleFunc("/api/v1/auth/login", wp.handleLogin)

	// Stats
	mux.HandleFunc("/api/v1/sys/stats", wp.authMiddleware(wp.handleSysStats))
	mux.HandleFunc("/api/v1/stats/query", wp.authMiddleware(wp.handleQueryStats))
	mux.HandleFunc("/api/v1/stats/online-users", wp.authMiddleware(wp.handleOnlineUsers))
	mux.HandleFunc("/api/v1/stats/online-ips", wp.authMiddleware(wp.handleOnlineIPs))

	// Inbounds
	mux.HandleFunc("/api/v1/inbounds", wp.authMiddleware(wp.handleInbounds))
	mux.HandleFunc("/api/v1/inbounds/", wp.authMiddleware(wp.handleInboundByTag))

	// Users
	mux.HandleFunc("/api/v1/users/", wp.authMiddleware(wp.handleUsers))

	// Outbounds
	mux.HandleFunc("/api/v1/outbounds", wp.authMiddleware(wp.handleOutbounds))
	mux.HandleFunc("/api/v1/outbounds/", wp.authMiddleware(wp.handleOutboundByTag))

	// Routing
	mux.HandleFunc("/api/v1/routing/rules", wp.authMiddleware(wp.handleRoutingRules))
	mux.HandleFunc("/api/v1/routing/rules/", wp.authMiddleware(wp.handleRoutingRuleByTag))
	mux.HandleFunc("/api/v1/routing/test", wp.authMiddleware(wp.handleRoutingTest))
	mux.HandleFunc("/api/v1/routing/balancers/", wp.authMiddleware(wp.handleBalancers))

	// Observatory
	mux.HandleFunc("/api/v1/observatory/status", wp.authMiddleware(wp.handleObservatoryStatus))

	// Logger
	mux.HandleFunc("/api/v1/logger/restart", wp.authMiddleware(wp.handleLoggerRestart))

	// Config
	mux.HandleFunc("/api/v1/config", wp.authMiddleware(wp.handleConfig))
	mux.HandleFunc("/api/v1/config/reload", wp.authMiddleware(wp.handleConfigReload))
	mux.HandleFunc("/api/v1/config/validate", wp.authMiddleware(wp.handleConfigValidate))
	mux.HandleFunc("/api/v1/config/backups", wp.authMiddleware(wp.handleConfigBackups))

	// Share link
	mux.HandleFunc("/api/v1/share/generate", wp.authMiddleware(wp.handleShareGenerate))

	// Subscription (public endpoint)
	mux.HandleFunc("/sub/", wp.handleSubscription)

	// WebSocket endpoints
	mux.HandleFunc("/api/v1/ws/routing-stats", wp.authMiddleware(wp.handleWSRoutingStats))
	mux.HandleFunc("/api/v1/ws/traffic", wp.authMiddleware(wp.handleWSTraffic))

	// Static files (Vue SPA)
	mux.HandleFunc("/", wp.handleStaticFiles)
}

func (wp *WebPanel) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Serve embedded frontend files
	distFS, err := fs.Sub(distFiles, "dist")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Try to serve the file directly
	f, err := distFS.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		// For SPA routing, serve index.html for non-API, non-file paths
		path = "/index.html"
	} else {
		f.Close()
	}

	fileServer := http.FileServer(http.FS(distFS))
	if path != r.URL.Path {
		r.URL.Path = path
	}
	fileServer.ServeHTTP(w, r)
}

func (wp *WebPanel) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Skip auth for WebSocket upgrade (auth is checked via query param)
		if r.Header.Get("Upgrade") == "websocket" {
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if _, err := wp.auth.ValidateToken(token); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := wp.auth.ValidateToken(token); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		return NewWebPanel(ctx, cfg.(*Config))
	}))
}
