package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is a lightweight OpenAI-compatible API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new API client.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
	}
}

// StreamChunk wraps a received chunk together with an optional error.
type StreamChunk struct {
	Chunk ChatCompletionChunk
	Err   error
}

// StreamChatCompletion sends a streaming chat completion request and returns
// a channel of chunks. The channel is closed when the stream ends.
func (c *Client) StreamChatCompletion(
	ctx context.Context,
	req ChatCompletionRequest,
) <-chan StreamChunk {
	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)

		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("marshal request: %w", err)}
			return
		}

		httpReq, err := http.NewRequestWithContext(
			ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body),
		)
		if err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("create request: %w", err)}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := c.http.Do(httpReq)
		if err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("http request: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
			ch <- StreamChunk{
				Err: fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body)),
			}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		// Some providers emit long lines (tool calls). 1 MB max.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				return
			}

			var chunk ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("parse chunk: %w", err)}
				return
			}

			if chunk.Error != nil {
				ch <- StreamChunk{
					Err: fmt.Errorf("API error: %s (%s)", chunk.Error.Message, chunk.Error.Type),
				}
				return
			}

			ch <- StreamChunk{Chunk: chunk}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("read stream: %w", err)}
		}
	}()

	return ch
}
