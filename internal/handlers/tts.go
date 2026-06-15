package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"mimo-webui/internal/mimo"
)

// ttsRequest is the JSON body for the TTS endpoint.
type ttsRequest struct {
	Text             string `json:"text"`
	StyleInstruction string `json:"style_instruction"`
	VoiceDescription string `json:"voice_description"`
	SampleFileID     string `json:"sample_file_id"`
	Voice            string `json:"voice"`
	AudioFormat      string `json:"audio_format"`
	ModelVariant     string `json:"model_variant"`
	Stream           bool   `json:"stream"`
}

// TTSHandler returns a handler that sends text to the MiMo API for
// text-to-speech synthesis, optionally streaming the response via SSE.
func TTSHandler(client *mimo.Client, uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ttsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		if req.Text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
			return
		}

		// Apply defaults.
		if req.AudioFormat == "" {
			req.AudioFormat = "wav"
		}
		if req.ModelVariant == "" {
			req.ModelVariant = "preset"
		}

		// For voice cloning, read the sample audio and base64-encode it.
		var sampleAudioData string
		if req.ModelVariant == "clone" && req.SampleFileID != "" {
			path := findUploadPath(uploadDir, req.SampleFileID)
			if path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "sample file not found for id: " + req.SampleFileID})
				return
			}
			dataURI, err := mimo.FileToBase64DataURI(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "read sample file: " + err.Error()})
				return
			}
			sampleAudioData = dataURI
		}

		ttsReq := mimo.TTSRequest{
			Text:             req.Text,
			StyleInstruction: req.StyleInstruction,
			VoiceDescription: req.VoiceDescription,
			SampleAudioData:  sampleAudioData,
			Voice:            req.Voice,
			AudioFormat:      req.AudioFormat,
			ModelVariant:     req.ModelVariant,
			Stream:           req.Stream,
		}

		resp, err := client.TTSCompletion(c.Request.Context(), ttsReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tts completion: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		if req.Stream {
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

			relaySSEStream(w, flusher, resp.Body, nil, nil)
			return
		}

		// Non-streaming: decode the full ChatResponse and extract audio.
		var chatResp mimo.ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "decode response: " + err.Error()})
			return
		}
		if resp.StatusCode != http.StatusOK {
			c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("api returned status %d", resp.StatusCode)})
			return
		}

		// Extract audio data from the response.
		if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message == nil || chatResp.Choices[0].Message.Audio == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no audio data in response"})
			return
		}

		audioB64 := chatResp.Choices[0].Message.Audio.Data
		audioBytes, err := base64.StdEncoding.DecodeString(audioB64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "decode audio: " + err.Error()})
			return
		}

		contentType := "audio/wav"
		if req.AudioFormat == "mp3" {
			contentType = "audio/mpeg"
		} else if req.AudioFormat == "pcm" {
			contentType = "audio/pcm"
		}

		c.Data(http.StatusOK, contentType, audioBytes)
	}
}
