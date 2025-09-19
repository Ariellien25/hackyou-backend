package handlers

import (
	"net/http"

	"github.com/steveyiyo/hackyou-backend/pkg/types"

	"github.com/gin-gonic/gin"
)

type WebRTCHandler struct{}

func NewWebRTCHandler() *WebRTCHandler { return &WebRTCHandler{} }

func (h *WebRTCHandler) Offer(c *gin.Context) {
	var req types.WebRTCOfferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	c.JSON(http.StatusOK, types.WebRTCAnswerResp{
		SDP: req.SDP,
		ICEServers: []map[string]interface{}{
			{"urls": []string{"stun:stun.l.google.com:19302"}},
		},
	})
}
