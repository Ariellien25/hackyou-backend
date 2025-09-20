package http

import (
	"os"

	"github.com/steveyiyo/hackyou-backend/internal/config"
	"github.com/steveyiyo/hackyou-backend/internal/core/gemini"
	"github.com/steveyiyo/hackyou-backend/internal/core/session"
	"github.com/steveyiyo/hackyou-backend/internal/core/tips"
	ttsprov "github.com/steveyiyo/hackyou-backend/internal/core/tts"
	"github.com/steveyiyo/hackyou-backend/internal/http/handlers"
	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"

	"github.com/gin-gonic/gin"
)

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func NewRouter(cfg config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(cors())

	repo := memory.NewSessionRepo()
	svc := session.NewService(repo)
	engine := tips.New()
	hub := ws.NewHub()
	tts := ttsprov.NewGoogleStub(cfg.TTSBase)

	var gclient *gemini.Client
	if k := os.Getenv("GOOGLE_API_KEY"); k != "" {
		m := os.Getenv("GEMINI_MODEL")
		if m == "" {
			m = "gemini-2.5-flash"
		}
		if gc, err := gemini.New(k, m); err == nil {
			gclient = gc
		}
	}

	baseScheme := "http"
	if os.Getenv("TLS") == "1" {
		baseScheme = "https"
	}
	host := os.Getenv("PUBLIC_HOST")
	if host == "" {
		host = "localhost:" + cfg.Port
	}

	sh := handlers.NewSessionsHandler(svc, baseScheme, host)
	wsh := handlers.NewStreamHandler(hub, repo, engine, svc, gclient)
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
