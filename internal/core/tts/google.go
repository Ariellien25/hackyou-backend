package tts

import (
	"crypto/sha1"
	"encoding/hex"
	"time"
)

type Provider interface {
	Synthesize(text, voice, format string, speed, pitch float32) (url string, durMs int64, err error)
}

type GoogleStub struct {
	Base string
}

func NewGoogleStub(base string) *GoogleStub {
	return &GoogleStub{Base: base}
}

func (g *GoogleStub) Synthesize(text, voice, format string, speed, pitch float32) (string, int64, error) {
	h := sha1.New()
	h.Write([]byte(text + voice + format))
	key := hex.EncodeToString(h.Sum(nil))[:16]
	url := g.Base + "/" + key + "." + format
	return url, 1200 + time.Now().Unix()%200, nil
}
