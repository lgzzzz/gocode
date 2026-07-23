// Package openai provides a lightweight OpenAI-compatible API client.
// It supports chat completions with streaming and DeepSeek reasoning_content.
package openai

// ---- message types ----

// Message represents a chat completion message (request and response).
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // DeepSeek reasoning
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
}

// ToolCall represents a function call requested by the model.
// The Index field is only present in streaming deltas and is used
// to accumulate fragments by position.
type ToolCall struct {
	Index    int64        `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function,omitempty"`
}

// FunctionCall holds the name and arguments of a function call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ---- tool definition ----

// Tool defines a function tool for the chat completion request.
type Tool struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes a function's interface for the API.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ---- request ----

// ChatCompletionRequest is the request body for chat completions.
type ChatCompletionRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	Tools           []Tool    `json:"tools,omitempty"`
	Stream          bool      `json:"stream,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"` // DeepSeek: "max"
}

// ---- streaming response ----

// ChatCompletionChunk is a single chunk from a streaming response.
type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a choice in the completion response.
type Choice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta represents the delta in a streaming chunk.
type Delta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // DeepSeek
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ---- errors ----

// APIError represents an API error response.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// ---- message constructors ----

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// ToolMessage creates a tool result message.
func ToolMessage(content, toolCallID string) Message {
	return Message{Role: "tool", Content: content, ToolCallID: toolCallID}
}
