package webpanel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	handlerservice "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/serial"
	core "github.com/xtls/xray-core/core"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type editableInboundConfig struct {
	Tag              string          `json:"tag"`
	ReceiverType     string          `json:"receiverType"`
	ReceiverSettings json.RawMessage `json:"receiverSettings"`
	ProxyType        string          `json:"proxyType"`
	ProxySettings    json.RawMessage `json:"proxySettings"`
}

type editableOutboundConfig struct {
	Tag            string          `json:"tag"`
	SenderType     string          `json:"senderType"`
	SenderSettings json.RawMessage `json:"senderSettings"`
	ProxyType      string          `json:"proxyType"`
	ProxySettings  json.RawMessage `json:"proxySettings"`
	Expire         int64           `json:"expire,omitempty"`
	Comment        string          `json:"comment,omitempty"`
}

func decodeInboundHandlerConfig(raw json.RawMessage) (*core.InboundHandlerConfig, error) {
	if inboundConfig, ok, err := decodeEditableInboundHandlerConfig(raw); ok || err != nil {
		return inboundConfig, err
	}

	var inboundConfig core.InboundHandlerConfig
	if err := protojson.Unmarshal(raw, &inboundConfig); err == nil && strings.TrimSpace(inboundConfig.Tag) != "" {
		return &inboundConfig, nil
	}

	return nil, fmt.Errorf("invalid inbound config")
}

func decodeEditableOutboundHandlerConfig(raw json.RawMessage) (*core.OutboundHandlerConfig, bool, error) {
	var outboundConfig editableOutboundConfig
	if err := json.Unmarshal(raw, &outboundConfig); err != nil {
		return nil, false, nil
	}
	if strings.TrimSpace(outboundConfig.SenderType) == "" && strings.TrimSpace(outboundConfig.ProxyType) == "" {
		return nil, false, nil
	}

	decoded, err := outboundConfig.toCore()
	if err != nil {
		return nil, true, err
	}

	return decoded, true, nil
}

func decodeEditableInboundHandlerConfig(raw json.RawMessage) (*core.InboundHandlerConfig, bool, error) {
	var inboundConfig editableInboundConfig
	if err := json.Unmarshal(raw, &inboundConfig); err != nil {
		return nil, false, nil
	}
	if strings.TrimSpace(inboundConfig.ReceiverType) == "" && strings.TrimSpace(inboundConfig.ProxyType) == "" {
		return nil, false, nil
	}

	decoded, err := inboundConfig.toCore()
	if err != nil {
		return nil, true, err
	}

	return decoded, true, nil
}

func editableInboundFromCore(inboundConfig *core.InboundHandlerConfig) (editableInboundConfig, error) {
	receiverType, receiverSettings, err := encodeJSONTypedMessage(inboundConfig.ReceiverSettings)
	if err != nil {
		return editableInboundConfig{}, err
	}
	proxyType, proxySettings, err := encodeJSONTypedMessage(inboundConfig.ProxySettings)
	if err != nil {
		return editableInboundConfig{}, err
	}

	return editableInboundConfig{
		Tag:              inboundConfig.Tag,
		ReceiverType:     receiverType,
		ReceiverSettings: receiverSettings,
		ProxyType:        proxyType,
		ProxySettings:    proxySettings,
	}, nil
}

func editableOutboundFromCore(outboundConfig *core.OutboundHandlerConfig) (editableOutboundConfig, error) {
	senderType, senderSettings, err := encodeJSONTypedMessage(outboundConfig.SenderSettings)
	if err != nil {
		return editableOutboundConfig{}, err
	}
	proxyType, proxySettings, err := encodeJSONTypedMessage(outboundConfig.ProxySettings)
	if err != nil {
		return editableOutboundConfig{}, err
	}

	return editableOutboundConfig{
		Tag:            outboundConfig.Tag,
		SenderType:     senderType,
		SenderSettings: senderSettings,
		ProxyType:      proxyType,
		ProxySettings:  proxySettings,
		Expire:         outboundConfig.Expire,
		Comment:        outboundConfig.Comment,
	}, nil
}

func (c editableInboundConfig) toCore() (*core.InboundHandlerConfig, error) {
	if strings.TrimSpace(c.Tag) == "" {
		return nil, fmt.Errorf("inbound tag is required")
	}

	receiverSettings, err := decodeJSONTypedMessage(c.ReceiverType, c.ReceiverSettings)
	if err != nil {
		return nil, fmt.Errorf("decode inbound receiver settings: %w", err)
	}
	proxySettings, err := decodeJSONTypedMessage(c.ProxyType, c.ProxySettings)
	if err != nil {
		return nil, fmt.Errorf("decode inbound proxy settings: %w", err)
	}
	if proxySettings == nil {
		return nil, fmt.Errorf("inbound proxy settings are required")
	}

	return &core.InboundHandlerConfig{
		Tag:              c.Tag,
		ReceiverSettings: receiverSettings,
		ProxySettings:    proxySettings,
	}, nil
}

func (c editableOutboundConfig) toCore() (*core.OutboundHandlerConfig, error) {
	if strings.TrimSpace(c.Tag) == "" {
		return nil, fmt.Errorf("outbound tag is required")
	}

	senderSettings, err := decodeJSONTypedMessage(c.SenderType, c.SenderSettings)
	if err != nil {
		return nil, fmt.Errorf("decode outbound sender settings: %w", err)
	}
	proxySettings, err := decodeJSONTypedMessage(c.ProxyType, c.ProxySettings)
	if err != nil {
		return nil, fmt.Errorf("decode outbound proxy settings: %w", err)
	}
	if proxySettings == nil {
		return nil, fmt.Errorf("outbound proxy settings are required")
	}

	return &core.OutboundHandlerConfig{
		Tag:            c.Tag,
		SenderSettings: senderSettings,
		ProxySettings:  proxySettings,
		Expire:         c.Expire,
		Comment:        c.Comment,
	}, nil
}

func encodeJSONTypedMessage(tm *serial.TypedMessage) (string, json.RawMessage, error) {
	if tm == nil {
		return "", nil, nil
	}

	message, err := tm.GetInstance()
	if err != nil {
		return "", nil, fmt.Errorf("decode typed message %q: %w", tm.Type, err)
	}

	raw, err := protojson.Marshal(message)
	if err != nil {
		return "", nil, fmt.Errorf("marshal typed message %q: %w", tm.Type, err)
	}
	return tm.Type, json.RawMessage(raw), nil
}

func decodeJSONTypedMessage(messageType string, raw json.RawMessage) (*serial.TypedMessage, error) {
	cleanType := strings.TrimSpace(messageType)
	cleanRaw := bytes.TrimSpace(raw)
	if cleanType == "" {
		if len(cleanRaw) == 0 || bytes.Equal(cleanRaw, []byte("null")) {
			return nil, nil
		}
		return nil, fmt.Errorf("message type is required")
	}

	instance, err := serial.GetInstance(cleanType)
	if err != nil {
		return nil, fmt.Errorf("unsupported message type %q: %w", cleanType, err)
	}

	protoMessage, ok := instance.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("message type %q is not a protobuf message", cleanType)
	}
	if len(cleanRaw) == 0 || bytes.Equal(cleanRaw, []byte("null")) {
		cleanRaw = []byte("{}")
	}

	if err := protojson.Unmarshal(cleanRaw, protoMessage); err != nil {
		return nil, fmt.Errorf("invalid settings for %q: %w", cleanType, err)
	}
	return serial.ToTypedMessage(protoMessage), nil
}

func messageProtocolLabel(messageType string) string {
	parts := strings.Split(messageType, ".")
	for i, part := range parts {
		if part == "proxy" && i+1 < len(parts) {
			return strings.ToLower(parts[i+1])
		}
	}
	return strings.ToLower(messageType)
}

func findInboundConfig(handler handlerservice.HandlerServiceClient, tag string) (*core.InboundHandlerConfig, error) {
	resp, err := handler.ListInbounds(context.Background(), &handlerservice.ListInboundsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list inbounds: %w", err)
	}
	for _, inboundConfig := range resp.Inbounds {
		if inboundConfig.GetTag() == tag {
			return proto.Clone(inboundConfig).(*core.InboundHandlerConfig), nil
		}
	}
	return nil, fmt.Errorf("inbound %q not found", tag)
}

func findOutboundConfig(handler handlerservice.HandlerServiceClient, tag string) (*core.OutboundHandlerConfig, error) {
	resp, err := handler.ListOutbounds(context.Background(), &handlerservice.ListOutboundsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list outbounds: %w", err)
	}
	for _, outboundConfig := range resp.Outbounds {
		if outboundConfig.GetTag() == tag {
			return proto.Clone(outboundConfig).(*core.OutboundHandlerConfig), nil
		}
	}
	return nil, fmt.Errorf("outbound %q not found", tag)
}

func (wp *WebPanel) replaceInboundConfig(tag string, inboundConfig *core.InboundHandlerConfig) error {
	currentConfig, err := findInboundConfig(wp.grpcClient.Handler(), tag)
	if err != nil {
		return err
	}

	ctx := wp.grpcClient.Context()
	if _, err := wp.grpcClient.Handler().RemoveInbound(ctx, &handlerservice.RemoveInboundRequest{Tag: tag}); err != nil {
		return fmt.Errorf("remove existing inbound: %w", err)
	}
	if _, err := wp.grpcClient.Handler().AddInbound(ctx, &handlerservice.AddInboundRequest{Inbound: inboundConfig}); err != nil {
		if _, rollbackErr := wp.grpcClient.Handler().AddInbound(ctx, &handlerservice.AddInboundRequest{Inbound: currentConfig}); rollbackErr != nil {
			return fmt.Errorf("apply replacement: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("apply replacement: %w", err)
	}
	return nil
}

func (wp *WebPanel) replaceOutboundConfig(tag string, outboundConfig *core.OutboundHandlerConfig) error {
	currentConfig, err := findOutboundConfig(wp.grpcClient.Handler(), tag)
	if err != nil {
		return err
	}

	ctx := wp.grpcClient.Context()
	if _, err := wp.grpcClient.Handler().RemoveOutbound(ctx, &handlerservice.RemoveOutboundRequest{Tag: tag}); err != nil {
		return fmt.Errorf("remove existing outbound: %w", err)
	}
	if _, err := wp.grpcClient.Handler().AddOutbound(ctx, &handlerservice.AddOutboundRequest{Outbound: outboundConfig}); err != nil {
		if _, rollbackErr := wp.grpcClient.Handler().AddOutbound(ctx, &handlerservice.AddOutboundRequest{Outbound: currentConfig}); rollbackErr != nil {
			return fmt.Errorf("apply replacement: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("apply replacement: %w", err)
	}
	return nil
}
