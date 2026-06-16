package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mimo-webui/internal/db"
	"mimo-webui/internal/middleware"
	"mimo-webui/internal/mimo"
)

// CreateSession creates a new chat session.
func CreateSession(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title string `json:"title"`
			Model string `json:"model"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		if req.Model == "" {
			req.Model = "mimo-v2.5"
		}

		user := middleware.GetAuthUser(c)
		id := uuid.New().String()

		var title *string
		if req.Title != "" {
			title = &req.Title
		}

		if err := db.CreateSession(c.Request.Context(), database, id, user.UserID, title, req.Model); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":    id,
			"title": title,
			"model": req.Model,
		})
	}
}

// ListSessions returns all sessions for the authenticated user.
func ListSessions(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetAuthUser(c)

		sessions, err := db.ListSessions(c.Request.Context(), database, user.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
			return
		}

		c.JSON(http.StatusOK, sessions)
	}
}

// DeleteSession deletes a session owned by the authenticated user.
func DeleteSession(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		user := middleware.GetAuthUser(c)

		if err := db.DeleteSession(c.Request.Context(), database, sessionID, user.UserID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// GenerateTitle calls MiMo API to generate a concise session title based on
// the first user message, then updates the session in DB.
func GenerateTitle(database *db.DB, client *mimo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		user := middleware.GetAuthUser(c)

		session, err := db.GetSession(c.Request.Context(), database, sessionID)
		if err != nil || session == nil || session.UserID != user.UserID {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		// Get the first user message
		msgs, err := db.ListMessages(c.Request.Context(), database, sessionID, 5)
		if err != nil || len(msgs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no messages"})
			return
		}

		var firstUserMsg string
		for _, m := range msgs {
			if m.Role == "user" && m.Content != nil && *m.Content != "" {
				firstUserMsg = *m.Content
				break
			}
		}
		if firstUserMsg == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no user message found"})
			return
		}

		// Call MiMo API to generate title
		prompt := "请根据以下对话内容生成一个简短的标题（不超过15个字），只输出标题本身，不要任何标点符号或额外文字：\n\n" + firstUserMsg
		messages := []mimo.Message{
			{Role: "user", Content: mimo.TextContent(prompt)},
		}

		resp, err := client.ChatCompletion(c.Request.Context(), "mimo-v2.5", messages, false, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate title"})
			return
		}
		defer resp.Body.Close()

		var chatResp mimo.ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
			return
		}

		title := ""
		if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message != nil && chatResp.Choices[0].Message.Content != nil {
			title = strings.TrimSpace(*chatResp.Choices[0].Message.Content)
			// Remove surrounding quotes if any
			title = strings.Trim(title, "\"'「」\u201c\u201d\u2018\u2019")
		}

		if title == "" {
			// Fallback: use first 15 chars of user message
			runes := []rune(firstUserMsg)
			if len(runes) > 15 {
				title = string(runes[:15]) + "..."
			} else {
				title = firstUserMsg
			}
		}

		titlePtr := &title
		if err := db.UpdateSessionTitle(c.Request.Context(), database, sessionID, titlePtr); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update title"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"title": title})
	}
}

// SendMessage sends a message to a chat session and returns the assistant's response.
func SendMessage(database *db.DB, client *mimo.Client, uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")

		var req struct {
			Content   string `json:"content"`
			MediaURL  string `json:"media_url"`
			MediaType string `json:"media_type"`
			Stream    bool   `json:"stream"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		user := middleware.GetAuthUser(c)

		// Verify session ownership.
		session, err := db.GetSession(c.Request.Context(), database, sessionID)
		if err != nil || session == nil || session.UserID != user.UserID {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		// Save user message to DB.
		var contentPtr, mediaTypePtr, mediaURLPtr *string
		if req.Content != "" {
			contentPtr = &req.Content
		}
		if req.MediaType != "" {
			mediaTypePtr = &req.MediaType
		}
		// The frontend sends media_url = file_id (UUID).
		// Save web-accessible URL for display, and resolve actual path for API.
		if req.MediaURL != "" {
			apiPath := "/api/media/" + req.MediaURL
			mediaURLPtr = &apiPath
		}

		if _, err := db.CreateMessage(c.Request.Context(), database, sessionID, "user", contentPtr, mediaTypePtr, mediaURLPtr, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message"})
			return
		}

		// Load recent messages from DB to build conversation context.
		recentMsgs, err := db.ListMessages(c.Request.Context(), database, sessionID, 20)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load messages"})
			return
		}

		// Build mimo.Message slice with system prompt.
		systemMsg := mimo.Message{
			Role:    "system",
			Content: mimo.TextContent("You are MiMo, an AI assistant developed by Xiaomi."),
		}
		messages := []mimo.Message{systemMsg}

		for _, m := range recentMsgs {
			msg := mimo.Message{Role: m.Role}
			if m.MediaURL != nil && *m.MediaURL != "" {
				// Build multimodal content parts.
				var parts []mimo.ContentPart
				if m.Content != nil && *m.Content != "" {
					parts = append(parts, mimo.ContentPart{Type: "text", Text: *m.Content})
				}

				mediaType := ""
				if m.MediaType != nil {
					mediaType = *m.MediaType
				}

				// Convert local file paths to base64 data URIs
				mediaURL := *m.MediaURL
				// Handle /api/media/{file_id} paths (stored in DB for display)
				if strings.HasPrefix(mediaURL, "/api/media/") {
					fileID := strings.TrimPrefix(mediaURL, "/api/media/")
					if actualPath := findUploadPath(uploadDir, fileID); actualPath != "" {
						mediaURL = actualPath
					}
				}
				// Convert local file paths to base64 data URIs
				if !strings.HasPrefix(mediaURL, "http") && !strings.HasPrefix(mediaURL, "data:") {
					if dataURI, err := mimo.FileToBase64DataURI(mediaURL); err == nil {
						mediaURL = dataURI
					}
				}

				switch {
				case mediaType == "image":
					parts = append(parts, mimo.ContentPart{
						Type:     "image_url",
						ImageURL: &mimo.ImageURLObj{URL: mediaURL},
					})
				case mediaType == "audio":
					parts = append(parts, mimo.ContentPart{
						Type:       "input_audio",
						InputAudio: &mimo.InputAudioObj{Data: mediaURL},
					})
				case mediaType == "video":
					parts = append(parts, mimo.ContentPart{
						Type:     "video_url",
						VideoURL: &mimo.VideoURLObj{URL: mediaURL},
					})
				}

				msg.Content = mimo.MultiContent(parts)
			} else if m.Content != nil {
				msg.Content = mimo.TextContent(*m.Content)
			}
			messages = append(messages, msg)
		}

		// If the current request has media but wasn't in the last DB message (edge case), handle inline.
		// The user message was already saved above, so it's already in recentMsgs.

		// Call MiMo API with background context (detached from browser connection).
		// If browser navigates away mid-stream, the API continues and message is saved to DB.
		resp, err := client.ChatCompletion(context.Background(), session.Model, messages, req.Stream, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call AI API"})
			return
		}
		defer resp.Body.Close()

		if req.Stream {
			// SSE streaming — relay raw API data lines directly to browser.
			// Per MiMo API docs, each data: line is a complete JSON object.
			w := c.Writer
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
				return
			}

			var fullContent string
			var fullReasoning string

			relaySSEStream(w, flusher, resp.Body,
				func(content string) { fullContent += content },
				func(reasoning string) { fullReasoning += reasoning },
			)

			// Save assistant message after stream completes
			if fullContent != "" {
				assistantContent := fullContent
				var reasoningPtr *string
				if fullReasoning != "" {
					reasoningPtr = &fullReasoning
				}
				db.CreateMessage(context.Background(), database, sessionID, "assistant", &assistantContent, nil, nil, reasoningPtr)
			}
		} else {
			// Non-streaming response.
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
				return
			}

			var chatResp mimo.ChatResponse
			if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
				return
			}

			var assistantContent string
			if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message != nil && chatResp.Choices[0].Message.Content != nil {
				assistantContent = *chatResp.Choices[0].Message.Content
			}

			if assistantContent != "" {
				db.CreateMessage(c.Request.Context(), database, sessionID, "assistant", &assistantContent, nil, nil, nil)
			}

			c.JSON(http.StatusOK, chatResp)
		}
	}
}

// ListMessages returns all messages for a session.
func ListMessages(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		user := middleware.GetAuthUser(c)

		// Verify session ownership before returning messages
		session, err := db.GetSession(c.Request.Context(), database, sessionID)
		if err != nil || session == nil || session.UserID != user.UserID {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		msgs, err := db.ListMessages(c.Request.Context(), database, sessionID, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list messages"})
			return
		}

		c.JSON(http.StatusOK, msgs)
	}
}
