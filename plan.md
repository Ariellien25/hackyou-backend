You are an expert Go developer. This is the final step of our integration plan. We have successfully created a robust, concurrent Gemini Live API client in `internal/core/gemini/live_client.go`. Your final task is to integrate this new client into our existing client-facing WebSocket handler.

**Your Task:**
Refactor the file `internal/http/handlers/stream.go`. You must replace the old logic that calls the standard Gemini `GenerateContent` API with the new streaming logic using our `LiveClient`.

**Refactoring Steps:**

1.  **Initialization:**
    -   At the beginning of the WebSocket handler function (likely named `Stream` or similar), instantiate our new live client: `liveClient := gemini.NewLiveClient()`.
    -   Immediately after, start the streaming session with Gemini: `err := liveClient.StartStreamingSession()`. You must handle any potential error here (e.g., log the error and close the client's connection).
    -   **Crucially**, use `defer liveClient.Close()` right after a successful connection to ensure resources are cleaned up when the handler function exits.
    -   **Authentication**: In the `StartStreamingSession` call within `live_client.go`, the API key is hardcoded as a placeholder. Modify the `live_client.go`'s `StartStreamingSession` to accept the API key as a parameter, and in `stream.go`, pass the key from a secure source like an environment variable or a config manager.

2.  **Advice Proxying Goroutine:**
    -   After starting the session, get the advice channel: `adviceChan, err := liveClient.ReceiveAdvice()`.
    -   Launch a **new goroutine**. The sole purpose of this goroutine is to proxy advice from Gemini back to the end-user's client. It should loop and read from `adviceChan`:
      ```go
      go func() {
          for advice := range adviceChan {
              // Write the 'advice' string back to the client's WebSocket connection
              // (e.g., clientConn.WriteMessage(websocket.TextMessage, []byte(advice)))
          }
      }()
      ```

3.  **Main Read Loop Modification:**
    -   Find the existing `for` loop that reads messages (image frames) from the end-user's client connection.
    -   **Remove** all the old code inside this loop that called the standard Gemini client (`Models.GenerateContent`).
    -   **Replace** it with a single, non-blocking call to our new client: `liveClient.SendImageFrame(imageBytes)`.
    -   Add a `time.Sleep` of 2-3 seconds inside this loop to implement the "Smart Sampling" strategy defined in `GEMINI.md`, preventing you from overwhelming the API and controlling costs.

**Final Instruction:**
Your only output should be the complete, refactored source code for `internal/http/handlers/stream.go`. This new version should be fully integrated with the `live_client.go` module and contain no trace of the old Gemini API calls.