package conf

import (
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/proxy/anytls"
	"google.golang.org/protobuf/proto"
)

type AnyTLSClientConfig struct {
	Address  *Address `json:"address"`
	Port     uint16   `json:"port"`
	Password string   `json:"password"`
}

func (c *AnyTLSClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, errors.New("AnyTLS server address is not set.")
	}
	if c.Port == 0 {
		return nil, errors.New("Invalid AnyTLS port.")
	}
	if c.Password == "" {
		return nil, errors.New("AnyTLS password is not specified.")
	}

	return &anytls.Config{
		Address:  c.Address.Address.String(),
		Port:     uint32(c.Port),
		Password: c.Password,
	}, nil
}
