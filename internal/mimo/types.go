package mimo

import "encoding/json"

// Message represents a chat message sent to the API.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// TextContent creates a JSON-encoded string suitable for Message.Content.
func TextContent(text string) json.RawMessage {
	b, _ := json.Marshal(text)
	return b
}

// MultiContent creates a JSON-encoded array of ContentPart suitable for Message.Content.
func MultiContent(parts []ContentPart) json.RawMessage {
	b, _ := json.Marshal(parts)
	return b
}

// ContentPart represents a single part in a multimodal content array.
type ContentPart struct {
	Type       string        `json:"type"`
	Text       string        `json:"text,omitempty"`
	ImageURL   *ImageURLObj  `json:"image_url,omitempty"`
	InputAudio *InputAudioObj `json:"input_audio,omitempty"`
	VideoURL   *VideoURLObj  `json:"video_url,omitempty"`
	FPS        *float64      `json:"fps,omitempty"`
	MediaRes   *string       `json:"media_resolution,omitempty"`
}

// ImageURLObj wraps an image URL for multimodal requests.
type ImageURLObj struct {
	URL string `json:"url"`
}

// InputAudioObj carries inline audio data.
type InputAudioObj struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

// VideoURLObj wraps a video URL for multimodal requests.
type VideoURLObj struct {
	URL string `json:"url"`
}

// AsrOptions controls automatic speech recognition behaviour.
type AsrOptions struct {
	Language string `json:"language,omitempty"`
}

// TtsAudioConfig controls text-to-speech output format.
type TtsAudioConfig struct {
	Format string  `json:"format"`
	Voice  *string `json:"voice,omitempty"`
}

// ---- Response types ----

// ChatResponse is the top-level response from /chat/completions.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents one completion choice.
type Choice struct {
	Message      *AssistantMessage `json:"message,omitempty"`
	Delta        *AssistantMessage `json:"delta,omitempty"`
	FinishReason *string           `json:"finish_reason,omitempty"`
}

// AssistantMessage holds the assistant's reply content.
type AssistantMessage struct {
	Content          *string    `json:"content,omitempty"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
	Audio            *AudioData `json:"audio,omitempty"`
}

// AudioData carries base64-encoded audio in a response.
type AudioData struct {
	Data string `json:"data"`
}

// Usage reports token counts for a request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SSEEvent represents a single server-sent event parsed from the stream.
type SSEEvent struct {
	Type    string // e.g. "message", "error", "done"
	Content string // raw JSON data string
	Audio   []byte // decoded audio bytes when present
}
