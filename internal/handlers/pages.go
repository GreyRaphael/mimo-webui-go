package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/middleware"
)

// ChatPage renders the chat page without a specific session.
func ChatPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/chat.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}

// ChatSessionPage renders the chat page with a specific session ID.
func ChatSessionPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/chat.html", gin.H{
			"user":       middleware.GetAuthUser(c),
			"session_id": c.Param("id"),
		})
	}
}

// ImagePage renders the image generation page.
func ImagePage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/image.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}

// AudioPage renders the audio page.
func AudioPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/audio.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}

// VideoPage renders the video page.
func VideoPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/video.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}

// ASRPage renders the automatic speech recognition page.
func ASRPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/asr.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}

// TTSPage renders the text-to-speech page.
func TTSPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "pages/tts.html", gin.H{
			"user": middleware.GetAuthUser(c),
		})
	}
}
