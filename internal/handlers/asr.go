package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/db"
	"mimo-webui/internal/middleware"
	"mimo-webui/internal/mimo"
)

// asrRequest is the JSON body for the ASR (speech recognition) endpoint.
type asrRequest struct {
	FileID   string `json:"file_id"`
	Language string `json:"language"`
	Stream   bool   `json:"stream"`
}

// ASRHandler returns a handler that sends audio to the MiMo ASR API for
// speech recognition, optionally streaming the response via SSE.
func ASRHandler(database *db.DB, uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetAuthUser(c)
		sess, err := getMiMoSession(database, user.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var req asrRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		if req.Language == "" {
			req.Language = "auto"
		}

		if req.FileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file_id is required"})
			return
		}

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

		mediaPart := mimo.ContentPart{
			Type:       "input_audio",
			InputAudio: &mimo.InputAudioObj{Data: dataURI},
		}

		userMsg := mimo.Message{Role: "user", Content: mimo.MultiContent([]mimo.ContentPart{mediaPart})}
		messages := []mimo.Message{userMsg}

		extra := map[string]any{
			"asr_options": map[string]string{
				"language": req.Language,
			},
		}

		resp, err := sess.Client.ChatCompletion(c.Request.Context(), sess.ModelVersion+"-asr", messages, req.Stream, extra)
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
		c.JSON(http.StatusOK, chatResp)
	}
}
