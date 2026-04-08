package anytls

import (
	"context"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))
}

type Outbound struct {
	address string
	port    uint32
}

func New(_ context.Context, config *Config) (*Outbound, error) {
	return &Outbound{
		address: config.GetAddress(),
		port:    config.GetPort(),
	}, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	_ = link
	_ = dialer

	outbounds := session.OutboundsFromContext(ctx)
	if len(outbounds) > 0 {
		outbounds[len(outbounds)-1].Name = "anytls"
	}

	return errors.New("AnyTLS outbound dial path is not implemented yet for ", o.address, ":", o.port).AtInfo()
}
