package memory

import (
	"sync"
	"time"

	"github.com/steveyiyo/hackyou-backend/pkg/types"
)

type Session struct {
	ID         string
	CreatedAt  time.Time
	Mode       string
	Locale     string
	Device     map[string]string
	Consent    map[string]bool
	Tips       []types.Tip
	Frames     int64
	LatencyP50 int64
}

type SessionRepo struct {
	m sync.Map
}

func NewSessionRepo() *SessionRepo {
	return &SessionRepo{}
}

func (r *SessionRepo) Save(s *Session) {
	r.m.Store(s.ID, s)
}

func (r *SessionRepo) Get(id string) (*Session, bool) {
	v, ok := r.m.Load(id)
	if !ok {
		return nil, false
	}
	return v.(*Session), true
}

func (r *SessionRepo) AppendTip(id string, t types.Tip) {
	v, ok := r.m.Load(id)
	if !ok {
		return
	}
	s := v.(*Session)
	s.Tips = append(s.Tips, t)
	r.m.Store(id, s)
}

func (r *SessionRepo) IncFrame(id string) {
	v, ok := r.m.Load(id)
	if !ok {
		return
	}
	s := v.(*Session)
	s.Frames++
	r.m.Store(id, s)
}
