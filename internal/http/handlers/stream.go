package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/steveyiyo/hackyou-backend/internal/config"
	"github.com/steveyiyo/hackyou-backend/internal/core/gemini"
	"github.com/steveyiyo/hackyou-backend/pkg/ws"
)

type StreamHandler struct {
	Hub      *ws.Hub
	Upgrader websocket.Upgrader
}

func NewStreamHandler(h *ws.Hub) *StreamHandler {
	return &StreamHandler{
		Hub: h,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *StreamHandler) WS(c *gin.Context) {
	id := c.Query("sess")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	clientConn, err := h.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	h.Hub.Add(id, clientConn)
	defer func() {
		h.Hub.Remove(id)
		clientConn.Close()
	}()

	cfg := config.Load()
	liveClient := gemini.NewLiveClient()
	if err := liveClient.StartStreamingSession(cfg.GeminiAPIKey); err != nil {
		log.Printf("Failed to start Gemini session: %v", err)
		clientConn.WriteMessage(websocket.TextMessage, []byte("Error: Could not connect to analysis service."))
		return
	}
	defer liveClient.Close()

	// Goroutine to proxy advice from Gemini to the client
	adviceChan, err := liveClient.ReceiveAdvice()
	if err != nil {
		log.Printf("Failed to get advice channel: %v", err)
		clientConn.WriteMessage(websocket.TextMessage, []byte("Error: Could not receive advice from service."))
		return
	}

	go func() {
		for advice := range adviceChan {
			if err := clientConn.WriteJSON(gin.H{"type": "tip", "source": "gemini", "text": advice}); err != nil {
				log.Printf("Error sending advice to client: %v", err)
				return
			}
		}
	}()

	// Main loop to read frames from client and send to Gemini
	for {
		_, message, err := clientConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client connection error: %v", err)
			}
			break // Exit loop on client disconnect
		}

		// For now, we assume the message is the raw image bytes.
		// A more robust implementation would parse a JSON message like the original code.
		// Refined version (simpler and more efficient)
		if err := liveClient.SendImageFrame(message); err != nil {
			log.Printf("Failed to send image frame: %v", err)
		}

		// Smart Sampling: Wait before processing the next frame.
		time.Sleep(2 * time.Second)
	}
}