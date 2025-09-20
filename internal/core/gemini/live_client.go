// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gemini

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// JSON Structures for API Communication

type sendTextPart struct {
	Text string `json:"text"`
}

type sendSystemInstruction struct {
	Parts []sendTextPart `json:"parts"`
}

type sendInitialConfig struct {
	SystemInstruction sendSystemInstruction `json:"system_instruction"`
}

type sendImagePart struct {
	DataBase64 string `json:"data"`
	MimeType   string `json:"mime_type"`
}

type sendImageFrame struct {
	Image sendImagePart `json:"image"`
}

type receivedTextPart struct {
	Text string `json:"text"`
}

type receivedModelTurn struct {
	Parts []receivedTextPart `json:"parts"`
}

type receivedServerContent struct {
	ModelTurn receivedModelTurn `json:"model_turn"`
}

type receivedMessage struct {
	ServerContent receivedServerContent `json:"server_content"`
}

// LiveClient manages the WebSocket connection to the Gemini Live API.
type LiveClient struct {
	conn       *websocket.Conn
	sendChan   chan []byte
	adviceChan chan string
	doneChan   chan struct{}
}

// NewLiveClient creates a new, uninitialized LiveClient.
func NewLiveClient() *LiveClient {
	return &LiveClient{}
}

// StartStreamingSession establishes a WebSocket connection and begins the streaming session.
func (c *LiveClient) StartStreamingSession(apiKey string) error {
	geminiURL := "wss://generativelanguage.googleapis.com/v1alpha/stream?model=gemini-live-2.5-flash-preview"
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)

	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(geminiURL, headers)
	if err != nil {
		return err
	}

	// Initialize channels
	c.sendChan = make(chan []byte)
	c.adviceChan = make(chan string)
	c.doneChan = make(chan struct{})

	// Send initial configuration
	config := sendInitialConfig{
		SystemInstruction: sendSystemInstruction{
			Parts: []sendTextPart{
				{Text: "You are \"Frame-GPT\", a world-class professional photography assistant. Your purpose is to analyze incoming video frames and provide concise, actionable, and encouraging advice to help the user take better photos. Your analysis should focus on three key areas: 1. Composition: Adherence to rules like the rule of thirds, leading lines, framing, symmetry, and depth. 2. Lighting: Identify the quality and direction of light (e.g., soft, hard, backlighting, golden hour). Suggest adjustments to exposure or position. 3. Subject: Help the user clarify the main subject. Suggest ways to make the subject stand out, like adjusting depth of field or removing distractions. Your responses MUST be: - Concise: No more than 1-2 short sentences. - Actionable: Give a clear instruction, e.g., \"Try lowering the camera angle...\" instead of \"The angle is okay.\" - Real-time: Frame your advice based on the immediate image. - Encouraging: Use a positive and helpful tone. Do not greet the user or engage in small talk. Provide only direct, photographic advice."},
			},
		},
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, configBytes); err != nil {
		return err
	}

	// Launch concurrent read/write handlers
	go c.readMessages()
	go c.writeMessages()

	return nil
}

// readMessages runs in a goroutine, continuously reading from the WebSocket.
func (c *LiveClient) readMessages() {
	defer close(c.adviceChan)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}

		var received receivedMessage
		if err := json.Unmarshal(message, &received); err != nil {
			log.Println("unmarshal error:", err)
			continue
		}

		if len(received.ServerContent.ModelTurn.Parts) > 0 {
			c.adviceChan <- received.ServerContent.ModelTurn.Parts[0].Text
		}
	}
}

// writeMessages runs in a goroutine, handling outgoing messages.
func (c *LiveClient) writeMessages() {
	for {
		select {
		case frame := <-c.sendChan:
			imgFrame := sendImageFrame{
				Image: sendImagePart{
					DataBase64: base64.StdEncoding.EncodeToString(frame),
					MimeType:   "image/jpeg",
				},
			}
			frameBytes, err := json.Marshal(imgFrame)
			if err != nil {
				log.Println("marshal error:", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, frameBytes); err != nil {
				log.Println("write error:", err)
			}
		case <-c.doneChan:
			// Cleanly close the connection by sending a close message and then waiting
			// for the server to close the connection.
			log.Println("closing connection")
			err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
			}
			return
		}
	}
}

// SendImageFrame sends a single image frame over the WebSocket connection.
func (c *LiveClient) SendImageFrame(frame []byte) error {
	if c.sendChan == nil {
		return errors.New("client is not started")
	}
	c.sendChan <- frame
	return nil
}

// ReceiveAdvice returns a channel that streams text advice from Gemini.
func (c *LiveClient) ReceiveAdvice() (<-chan string, error) {
	if c.adviceChan == nil {
		return nil, errors.New("client is not started")
	}
	return c.adviceChan, nil
}

// Close shuts down the client and closes the WebSocket connection.
func (c *LiveClient) Close() {
	if c.doneChan != nil {
		close(c.doneChan)
	}
	if c.conn != nil {
		c.conn.Close()
	}
}