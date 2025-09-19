package handlers

import (
	"net/http"

	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/pkg/types"

	"github.com/gin-gonic/gin"
)

type SessionsHandler struct {
	Svc    *session.Service
	Scheme string
	Host   string
}

func NewSessionsHandler(svc *session.Service, scheme, host string) *SessionsHandler {
	return &SessionsHandler{Svc: svc, Scheme: scheme, Host: host}
}

func (h *SessionsHandler) Create(c *gin.Context) {
	var req types.CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	sess := h.Svc.Create(req.Mode, req.Locale, req.Device, req.Consent)
	ws := h.Scheme + "://" + h.Host + "/v1/stream?sess=" + sess.ID
	c.JSON(http.StatusOK, types.CreateSessionResp{
		SessionID: sess.ID,
		WSURL:     ws,
		WebRTC:    map[string]interface{}{"offer_url": "/v1/webrtc/offer"},
	})
}

func (h *SessionsHandler) Summary(c *gin.Context) {
	id := c.Param("id")
	sum, ok := h.Svc.Summary(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, sum)
}
