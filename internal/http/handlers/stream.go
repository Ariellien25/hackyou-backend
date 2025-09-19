package handlers

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/internal/core/tips"
	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
	conn.SetReadLimit(5 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		h.Repo.IncFrame(id)
		if len(msg) > 0 {
			_ = base64.StdEncoding
		}
		t := h.Tips.DecideTip()
		h.Repo.AppendTip(id, *t)
		_ = conn.WriteJSON(gin.H{
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
		})
	}
}
