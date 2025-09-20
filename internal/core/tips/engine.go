package tips

import (
	"context"
	"os"
	"time"

	"github.com/steveyiyo/hackyou-backend/internal/core/gemini"
	"github.com/steveyiyo/hackyou-backend/pkg/types"
)

type Provider interface {
	TipFromImage(ctx context.Context, img []byte, mime string) (*types.Tip, string, error)
}

type Engine struct {
	P Provider
}

func New() *Engine {
	var p Provider
	if k := os.Getenv("GOOGLE_API_KEY"); k != "" {
		m := os.Getenv("GEMINI_MODEL")
		if m == "" {
			m = "gemini-2.5-flash"
		}
		if g, err := gemini.New(k, m); err == nil {
			p = g
		}
	}
	return &Engine{P: p}
}

func (e *Engine) DecideTip() *types.Tip {
	return &types.Tip{
		T:        time.Now().UnixMilli(),
		Text:     "Gemini API 連線中 ...",
		Priority: "high",
		Yaw:      -3,
		Pitch:    6,
		Roll:     0,
		Reason:   "framing_face_rule",
	}
}

func (e *Engine) DecideTipFromFrame(ctx context.Context, img []byte, mime string) *types.Tip {
	if e.P != nil && len(img) > 0 && mime != "" {
		if tip, _, err := e.P.TipFromImage(ctx, img, mime); err == nil && tip != nil {
			if tip.T == 0 {
				tip.T = time.Now().UnixMilli()
			}
			if tip.Priority == "" {
				tip.Priority = "high"
			}
			if tip.Reason == "" {
				tip.Reason = "gemini"
			}
			return tip
		}
	}
	return e.DecideTip()
}
