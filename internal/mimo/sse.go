package mimo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// sseData is the wire format of a single SSE data payload.
type sseData struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Choices []sseChoice `json:"choices"`
}

type sseChoice struct {
	Delta        *sseDelta `json:"delta,omitempty"`
	FinishReason *string   `json:"finish_reason,omitempty"`
}

type sseDelta struct {
	Content          *string        `json:"content,omitempty"`
	ReasoningContent *string        `json:"reasoning_content,omitempty"`
	Audio            *sseAudioChunk `json:"audio,omitempty"`
}

type sseAudioChunk struct {
	Data string `json:"data"`
}

// ProcessSSEStream reads an SSE stream from reader, parses each "data: ..."
// line into a typed SSEEvent, and sends it on the events channel.
// The channel is closed when the stream ends or a [DONE] marker is received.
func ProcessSSEStream(reader io.Reader, events chan<- SSEEvent) {
	defer close(events)

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		// Careful: only strip "data:" prefix + one optional space.
		// Do NOT use TrimSpace — it would strip trailing spaces from content.
		data := strings.TrimPrefix(line, "data:")
		if len(data) > 0 && data[0] == ' ' {
			data = data[1:]
		}

		if data == "[DONE]" {
			events <- SSEEvent{Type: "done"}
			return
		}

		var chunk sseData
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta == nil {
			continue
		}

		// Send reasoning content as a separate event type
		if delta.ReasoningContent != nil {
			events <- SSEEvent{Type: "reasoning", Content: *delta.ReasoningContent}
		}

		// Send actual content — even a space " " is meaningful
		if delta.Content != nil {
			events <- SSEEvent{Type: "message", Content: *delta.Content}
			fmt.Fprintf(os.Stderr, "[SSE-DEBUG] content=%q len=%d\n", *delta.Content, len(*delta.Content))
		}

		// Audio data
		if delta.Audio != nil && delta.Audio.Data != "" {
			events <- SSEEvent{Type: "audio", Content: delta.Audio.Data}
		}
	}
}
