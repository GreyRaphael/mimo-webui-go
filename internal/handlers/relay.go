package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// relaySSEStream reads raw SSE data lines from the MiMo API response and
// forwards them directly to the browser without re-encoding.
// It also parses the JSON to accumulate content/reasoning for DB storage.
//
// Per MiMo API docs: each data: line is a complete JSON object with
// standard JSON escaping. No special multi-line SSE handling needed.
func relaySSEStream(w http.ResponseWriter, flusher http.Flusher, apiReader io.Reader, onContent func(string), onReasoning func(string)) {
	scanner := bufio.NewScanner(apiReader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		if len(data) > 0 && data[0] == ' ' {
			data = data[1:]
		}

		// Forward the raw data line to the browser exactly as received from the API.
		// The browser's EventSource or fetch SSE parser handles this natively.
		fmt.Fprintf(w, "data:%s\n\n", data)
		flusher.Flush()

		if data == "[DONE]" {
			return
		}

		// Parse JSON to extract content/reasoning for DB storage only.
		// Do NOT re-encode or manipulate the content.
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          *string `json:"content"`
					ReasoningContent *string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.ReasoningContent != nil && onReasoning != nil {
			onReasoning(*delta.ReasoningContent)
		}
		if delta.Content != nil && onContent != nil {
			onContent(*delta.Content)
		}
	}
}
