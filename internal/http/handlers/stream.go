package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/internal/core/tips"
	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"
)

type StreamHandler struct {
	Hub      *ws.Hub
	Repo     *memory.SessionRepo
	Tips     *tips.Engine
	Sess     *session.Service
	Upgrader websocket.Upgrader
}

func NewStreamHandler(h *ws.Hub, r *memory.SessionRepo, e *tips.Engine, s *session.Service) *StreamHandler {
	return &StreamHandler{
		Hub:  h,
		Repo: r,
		Tips: e,
		Sess: s,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *StreamHandler) WS(c *gin.Context) {
	id := c.Query("sess")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	conn, err := h.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.Hub.Add(id, conn)
	defer func() {
		h.Hub.Remove(id)
		conn.Close()
	}()

	conn.SetReadLimit(8 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	_ = conn.WriteJSON(gin.H{
		"type": "hello",
		"ts":   time.Now().UnixMilli(),
	})

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}

		h.Repo.IncFrame(id)

		var kind struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(msg, &kind)

		t := h.Tips.DecideTip()
		h.Repo.AppendTip(id, *t)

		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteJSON(gin.H{
			"type":     "tip",
			"ts":       t.T,
			"priority": t.Priority,
			"text":     t.Text,
			"hint": gin.H{
				"yaw_deg":   t.Yaw,
				"pitch_deg": t.Pitch,
				"roll_deg":  t.Roll,
			},
			"reason": t.Reason,
		}); err != nil {
			return
		}
	}
}
