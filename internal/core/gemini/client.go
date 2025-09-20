package gemini

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/steveyiyo/hackyou-backend/pkg/types"
)

type Client struct {
	c     *genai.Client
	model string
}

func New(apiKey, model string) (*Client, error) {
	tr := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		ForceAttemptHTTP2: false,
		MaxIdleConns:      100,
		IdleConnTimeout:   90 * time.Second,
	}
	hc := &http.Client{Transport: tr, Timeout: 30 * time.Second}
	reqTimeout := 15 * time.Second
	cl, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: hc,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1",
			Timeout:    &reqTimeout,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{c: cl, model: model}, nil
}

func (g *Client) Close() error { return nil }

func (g *Client) TipFromImage(ctx context.Context, img []byte, mime string) (*types.Tip, string, error) {
	parts := []*genai.Part{
		{Text: "你是專業攝影教練。僅輸出 JSON，格式: {\"text\":\"string\",\"yaw_deg\":\"number\",\"pitch_deg\":\"number\",\"roll_deg\":\"number\"}，角度欄位可省略。語言用繁體中文，text 要短且可操作。"},
		{InlineData: &genai.Blob{Data: img, MIMEType: mime}},
	}

	temp := float32(0.2)
	topP := float32(0.8)
	maxTok := int32(12800)

	cfgJSON := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"text":      {Type: genai.TypeString},
				"yaw_deg":   {Type: genai.TypeNumber},
				"pitch_deg": {Type: genai.TypeNumber},
				"roll_deg":  {Type: genai.TypeNumber},
			},
			Required: []string{"text"},
		},
		Temperature:     &temp,
		TopP:            &topP,
		MaxOutputTokens: maxTok,
	}
	cfgText := &genai.GenerateContentConfig{
		Temperature:     &temp,
		TopP:            &topP,
		MaxOutputTokens: maxTok,
	}

	if tip, raw, err := g.callOnce(ctx, parts, cfgJSON); err == nil && tip != nil {
		return tip, raw, nil
	}
	return g.callOnce(ctx, parts, cfgText)
}

func (g *Client) callOnce(ctx context.Context, parts []*genai.Part, cfg *genai.GenerateContentConfig) (*types.Tip, string, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := g.c.Models.GenerateContent(ctx, g.model, []*genai.Content{{Parts: parts}}, cfg)
		if err != nil {
			lastErr = err
			if retriable(err) {
				time.Sleep(time.Duration(300*(i+1)) * time.Millisecond)
				continue
			}
			return nil, "", err
		}
		if tip, raw, ok := parseTip(resp); ok {
			finalize(tip)
			return tip, raw, nil
		}
		lastErr = errors.New("empty response")
		time.Sleep(time.Duration(300*(i+1)) * time.Millisecond)
	}
	return nil, "", lastErr
}

func parseTip(resp *genai.GenerateContentResponse) (*types.Tip, string, bool) {
	var out types.Tip
	var raw string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, p := range cand.Content.Parts {
				if p.InlineData != nil && p.InlineData.MIMEType == "application/json" {
					raw = string(p.InlineData.Data)
					if json.Unmarshal(p.InlineData.Data, &out) == nil && out.Text != "" {
						return &out, raw, true
					}
				}
				if p.Text != "" {
					raw = p.Text
					var tmp types.Tip
					if json.Unmarshal([]byte(p.Text), &tmp) == nil && tmp.Text != "" {
						return &tmp, raw, true
					}
				}
			}
		}
	}
	if t := resp.Text(); t != "" {
		raw = t
		out = types.Tip{T: time.Now().UnixMilli(), Text: t, Priority: "high", Reason: "gemini"}
		return &out, raw, true
	}
	return nil, "", false
}

func finalize(t *types.Tip) {
	if t.T == 0 {
		t.T = time.Now().UnixMilli()
	}
	if t.Priority == "" {
		t.Priority = "high"
	}
	if t.Reason == "" {
		t.Reason = "gemini"
	}
}

func retriable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unexpected EOF") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "RST_STREAM") ||
		strings.Contains(s, "connection reset")
}
