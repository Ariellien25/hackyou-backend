package handlers

import (
	"net/http"

	"github.com/steveyiyo/hackyou-backend/internal/core/tts"
	"github.com/steveyiyo/hackyou-backend/pkg/types"

	"github.com/gin-gonic/gin"
)

type TTSHandler struct {
	Provider tts.Provider
}

func NewTTSHandler(p tts.Provider) *TTSHandler {
	return &TTSHandler{Provider: p}
}

func (h *TTSHandler) Synthesize(c *gin.Context) {
	var req types.TTSReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}
	url, dur, err := h.Provider.Synthesize(req.Text, req.Voice, req.Format, req.Speed, req.Pitch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tts_failed"})
		return
	}
	c.JSON(http.StatusOK, types.TTSResp{AudioURL: url, DurationMs: dur})
}
