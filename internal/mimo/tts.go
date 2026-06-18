package mimo

import (
	"context"
	"net/http"
)

// TTSRequest holds all parameters for a text-to-speech API call.
type TTSRequest struct {
	Text             string // assistant content (target text)
	StyleInstruction string // user content (style instruction, optional for preset)
	VoiceDescription string // user content (voice description, required for voicedesign)
	SampleAudioData  string // user content (base64 data URI for voiceclone)
	Voice            string // voice ID for preset mode
	AudioFormat      string // "wav" or "pcm16"
	ModelVariant     string // "preset" | "design" | "clone"
	ModelVersion     string // base model prefix, e.g. "mimo-v2.5"
	Stream           bool
}

// TTSCompletion sends a text-to-speech request via the /chat/completions endpoint.
func (c *Client) TTSCompletion(ctx context.Context, req TTSRequest) (*http.Response, error) {
	messages := make([]Message, 0, 2)

	switch req.ModelVariant {
	case "clone":
		// voiceclone: audio sample goes to audio.voice only, NOT in messages
		// user message is optional style instruction
		if req.StyleInstruction != "" {
			messages = append(messages, Message{
				Role:    "user",
				Content: TextContent(req.StyleInstruction),
			})
		}
	case "design":
		// voicedesign: user = voice description (required), assistant = target text
		messages = append(messages, Message{
			Role:    "user",
			Content: TextContent(req.VoiceDescription),
		})
	default:
		// preset: user = style instruction (optional), assistant = target text
		if req.StyleInstruction != "" {
			messages = append(messages, Message{
				Role:    "user",
				Content: TextContent(req.StyleInstruction),
			})
		}
	}

	// assistant message = target text
	messages = append(messages, Message{
		Role:    "assistant",
		Content: TextContent(req.Text),
	})

	// Determine model name
	model := req.ModelVersion + "-tts"
	switch req.ModelVariant {
	case "design":
		model = req.ModelVersion + "-tts-voicedesign"
	case "clone":
		model = req.ModelVersion + "-tts-voiceclone"
	}

	// Audio configuration
	audioCfg := map[string]any{
		"format": req.AudioFormat,
	}
	if req.ModelVariant == "preset" && req.Voice != "" {
		audioCfg["voice"] = req.Voice
	} else if req.ModelVariant == "clone" {
		// voiceclone requires audio.voice to be the reference audio DataURL
		if req.SampleAudioData != "" {
			audioCfg["voice"] = req.SampleAudioData
		}
	}

	extra := map[string]any{
		"audio": audioCfg,
	}

	return c.ChatCompletion(ctx, model, messages, req.Stream, extra)
}
