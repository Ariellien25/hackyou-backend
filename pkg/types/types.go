package types

type CreateSessionReq struct {
	Device  map[string]string `json:"device"`
	Mode    string            `json:"mode"`
	Locale  string            `json:"locale"`
	Consent map[string]bool   `json:"consent"`
}

type CreateSessionResp struct {
	SessionID string                 `json:"session_id"`
	WSURL     string                 `json:"ws_url"`
	WebRTC    map[string]interface{} `json:"webrtc"`
}

type TTSReq struct {
	SessionID string  `json:"session_id"`
	Text      string  `json:"text"`
	Voice     string  `json:"voice"`
	Speed     float32 `json:"speed"`
	Pitch     float32 `json:"pitch"`
	Format    string  `json:"format"`
	Cache     bool    `json:"cache"`
}

type TTSResp struct {
	AudioURL   string `json:"audio_url"`
	DurationMs int64  `json:"duration_ms"`
}

type Tip struct {
	T        int64   `json:"t"`
	Text     string  `json:"text"`
	Priority string  `json:"priority"`
	Yaw      float64 `json:"yaw_deg,omitempty"`
	Pitch    float64 `json:"pitch_deg,omitempty"`
	Roll     float64 `json:"roll_deg,omitempty"`
	Reason   string  `json:"reason,omitempty"`
}

type SummaryResp struct {
	SessionID      string `json:"session_id"`
	LatencyP50Ms   int64  `json:"latency_ms_p50"`
	FramesAnalyzed int64  `json:"frames_analyzed"`
	Tips           []Tip  `json:"tips"`
}

type WebRTCOfferReq struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
}

type WebRTCAnswerResp struct {
	SDP        string                   `json:"sdp"`
	ICEServers []map[string]interface{} `json:"ice_servers"`
}
