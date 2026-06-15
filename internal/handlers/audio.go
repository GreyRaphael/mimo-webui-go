package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/mimo"
)

// audioRequest is the JSON body for the audio understanding endpoint.
type audioRequest struct {
	FileID string `json:"file_id"`
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// AudioUnderstandingHandler returns a handler that sends audio to the MiMo
// API for understanding, optionally streaming the response via SSE.
func AudioUnderstandingHandler(client *mimo.Client, uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req audioRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		url := req.URL
		if req.FileID != "" {
			path := findUploadPath(uploadDir, req.FileID)
			if path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "file not found for id: " + req.FileID})
				return
			}
			dataURI, err := mimo.FileToBase64DataURI(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "read file: " + err.Error()})
				return
			}
			url = dataURI
		}
		if url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "either file_id or url is required"})
			return
		}

		mediaPart := mimo.ContentPart{
			Type:       "input_audio",
			InputAudio: &mimo.InputAudioObj{Data: url},
		}
		textPart := mimo.ContentPart{Type: "text", Text: req.Prompt}

		systemMsg := mimo.Message{Role: "system", Content: mimo.TextContent("You are a helpful assistant that can understand audio.")}
		userMsg := mimo.Message{Role: "user", Content: mimo.MultiContent([]mimo.ContentPart{mediaPart, textPart})}
		messages := []mimo.Message{systemMsg, userMsg}

		resp, err := client.ChatCompletion(c.Request.Context(), "mimo-v2.5", messages, req.Stream, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "chat completion: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		if req.Stream {
			flusher, _ := c.Writer.(http.Flusher)
			relaySSEStream(c.Writer, flusher, resp.Body, nil, nil)
			return
		}

		var chatResp mimo.ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "decode response: " + err.Error()})
			return
		}
		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("api returned status %d", resp.StatusCode)})
			return
		}
		c.JSON(http.StatusOK, chatResp)
	}
}
