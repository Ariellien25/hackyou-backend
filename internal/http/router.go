package http

import (
	"os"

	"github.com/steveyiyo/hackyou-backend/internal/config"
	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/internal/core/tips"
	ttsprov "github.com/steveyiyo/hackyou-backend/internal/core/tts"
	"github.com/steveyiyo/hackyou-backend/internal/http/handlers"
	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config) *gin.Engine {
	r := gin.Default()
	repo := memory.NewSessionRepo()
	svc := session.NewService(repo)
	engine := tips.New()
	hub := ws.NewHub()
	tts := ttsprov.NewGoogleStub(cfg.TTSBase)
	baseScheme := "http"
	if os.Getenv("TLS") == "1" {
		baseScheme = "https"
	}
	host := os.Getenv("PUBLIC_HOST")
	if host == "" {
		host = "localhost:" + cfg.Port
	}
	sh := handlers.NewSessionsHandler(svc, baseScheme, host)
	wsh := handlers.NewStreamHandler(hub, repo, engine, svc)
	wh := handlers.NewWebRTCHandler()
	th := handlers.NewTTSHandler(tts)
	api := r.Group("/v1")
	api.POST("/sessions", sh.Create)
	api.GET("/sessions/:id/summary", sh.Summary)
	api.POST("/webrtc/offer", wh.Offer)
	api.POST("/tts", th.Synthesize)
	r.GET("/v1/stream", wsh.WS)
	return r
}
