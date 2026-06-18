package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/db"
	"mimo-webui/internal/middleware"
	"mimo-webui/internal/mimo"
)

// imageRequest is the JSON body for the image understanding endpoint.
type imageRequest struct {
	FileID string `json:"file_id"`
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ImageUnderstandingHandler returns a handler that sends an image to the MiMo
// API for understanding, optionally streaming the response via SSE.
func ImageUnderstandingHandler(database *db.DB, uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetAuthUser(c)
		sess, err := getMiMoSession(database, user.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var req imageRequest
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
			Type:     "image_url",
			ImageURL: &mimo.ImageURLObj{URL: url},
		}
		textPart := mimo.ContentPart{Type: "text", Text: req.Prompt}

		systemMsg := mimo.Message{Role: "system", Content: mimo.TextContent("You are a helpful assistant that can understand images.")}
		userMsg := mimo.Message{Role: "user", Content: mimo.MultiContent([]mimo.ContentPart{mediaPart, textPart})}
		messages := []mimo.Message{systemMsg, userMsg}

		resp, err := sess.Client.ChatCompletion(c.Request.Context(), sess.ModelVersion, messages, req.Stream, nil)
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

// findUploadPath scans the uploadDir for a file whose name starts with fileID
// and returns the full path, or "" if not found.
func findUploadPath(uploadDir, fileID string) string {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), fileID) {
			return filepath.Join(uploadDir, e.Name())
		}
	}
	return ""
}
