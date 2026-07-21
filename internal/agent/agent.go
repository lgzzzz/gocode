package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"github.com/lgzzzz/gocode/internal/tools"
)

// ---- Callback message types ----

// MsgType identifies the kind of callback message.
type MsgType string

const (
	MsgThinkingStream  MsgType = "thinking_stream"  // streaming thinking/reasoning content
	MsgAssistantStream MsgType = "assistant_stream" // streaming assistant content
	MsgToolCall        MsgType = "tool_call"        // tool call issued
	MsgToolResult      MsgType = "tool_result"      // tool execution result
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
	msgCount int                                      // counter for generating unique message IDs
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
func (a *Agent) Run(ctx context.Context, userMessage string, cb func(CallbackMsg)) (string, error) {
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
		a.msgCount++
		msgID := fmt.Sprintf("msg-%d", a.msgCount)

		fullContent, toolCalls, err := a.streamOne(ctx, messages, msgID, cb)
		if err != nil {
			return "", err
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
						if tc.Name == "bash" && (strings.HasPrefix(result, "exit ") || strings.HasPrefix(result, "(timed out")) {
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
		return fullContent, nil
	}
}

// streamOne runs a single streaming chat completion and returns the accumulated
// content and any tool calls. It calls cb with streaming deltas as they arrive.
// msgID is a unique identifier for this assistant turn, used to correlate
// streaming updates and allow the TUI to update components in-place.
func (a *Agent) streamOne(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	msgID string,
	cb func(CallbackMsg),
) (content string, toolCalls []toolCallAccum, err error) {
	stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:           a.model,
		Messages:        messages,
		Tools:           a.oaiTools,
		ReasoningEffort: "max",
	})
	if stream.Err() != nil {
		return "", nil, fmt.Errorf("API error: %w", stream.Err())
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
		return "", nil, fmt.Errorf("stream error: %w", stream.Err())
	}

	// Collect accumulated tool calls in index order
	for i := int64(0); i < int64(len(tcMap)); i++ {
		if acc, ok := tcMap[i]; ok {
			toolCalls = append(toolCalls, *acc)
		}
	}

	return fullContent.String(), toolCalls, nil
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

// ClearHistory resets the conversation history for a fresh session.
func (a *Agent) ClearHistory() {
	a.history = nil
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
