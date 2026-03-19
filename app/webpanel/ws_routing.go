package webpanel

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	routerservice "github.com/xtls/xray-core/app/router/command"
	statsservice "github.com/xtls/xray-core/app/stats/command"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleWSRoutingStats handles WebSocket /api/v1/ws/routing-stats.
// Subscribes to routing stats stream and pushes to client.
func (wp *WebPanel) handleWSRoutingStats(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for client disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	stream, err := wp.grpcClient.Routing().SubscribeRoutingStats(ctx, &routerservice.SubscribeRoutingStatsRequest{})
	if err != nil {
		conn.WriteJSON(map[string]string{"error": "Failed to subscribe to routing stats: " + err.Error()})
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := stream.Recv()
			if err != nil {
				conn.WriteJSON(map[string]string{"error": "Stream ended: " + err.Error()})
				return
			}

			data := map[string]interface{}{
				"inboundTag":  msg.InboundTag,
				"outboundTag": msg.OutboundTag,
				"network":     msg.Network.String(),
				"user":        msg.User,
			}
			if msg.TargetDomain != "" {
				data["targetDomain"] = msg.TargetDomain
			}
			if msg.TargetPort != 0 {
				data["targetPort"] = msg.TargetPort
			}
			if msg.Protocol != "" {
				data["protocol"] = msg.Protocol
			}
			if err := conn.WriteJSON(data); err != nil {
				return
			}
		}
	}
}

// handleWSTraffic handles WebSocket /api/v1/ws/traffic.
// Polls traffic stats and pushes to client.
func (wp *WebPanel) handleWSTraffic(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for client disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Query all traffic stats
			resp, err := wp.grpcClient.Stats().QueryStats(wp.grpcClient.Context(), &statsservice.QueryStatsRequest{
				Pattern: "",
				Reset_:  true, // Reset for rate calculation
			})
			if err != nil {
				conn.WriteJSON(map[string]string{"error": "Failed to query stats: " + err.Error()})
				continue
			}

			trafficData := make(map[string]map[string]int64)
			for _, stat := range resp.Stat {
				name := stat.Name
				trafficData[name] = map[string]int64{
					"value": stat.Value,
				}
			}

			payload, _ := json.Marshal(map[string]interface{}{
				"timestamp": time.Now().Unix(),
				"stats":     trafficData,
			})

			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		}
	}
}
