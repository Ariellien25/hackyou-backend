package session

import (
	"time"

	"github.com/steveyiyo/hackyou-backend/internal/repo/memory"
	"github.com/steveyiyo/hackyou-backend/pkg/types"

	"github.com/google/uuid"
)

type Service struct {
	Repo *memory.SessionRepo
}

func NewService(repo *memory.SessionRepo) *Service {
	return &Service{Repo: repo}
}

func (s *Service) Create(mode, locale string, device map[string]string, consent map[string]bool) *memory.Session {
	id := "sess_" + uuid.NewString()
	sess := &memory.Session{
		ID:         id,
		CreatedAt:  time.Now(),
		Mode:       mode,
		Locale:     locale,
		Device:     device,
		Consent:    consent,
		Tips:       []types.Tip{},
		LatencyP50: 380,
	}
	s.Repo.Save(sess)
	return sess
}

func (s *Service) Summary(id string) (types.SummaryResp, bool) {
	sess, ok := s.Repo.Get(id)
	if !ok {
		return types.SummaryResp{}, false
	}
	return types.SummaryResp{
		SessionID:      sess.ID,
		LatencyP50Ms:   sess.LatencyP50,
		FramesAnalyzed: sess.Frames,
		Tips:           sess.Tips,
	}, true
}
