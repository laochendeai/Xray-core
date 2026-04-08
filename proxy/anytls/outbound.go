package anytls

import (
	"context"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/common/singbridge"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))
}

type Outbound struct {
	address  string
	port     uint32
	password string
}

func New(_ context.Context, config *Config) (*Outbound, error) {
	return &Outbound{
		address:  config.GetAddress(),
		port:     config.GetPort(),
		password: config.GetPassword(),
	}, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	var inboundConn net.Conn
	if inbound := session.InboundFromContext(ctx); inbound != nil {
		inboundConn = inbound.Conn
	}

	outbounds := session.OutboundsFromContext(ctx)
	if len(outbounds) == 0 {
		return errors.New("target not specified")
	}

	ob := outbounds[len(outbounds)-1]
	if !ob.Target.IsValid() {
		return errors.New("target not specified")
	}
	if ob.Target.Network != net.Network_TCP {
		return errors.New("AnyTLS outbound does not support UDP targets yet")
	}
	ob.Name = "anytls"
	destination := ob.Target

	server := net.TCPDestination(net.ParseAddress(o.address), net.Port(o.port))
	errors.LogInfo(ctx, "tunneling request to ", destination, " via ", server.NetAddr())

	connection, err := dialer.Dial(ctx, server)
	if err != nil {
		return errors.New("failed to connect to AnyTLS server").Base(err)
	}

	if session.TimeoutOnlyFromContext(ctx) {
		ctx, _ = context.WithCancel(context.Background())
	}

	serverConn, err := newClientStreamConn(connection, o.password, destination)
	if err != nil {
		_ = connection.Close()
		return errors.New("failed to establish AnyTLS session").Base(err)
	}

	return singbridge.CopyConn(ctx, inboundConn, link, serverConn)
}
