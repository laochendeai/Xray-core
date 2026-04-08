package all_test

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func TestAllRegistersAnyTLSOutbound(t *testing.T) {
	t.Parallel()

	instance, err := serial.GetInstance("xray.proxy.anytls.Config")
	if err != nil {
		t.Fatalf("resolve anytls proto type: %v", err)
	}

	if _, err := common.CreateObject(context.Background(), instance); err != nil {
		t.Fatalf("create anytls outbound from registered config: %v", err)
	}
}
