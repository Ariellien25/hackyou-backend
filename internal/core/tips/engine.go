package tips

import (
	"time"

	"github.com/steveyiyo/hackyou-backend/pkg/types"
)

type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) DecideTip() *types.Tip {
	return &types.Tip{
		T:        time.Now().UnixMilli(),
		Text:     "把手機抬高一點",
		Priority: "high",
		Yaw:      -3,
		Pitch:    6,
		Roll:     0,
		Reason:   "framing_face_rule",
	}
}
