package main

import (
	"log"

	"github.com/steveyiyo/hackyou-backend/internal/config"
	h "github.com/steveyiyo/hackyou-backend/internal/http"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	r := h.NewRouter(cfg)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
