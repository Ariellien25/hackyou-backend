package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	conns map[string]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{conns: map[string]*websocket.Conn{}}
}

func (h *Hub) Add(id string, c *websocket.Conn) {
	h.mu.Lock()
	h.conns[id] = c
	h.mu.Unlock()
}

func (h *Hub) Get(id string) (*websocket.Conn, bool) {
	h.mu.RLock()
	c, ok := h.conns[id]
	h.mu.RUnlock()
	return c, ok
}

func (h *Hub) Remove(id string) {
	h.mu.Lock()
	delete(h.conns, id)
	h.mu.Unlock()
}
