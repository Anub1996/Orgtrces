package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type client struct {
	conn  *websocket.Conn
	orgID string
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]bool),
	}
}

func (h *Hub) Register(conn *websocket.Conn, orgID string) {
	c := &client{conn: conn, orgID: orgID}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	log.Printf("WS client connected (org: %s), total: %d", orgID, len(h.clients))

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		conn.Close()
		log.Printf("WS client disconnected (org: %s), total: %d", orgID, len(h.clients))
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) Broadcast(orgID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.orgID == orgID {
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WS write error: %v", err)
			}
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
