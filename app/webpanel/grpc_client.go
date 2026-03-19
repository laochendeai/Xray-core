package webpanel

import (
	"context"
	"sync"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	routerservice "github.com/xtls/xray-core/app/router/command"
	statsservice "github.com/xtls/xray-core/app/stats/command"
	loggerservice "github.com/xtls/xray-core/app/log/command"
	observatoryservice "github.com/xtls/xray-core/app/observatory/command"
	"github.com/xtls/xray-core/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCClient manages the gRPC connection to the Xray Commander API.
type GRPCClient struct {
	conn        *grpc.ClientConn
	endpoint    string
	mu          sync.Mutex

	statsClient       statsservice.StatsServiceClient
	handlerClient     handlerservice.HandlerServiceClient
	routingClient     routerservice.RoutingServiceClient
	loggerClient      loggerservice.LoggerServiceClient
	observatoryClient observatoryservice.ObservatoryServiceClient
}

// NewGRPCClient creates a new gRPC client connection.
func NewGRPCClient(endpoint string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.New("failed to connect to gRPC endpoint ", endpoint).Base(err)
	}

	return &GRPCClient{
		conn:              conn,
		endpoint:          endpoint,
		statsClient:       statsservice.NewStatsServiceClient(conn),
		handlerClient:     handlerservice.NewHandlerServiceClient(conn),
		routingClient:     routerservice.NewRoutingServiceClient(conn),
		loggerClient:      loggerservice.NewLoggerServiceClient(conn),
		observatoryClient: observatoryservice.NewObservatoryServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *GRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Stats returns the stats service client.
func (c *GRPCClient) Stats() statsservice.StatsServiceClient {
	return c.statsClient
}

// Handler returns the handler service client.
func (c *GRPCClient) Handler() handlerservice.HandlerServiceClient {
	return c.handlerClient
}

// Routing returns the routing service client.
func (c *GRPCClient) Routing() routerservice.RoutingServiceClient {
	return c.routingClient
}

// Logger returns the logger service client.
func (c *GRPCClient) Logger() loggerservice.LoggerServiceClient {
	return c.loggerClient
}

// Observatory returns the observatory service client.
func (c *GRPCClient) Observatory() observatoryservice.ObservatoryServiceClient {
	return c.observatoryClient
}

// Context returns a background context for gRPC calls.
func (c *GRPCClient) Context() context.Context {
	return context.Background()
}
