package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/steveyiyo/hackyou-backend/internal/core/gemini"
	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/internal/core/tips"
	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/types"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"
)

type StreamHandler struct {
	Hub      *ws.Hub
	Repo     *memory.SessionRepo
	Tips     *tips.Engine
	Sess     *session.Service
	Gem      *gemini.Client
	Upgrader websocket.Upgrader
}

func NewStreamHandler(h *ws.Hub, r *memory.SessionRepo, e *tips.Engine, s *session.Service, g *gemini.Client) *StreamHandler {
	return &StreamHandler{
		Hub:  h,
		Repo: r,
		Tips: e,
		Sess: s,
		Gem:  g,
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

	_ = conn.WriteJSON(gin.H{"type": "hello", "ts": time.Now().UnixMilli()})

	type frameMsg struct {
		Type        string `json:"type"`
		Bytes       string `json:"bytes"`
		ContentType string `json:"content_type"`
	}

	done := make(chan struct{})
	started := make(chan struct{}, 1)
	startOnce := false

	go func() {
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				close(done)
				return
			}
			if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
				continue
			}
			h.Repo.IncFrame(id)
			var fm frameMsg
			if json.Unmarshal(msg, &fm) == nil && fm.Bytes != "" && fm.ContentType != "" {
				if b, err := base64.StdEncoding.DecodeString(fm.Bytes); err == nil {
					h.Repo.SetFrame(id, fm.ContentType, b)
				}
			}
			if !startOnce {
				startOnce = true
				select {
				case started <- struct{}{}:
				default:
				}
			}
		}
	}()

	interval := 2 * time.Second
	var next time.Time
	ctx := context.Background()

	for {
		if next.IsZero() {
			select {
			case <-done:
				return
			case <-started:
				next = time.Now().Add(interval)
			}
			continue
		}
		d := time.Until(next)
		if d < 0 {
			d = 0
		}
		timer := time.NewTimer(d)
		select {
		case <-done:
			timer.Stop()
			return
		case <-timer.C:
			sess, ok := h.Repo.Get(id)
			if !ok {
				return
			}

			var out types.Tip
			var respRaw string
			var usedGemini bool

			if h.Gem != nil && len(sess.LastFrame) > 0 && sess.LastFrameMIM != "" {
				if tip, raw, err := h.Gem.TipFromImage(ctx, sess.LastFrame, sess.LastFrameMIM); err == nil && tip != nil {
					out = *tip
					respRaw = raw
					usedGemini = true
				} else {
					t := h.Tips.DecideTip()
					out = *t
				}
			} else {
				t := h.Tips.DecideTip()
				out = *t
			}

			h.Repo.AppendTip(id, out)

			m := gin.H{
				"type":     "tip",
				"ts":       out.T,
				"priority": out.Priority,
				"text":     out.Text,
				"hint": gin.H{
					"yaw_deg":   out.Yaw,
					"pitch_deg": out.Pitch,
					"roll_deg":  out.Roll,
				},
				"reason": out.Reason,
			}
			if usedGemini {
				m["source"] = "gemini"
				m["resp"] = respRaw
			} else {
				m["source"] = "stub"
			}

			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(m); err != nil {
				return
			}
			next = next.Add(interval)
		}
	}
}
