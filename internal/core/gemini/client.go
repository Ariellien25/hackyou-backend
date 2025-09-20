package gemini

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
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
	var rt http.RoundTripper = tr
	if os.Getenv("GEMINI_DEBUG") == "1" {
		_ = os.MkdirAll("logs", 0755)
		rt = &dumpTransport{base: tr, w: &lumberjack.Logger{Filename: "logs/gemini-http.log", MaxSize: 50, MaxBackups: 3, MaxAge: 7, Compress: true}}
	}
	hc := &http.Client{Transport: rt, Timeout: 30 * time.Second}
	reqTimeout := 15 * time.Second
	cl, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: hc,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1beta",
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
		{Text: "你現在是專業的攝影教練，假設這個被拍者不太會比姿勢擺表情，你要適當的給他姿勢的指引，包括但不限於「撩一下頭髮」，「雙手叉腰」，「左手扶手肘」，「右手撐臉」。另外要提升他的自信，你要適當的給他讚美，包括但不限於「這樣笑很好看」「Awesome」「Slay」「You look perfect」。所有輸出都必須為 JSON，格式: {\"text\":\"string\",\"yaw_deg\":\"number\",\"pitch_deg\":\"number\",\"roll_deg\":\"number\"}，角度欄位可省略，text 要短且可操作。所有內容用英文回傳。如果你認為使用者的角度很棒，就回傳「Ready!」"},
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
	tip, raw, err := g.callOnce(ctx, parts, cfgJSON)
	// debug
	fmt.Printf("Raw: %s\n", raw)
	if err != nil && (schemaUnsupported(err) || invalidGenerationConfig(err)) {
		return g.callOnce(ctx, parts, cfgText)
	}
	if tip != nil && err == nil {
		return tip, raw, nil
	}
	return g.callOnce(ctx, parts, cfgText)
}

func (g *Client) callOnce(ctx context.Context, parts []*genai.Part, cfg *genai.GenerateContentConfig) (*types.Tip, string, error) {
	var lastErr error
	mt := cfg.MaxOutputTokens
	for i := 0; i < 3; i++ {
		resp, err := g.c.Models.GenerateContent(ctx, g.model, []*genai.Content{{Parts: parts}}, cfg)
		if err != nil {
			lastErr = err
			fmt.Printf("Error: %v\n", err)
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
		fr := ""
		if len(resp.Candidates) > 0 {
			fr = string(resp.Candidates[0].FinishReason)
		}
		if fr == "MAX_TOKENS" && cfg.MaxOutputTokens < mt+256 {
			cfg.MaxOutputTokens += 256
			time.Sleep(time.Duration(300*(i+1)) * time.Millisecond)
			continue
		}
		lastErr = errors.New("empty response")
		time.Sleep(time.Duration(300*(i+1)) * time.Millisecond)
	}
	return nil, "", lastErr
}

func DumpResp(resp *genai.GenerateContentResponse) {
	if resp == nil {
		return
	}
	fmt.Printf("model=%s id=%s\n", resp.ModelVersion, resp.ResponseID)
	var b strings.Builder
	for _, c := range resp.Candidates {
		if c == nil || c.Content == nil {
			continue
		}
		fmt.Printf("finish=%s\n", c.FinishReason)
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				b.WriteString(p.Text)
			}
			if p.InlineData != nil && p.InlineData.MIMEType == "application/json" {
				fmt.Println("json:", string(p.InlineData.Data))
			}
		}
	}
	if u := resp.UsageMetadata; u != nil {
		out := u.CandidatesTokenCount
		fmt.Printf("tokens prompt=%d output=%d total=%d\n", u.PromptTokenCount, out, u.TotalTokenCount)
	}
	txt := strings.TrimSpace(b.String())
	if txt != "" {
		fmt.Println("text:", txt)
	}
}

func parseTip(resp *genai.GenerateContentResponse) (*types.Tip, string, bool) {
	var out types.Tip
	var raw string
	// debug
	DumpResp(resp)

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
					fmt.Printf("!!!!! %s", raw)
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
		// debug
		fmt.Println(raw)
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
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "UNAVAILABLE") ||
		strings.Contains(s, "internal error")
}

func schemaUnsupported(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Unknown name \"responseMimeType\"") ||
		strings.Contains(s, "Unknown name \"responseSchema\"") ||
		strings.Contains(s, "Cannot find field") ||
		strings.Contains(s, "INVALID_ARGUMENT")
}

func invalidGenerationConfig(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "generation_config")
}

type dumpTransport struct {
	base http.RoundTripper
	w    io.Writer
}

func (d *dumpTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	var rb []byte
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		rb = b
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	resp, err := d.base.RoundTrip(r)
	end := time.Now()
	if err != nil {
		io.WriteString(d.w, "ERR "+start.UTC().Format(time.RFC3339Nano)+" "+r.Method+" "+r.URL.String()+" "+end.UTC().Format(time.RFC3339Nano)+"\n")
		return resp, err
	}
	var b2 []byte
	if resp.Body != nil {
		b, _ := io.ReadAll(resp.Body)
		b2 = b
		resp.Body = io.NopCloser(bytes.NewReader(b))
	}
	io.WriteString(d.w, "REQ "+start.UTC().Format(time.RFC3339Nano)+" "+r.Method+" "+r.URL.String()+"\n")
	if len(rb) > 0 {
		d.w.Write(rb)
		io.WriteString(d.w, "\n")
	}
	io.WriteString(d.w, "HDR\n")
	for k, v := range resp.Header {
		io.WriteString(d.w, k+": "+strings.Join(v, ",")+"\n")
	}
	io.WriteString(d.w, "RESP "+end.UTC().Format(time.RFC3339Nano)+" "+r.Method+" "+r.URL.String()+" "+resp.Status+"\n")
	if len(b2) > 0 {
		d.w.Write(b2)
		io.WriteString(d.w, "\n")
	}
	return resp, nil
}
