// Package websocket — hub realtime para broadcast de métricas e progresso.
//
// Padrão "fan-out": vários clientes conectam, hub recebe mensagens e replica
// para todos. Cada cliente roda numa goroutine isolada (read + write loops).
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pinas/pinas/internal/middleware"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

// Message é o envelope universal de eventos.
type Message struct {
	Type    string      `json:"type"`             // "stats", "upload_progress", etc.
	Payload interface{} `json:"payload,omitempty"`
	TS      int64       `json:"ts"`
}

type client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	user string
	role string
}

type Hub struct {
	clients    map[*client]bool
	register   chan *client
	unregister chan *client
	broadcast  chan []byte
	log        *slog.Logger
	mu         sync.RWMutex
}

func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		register:   make(chan *client, 16),
		unregister: make(chan *client, 16),
		broadcast:  make(chan []byte, 256),
		log:        log,
	}
}

// Run loop principal do hub. Chamar em goroutine.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for c := range h.clients {
				close(c.send)
			}
			h.clients = nil
			h.mu.Unlock()
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Cliente lento: descarta a conexão para não travar o hub.
					go func(cc *client) { h.unregister <- cc }(c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast envia para todos os clientes conectados.
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	b, err := json.Marshal(Message{
		Type:    msgType,
		Payload: payload,
		TS:      time.Now().Unix(),
	})
	if err != nil {
		return
	}
	select {
	case h.broadcast <- b:
	default:
		h.log.Warn("hub broadcast buffer cheio, descartando msg", "type", msgType)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Caddy é proxy reverso, então o origin chega corretamente.
		// Restrição de origin é feita no Caddy/CORS — aqui aceitamos.
		return true
	},
}

// ServeHTTP atende a /ws — handler para o chi router. Requer autenticação prévia (JWT middleware).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws upgrade failed", "err", err)
		return
	}
	c := &client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 64),
		user: claims.Username,
		role: claims.Role,
	}
	h.register <- c
	go c.writePump()
	go c.readPump()
}

func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		// Não esperamos comandos do cliente no MVP — só ping/pong.
		// Subscriptions topicadas ficam para v2.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// StartMetricsBroadcaster dispara um sinal de tick a cada `every`. Cancela com ctx.
// O frontend escuta e re-fetch /system/stats.
// Em v2: serializar Stats inline aqui e remover o polling.
func (h *Hub) StartMetricsBroadcaster(ctx context.Context, every time.Duration) {
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.Broadcast("metrics_tick", map[string]int64{"ts": time.Now().Unix()})
			}
		}
	}()
}
