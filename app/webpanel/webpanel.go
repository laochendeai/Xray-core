package webpanel

import (
	"context"
	"crypto/tls"
	"io"
	"io/fs"
	"net"
	"net/http"
	"runtime/debug"
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
	config         *Config
	server         *http.Server
	listener       net.Listener
	grpcClient     *GRPCClient
	auth           *AuthManager
	instance       *core.Instance
	runtimeCtx     context.Context
	subManager     *SubscriptionManager
	tunManager     *TunManager
	controlPlane   *ControlPlaneStateStore
	releaseChecker *releaseChecker
}

// NewWebPanel creates a new WebPanel instance from config.
func NewWebPanel(ctx context.Context, config *Config) (*WebPanel, error) {
	wp := &WebPanel{
		config:         config,
		releaseChecker: newReleaseChecker(nil, defaultReleaseFeedURL, defaultReleaseSource, 30*time.Minute),
	}

	s := core.FromContext(ctx)
	if s != nil {
		wp.instance = s
		wp.runtimeCtx = ctx
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

	// Initialize subscription manager if configPath is available
	if wp.config.ConfigPath != "" {
		wp.subManager = NewSubscriptionManager(wp.config.ConfigPath, wp.grpcClient, wp.instance, wp.runtimeCtx)
		wp.tunManager, _ = NewTunManager(wp.config.ConfigPath)
		wp.controlPlane = NewControlPlaneStateStore(wp.config.ConfigPath)
		wp.subManager.SetPoolHealthCallback(wp.handlePoolHealthChange)
	}

	// Build HTTP mux
	mux := http.NewServeMux()
	wp.registerRoutes(mux)

	wp.server = &http.Server{
		Handler:      wp.recoverMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
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

	// Start subscription manager
	if wp.subManager != nil {
		if err := wp.subManager.Start(); err != nil {
			errors.LogWarning(context.Background(), "failed to start subscription manager: ", err.Error())
		}
	}
	if wp.controlPlane != nil {
		wp.ensureCleanStartupState()
	}

	return nil
}

func (wp *WebPanel) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				errors.LogError(context.Background(), "web panel handler panic: ", recovered, "\n", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (wp *WebPanel) Close() error {
	wp.Lock()
	defer wp.Unlock()

	if wp.subManager != nil {
		wp.subManager.Stop()
		wp.subManager = nil
	}
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
	mux.HandleFunc("/api/v1/sys/update", wp.authMiddleware(wp.handleUpdateStatus))
	mux.HandleFunc("/api/v1/readiness", wp.authMiddleware(wp.handleReadiness))
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

	// Transparent TUN
	mux.HandleFunc("/api/v1/tun/status", wp.authMiddleware(wp.handleTunStatus))
	mux.HandleFunc("/api/v1/tun/settings", wp.authMiddleware(wp.handleTunSettings))
	mux.HandleFunc("/api/v1/tun/start", wp.authMiddleware(wp.handleTunStart))
	mux.HandleFunc("/api/v1/tun/stop", wp.authMiddleware(wp.handleTunStop))
	mux.HandleFunc("/api/v1/tun/restore-clean", wp.authMiddleware(wp.handleTunRestoreClean))
	mux.HandleFunc("/api/v1/tun/toggle", wp.authMiddleware(wp.handleTunToggle))
	mux.HandleFunc("/api/v1/tun/install-privilege", wp.authMiddleware(wp.handleTunInstallPrivilege))

	// Share link
	mux.HandleFunc("/api/v1/share/generate", wp.authMiddleware(wp.handleShareGenerate))

	// Subscriptions & Node Pool
	mux.HandleFunc("/api/v1/subscriptions", wp.authMiddleware(wp.handleSubscriptions))
	mux.HandleFunc("/api/v1/subscriptions/", wp.authMiddleware(wp.handleSubscriptionByID))
	mux.HandleFunc("/api/v1/node-pool", wp.authMiddleware(wp.handleNodePool))
	mux.HandleFunc("/api/v1/node-pool/", wp.authMiddleware(wp.handleNodePoolByID))

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

	if path == "/index.html" {
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()

		data, err := io.ReadAll(indexFile)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
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
