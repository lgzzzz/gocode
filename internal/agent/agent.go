package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"github.com/lgzzzz/gocode/internal/tools"
)

// ---- Callback message types ----

// MsgType identifies the kind of callback message.
type MsgType string

const (
	MsgThinkingStream  MsgType = "thinking_stream"  // streaming thinking/reasoning content (not persisted)
	MsgAssistantStream MsgType = "assistant_stream" // streaming assistant content (not persisted)
	MsgThinking        MsgType = "thinking"         // complete thinking/reasoning (persisted)
	MsgAssistant       MsgType = "assistant"        // complete assistant reply (persisted)
	MsgToolCall        MsgType = "tool_call"        // tool call issued
	MsgToolResult      MsgType = "tool_result"      // tool execution result
	MsgError           MsgType = "error"            // API call error (may be followed by retry)
	MsgRetryWait       MsgType = "retry_wait"       // waiting before retrying after an error
	MsgUser            MsgType = "user"
)

// CallbackMsg is passed to the progress callback during agent execution.
// It carries structured information about each event in the ReAct loop.
type CallbackMsg struct {
	Type       MsgType // event type
	ID         string  // message ID (for tool calls: tool call ID; for streaming: generated message ID)
	Content    string  // accumulated content (streaming) or full message
	ToolCallID string  // tool call ID (set for tool_call and tool_result)
	ToolName   string  // tool name (set for tool_call)
	ToolArgs   string  // tool arguments JSON (set for tool_call)
	Err        error   // tool execution error (set for tool_result)
}

// Agent implements a ReAct-style loop using OpenAI-compatible function calling.
type Agent struct {
	client   openai.Client
	model    string
	oaiTools []openai.ChatCompletionToolParam
	toolDefs []tools.ToolDef // tool definitions for system prompt generation
	toolMap  map[string]tools.ToolExecutor
	cwd      string
	history  []openai.ChatCompletionMessageParamUnion // conversation history
}

func New(apiKey, model, baseURL string) *Agent {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}
	client := openai.NewClient(opts...)

	tm, defs := tools.AllTools()

	var oaiTools []openai.ChatCompletionToolParam
	for _, d := range defs {
		oaiTools = append(oaiTools, openai.ChatCompletionToolParam{
			Type: "function",
			Function: shared.FunctionDefinitionParam{
				Name:        d.Name,
				Description: openai.String(d.Description),
				Parameters:  shared.FunctionParameters(d.Parameters.(map[string]any)),
			},
		})
	}

	cwd, _ := os.Getwd()

	return &Agent{
		client:   client,
		model:    model,
		oaiTools: oaiTools,
		toolDefs: defs,
		toolMap:  tm,
		cwd:      cwd,
	}
}

// Run executes the ReAct loop with streaming. The callback receives structured
// CallbackMsg values as events occur:
//
//   - MsgThinkingStream:  streaming reasoning/thinking content (accumulated)
//   - MsgAssistantStream: streaming assistant content (accumulated)
//   - MsgAssistantDone:   assistant streaming finished (before tool execution or return)
//   - MsgToolCall:        tool call issued (ToolCallID, ToolName, ToolArgs set)
//   - MsgToolResult:      tool execution result (Content, ToolCallID set)
//   - MsgThinking:        non-streaming thinking block
//
// It maintains conversation history across calls so the LLM can refer to previous messages.
func (a *Agent) Run(ctx context.Context, userMessage string, cb func(CallbackMsg)) {
	// Initialize history with system prompt on first run
	if len(a.history) == 0 {
		a.history = []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(a.systemPrompt()),
		}
	}

	// Append the new user message to history
	a.history = append(a.history, openai.UserMessage(userMessage))

	messages := a.history

	for {
		msgID := uuid.New().String()

		fullContent, _, toolCalls, err := a.streamOne(ctx, messages, msgID, cb)
		if err != nil {
			return
		}

		// If the model wants to call tools
		if len(toolCalls) > 0 {
			// Build the assistant message that requested tool calls
			var oaiToolCalls []openai.ChatCompletionMessageToolCallParam
			for _, tc := range toolCalls {
				oaiToolCalls = append(oaiToolCalls, openai.ChatCompletionMessageToolCallParam{
					ID:   tc.ID,
					Type: "function",
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
			assistantMsg := openai.AssistantMessage(fullContent)
			// Set tool calls on the assistant message
			assistantMsg.OfAssistant.ToolCalls = oaiToolCalls
			messages = append(messages, assistantMsg)

			for _, tc := range toolCalls {
				cb(CallbackMsg{
					Type:       MsgToolCall,
					ID:         tc.ID,
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
					ToolArgs:   tc.Arguments,
				})

				tool, ok := a.toolMap[tc.Name]
				var result string
				var toolErr error
				if !ok {
					result = fmt.Sprintf("Error: unknown tool '%s'", tc.Name)
					toolErr = fmt.Errorf("unknown tool: %s", tc.Name)
				} else {
					res, err := tool.Execute(tc.Arguments)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
						toolErr = err
					} else {
						result = res
						// Bash returns nil error even on non-zero exit; detect via result prefix.
						if (tc.Name == "bash" || tc.Name == "powershell") && (strings.HasPrefix(result, "exit ") || strings.HasPrefix(result, "(timed out")) {
							toolErr = fmt.Errorf("%s", strings.SplitN(result, "\n", 2)[0])
						}
					}
				}

				cb(CallbackMsg{
					Type:       MsgToolResult,
					ID:         tc.ID,
					Content:    result,
					ToolCallID: tc.ID,
					Err:        toolErr,
				})

				messages = append(messages, openai.ToolMessage(result, tc.ID))
			}
			continue
		}

		// Final text response — append to history
		messages = append(messages, openai.AssistantMessage(fullContent))
		a.history = messages
		return
	}
}

// streamOne runs a single streaming chat completion and returns the accumulated
// content and any tool calls. It calls cb with streaming deltas as they arrive.
// msgID is a unique identifier for this assistant turn, used to correlate
// streaming updates and allow the TUI to update components in-place.
//
// It retries up to maxRetries times on transient errors (e.g. network issues,
// rate limiting, server errors), with exponential backoff between attempts.
func (a *Agent) streamOne(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	msgID string,
	cb func(CallbackMsg),
) (content string, reasoning string, toolCalls []toolCallAccum, err error) {
	const maxRetries = 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, reasoning, toolCalls, err = a.streamOneAttempt(ctx, messages, msgID, cb)
		if err == nil {
			return content, reasoning, toolCalls, nil
		}

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			cb(CallbackMsg{Type: MsgError, Content: fmt.Sprintf("请求已取消: %v", ctx.Err())})
			return "", "", nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		// Last attempt failed — return the error
		if attempt == maxRetries {
			cb(CallbackMsg{Type: MsgError, Content: fmt.Sprintf("API 调用失败（已重试 %d 次）: %v", maxRetries, err)})
			return "", "", nil, fmt.Errorf("API call failed after %d retries: %w", maxRetries, err)
		}

		// Report the error and notify that we'll retry
		cb(CallbackMsg{Type: MsgError, Content: fmt.Sprintf("API 调用出错: %v", err)})

		cb(CallbackMsg{Type: MsgRetryWait, Content: fmt.Sprintf("将在 %.0f 秒后重试（第 %d/%d 次）...", baseDelay.Seconds(), attempt+1, maxRetries)})

		select {
		case <-ctx.Done():
			return "", "", nil, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		case <-time.After(baseDelay):
		}
	}

	return "", "", nil, fmt.Errorf("unreachable")
}

// streamOneAttempt performs a single streaming chat completion attempt.
func (a *Agent) streamOneAttempt(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	msgID string,
	cb func(CallbackMsg),
) (content string, reasoning string, toolCalls []toolCallAccum, err error) {
	stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:           a.model,
		Messages:        messages,
		Tools:           a.oaiTools,
		ReasoningEffort: "max",
	})
	if stream.Err() != nil {
		return "", "", nil, fmt.Errorf("API error: %w", stream.Err())
	}
	defer stream.Close()

	var (
		fullContent   strings.Builder
		fullReasoning strings.Builder
		tcMap         = make(map[int64]*toolCallAccum) // index -> accumulator
	)

	for stream.Next() {
		chunk := stream.Current()

		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		// Handle reasoning_content (DeepSeek-style) via raw JSON
		if rc := reasoningContent(delta); rc != "" {
			fullReasoning.WriteString(rc)
			cb(CallbackMsg{Type: MsgThinkingStream, ID: msgID, Content: fullReasoning.String()})
		}

		// Handle regular content
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			cb(CallbackMsg{Type: MsgAssistantStream, ID: msgID, Content: fullContent.String()})
		}

		// Handle tool calls (accumulate by index)
		for _, tc := range delta.ToolCalls {
			idx := tc.Index
			acc, ok := tcMap[idx]
			if !ok {
				acc = &toolCallAccum{}
				tcMap[idx] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function.Name != "" {
				acc.Name = tc.Function.Name
			}
			acc.Arguments += tc.Function.Arguments
		}
	}

	if stream.Err() != nil {
		return "", "", nil, fmt.Errorf("stream error: %w", stream.Err())
	}

	// Collect accumulated tool calls in index order
	for i := int64(0); i < int64(len(tcMap)); i++ {
		if acc, ok := tcMap[i]; ok {
			toolCalls = append(toolCalls, *acc)
		}
	}

	return fullContent.String(), fullReasoning.String(), toolCalls, nil
}

// reasoningContent extracts reasoning_content from the delta raw JSON.
// DeepSeek puts reasoning_content in the delta JSON alongside "content".
// The official OpenAI library doesn't parse it as a named field, so we
// extract it from the raw JSON.
func reasoningContent(delta openai.ChatCompletionChunkChoiceDelta) string {
	raw := delta.RawJSON()
	if raw == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if rc, ok := m["reasoning_content"]; ok {
		if s, ok := rc.(string); ok {
			return s
		}
	}
	return ""
}

// toolCallAccum accumulates a tool call from streaming deltas.
type toolCallAccum struct {
	ID        string
	Type      string
	Name      string
	Arguments string
}

// Model returns the model name used by this agent.
func (a *Agent) Model() string { return a.model }

// ClearHistory resets the conversation history for a fresh session.
func (a *Agent) ClearHistory() {
	a.history = nil
}

// SetHistory replaces the conversation history with the given messages.
// This is used when loading a previous session.
func (a *Agent) SetHistory(history []openai.ChatCompletionMessageParamUnion) {
	a.history = history
}

// SystemPrompt returns the system prompt string.
func (a *Agent) SystemPrompt() string {
	return a.systemPrompt()
}

// ---- history reconstruction ----

// HistoryMessage is a simplified representation of a persisted message,
// used by ReconstructHistory to rebuild the OpenAI conversation format.
type HistoryMessage struct {
	MsgType    string // uses MsgXxx constants (MsgUser, MsgAssistant, etc.)
	Content    string
	ToolCallID string
	ToolName   string
	ToolArgs   string
}

// ReconstructHistory rebuilds an OpenAI-compatible conversation history
// from persisted messages. thinking messages are skipped (not part of
// the OpenAI conversation format). tool_call / tool_result pairs are
// embedded into the preceding assistant message.
func ReconstructHistory(msgs []HistoryMessage, systemPrompt string) []openai.ChatCompletionMessageParamUnion {
	history := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	i := 0
	for i < len(msgs) {
		m := msgs[i]

		switch m.MsgType {
		case string(MsgUser):
			history = append(history, openai.UserMessage(m.Content))
			i++

		case string(MsgAssistant):
			// assistant: normal path — host for subsequent tool_calls.
			// tool_call: orphan path — no preceding assistant message
			//   (happens when model calls tools without outputting text).
			//   Create an empty assistant message as host.
			var toolCalls []openai.ChatCompletionMessageToolCallParam

			// If this is an assistant message, collect its content.
			// If this is an orphan tool_call, content stays empty.
			assistantContent := ""
			if m.MsgType == string(MsgAssistant) {
				assistantContent = m.Content
				i++ // consume assistant
			}

			// Collect tool_calls that immediately follow.
			for i < len(msgs) && msgs[i].MsgType == string(MsgToolCall) {
				tc := msgs[i]
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
					ID:   tc.ToolCallID,
					Type: "function",
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.ToolName,
						Arguments: tc.ToolArgs,
					},
				})
				i++
			}

			assistantMsg := openai.AssistantMessage(assistantContent)
			if len(toolCalls) > 0 {
				assistantMsg.OfAssistant.ToolCalls = toolCalls
			}
			history = append(history, assistantMsg)

			// Collect tool_results that follow.
			for i < len(msgs) && msgs[i].MsgType == string(MsgToolResult) {
				tr := msgs[i]
				history = append(history, openai.ToolMessage(tr.Content, tr.ToolCallID))
				i++
			}

		case string(MsgThinking):
			i++

		default:
			i++
		}
	}

	return history
}

// systemPrompt builds the system prompt dynamically from the available tool definitions,
// matching the structure and style of pi's system prompt construction.
func (a *Agent) systemPrompt() string {
	var sb strings.Builder

	// Opening line — matches pi's tone
	sb.WriteString(`You are an expert coding assistant called GoCode.
You help users by reading files, executing commands, editing code, and writing new files.`)

	// Build the tool list from tool definitions using one-line snippets (matching pi's style)
	sb.WriteString("Available tools:\n")
	for _, t := range a.oaiTools {
		// Look up the ToolDef to get the PromptSnippet
		snippet := t.Function.Description.Value // fallback to full description
		for _, d := range a.toolDefs {
			if d.Name == t.Function.Name && d.PromptSnippet != "" {
				snippet = d.PromptSnippet
				break
			}
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, snippet))
	}

	sb.WriteString("\nGuidelines:\n")
	seen := make(map[string]bool)
	for _, t := range a.oaiTools {
		for _, d := range a.toolDefs {
			if d.Name == t.Function.Name {
				for _, g := range d.PromptGuidelines {
					g = strings.TrimSpace(g)
					if g != "" && !seen[g] {
						seen[g] = true
						sb.WriteString(fmt.Sprintf("- %s\n", g))
					}
				}
				break
			}
		}
	}

	// Common guidelines — always appended, matching pi
	addCommonGuideline := func(g string) {
		g = strings.TrimSpace(g)
		if g != "" && !seen[g] {
			seen[g] = true
			sb.WriteString(fmt.Sprintf("- %s\n", g))
		}
	}
	addCommonGuideline("Work step by step: think → act → observe → decide")
	addCommonGuideline("Be concise. When done, summarize what you accomplished.")
	addCommonGuideline("Be concise in your responses")
	addCommonGuideline("Show file paths clearly when working with files")

	// Current working directory
	sb.WriteString(fmt.Sprintf("\nCurrent working directory: %s", a.cwd))

	return sb.String()
}
