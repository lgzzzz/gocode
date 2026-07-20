package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"

	"github.com/lgzzzz/gocode/internal/tools"
)

// ---- Callback message types ----

// MsgType identifies the kind of callback message.
type MsgType string

const (
	MsgThinkingStream  MsgType = "thinking_stream"  // streaming thinking/reasoning content
	MsgAssistantStream MsgType = "assistant_stream" // streaming assistant content
	MsgAssistantDone   MsgType = "assistant_done"   // assistant streaming finished
	MsgThinking        MsgType = "thinking"         // non-streaming thinking block
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
}

// Agent implements a ReAct-style loop using OpenAI-compatible function calling.
type Agent struct {
	client   *openai.Client
	model    string
	tools    []openai.Tool
	toolDefs []tools.ToolDef             // tool definitions for system prompt generation
	toolMap  map[string]tools.ToolExecutor
	cwd      string
	history  []openai.ChatCompletionMessage // conversation history
	msgCount int                            // counter for generating unique message IDs
}

func New(apiKey, model, baseURL string) *Agent {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	client := openai.NewClientWithConfig(cfg)

	tm, defs := tools.AllTools()

	var oaiTools []openai.Tool
	for _, d := range defs {
		oaiTools = append(oaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  d.Parameters,
			},
		})
	}

	cwd, _ := os.Getwd()

	return &Agent{
		client:   client,
		model:    model,
		tools:    oaiTools,
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
		a.history = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: a.systemPrompt(),
			},
		}
	}

	// Append the new user message to history
	a.history = append(a.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})

	messages := a.history

	for {
		a.msgCount++
		msgID := fmt.Sprintf("msg-%d", a.msgCount)

		fullContent, toolCalls, err := a.streamOne(ctx, messages, msgID, cb)
		if err != nil {
			return "", err
		}

		// Signal end of streaming for the assistant message
		cb(CallbackMsg{Type: MsgAssistantDone, ID: msgID})

		// If the model wants to call tools
		if len(toolCalls) > 0 {
			// Build the assistant message that requested tool calls
			var oaiToolCalls []openai.ToolCall
			for _, tc := range toolCalls {
				oaiToolCalls = append(oaiToolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
			assistantMsg := openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   fullContent,
				ToolCalls: oaiToolCalls,
			}
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
				if !ok {
					result = fmt.Sprintf("Error: unknown tool '%s'", tc.Name)
				} else {
					res, err := tool.Execute(tc.Arguments)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					} else {
						result = res
					}
				}

				cb(CallbackMsg{
					Type:       MsgToolResult,
					ID:         tc.ID,
					Content:    result,
					ToolCallID: tc.ID,
				})

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			continue
		}

		// Final text response — append to history
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fullContent,
		})
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
	messages []openai.ChatCompletionMessage,
	msgID string,
	cb func(CallbackMsg),
) (content string, toolCalls []toolCallAccum, err error) {
	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: messages,
		Tools:    a.tools,
	})
	if err != nil {
		return "", nil, fmt.Errorf("API error: %w", err)
	}
	defer stream.Close()

	var (
		fullContent   strings.Builder
		fullReasoning strings.Builder
		tcMap         = make(map[int]*toolCallAccum) // index -> accumulator
	)

	for {
		response, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return "", nil, fmt.Errorf("stream error: %w", recvErr)
		}

		if len(response.Choices) == 0 {
			continue
		}
		delta := response.Choices[0].Delta

		// Handle reasoning_content (DeepSeek-style)
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
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
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

	// Collect accumulated tool calls in index order
	for i := 0; i < len(tcMap); i++ {
		if acc, ok := tcMap[i]; ok {
			toolCalls = append(toolCalls, *acc)
		}
	}

	// If there was reasoning but no content, show reasoning as thinking
	if fullContent.Len() == 0 && fullReasoning.Len() > 0 {
		cb(CallbackMsg{Type: MsgThinking, ID: msgID, Content: fullReasoning.String()})
	}

	return fullContent.String(), toolCalls, nil
}

// reasoningContent extracts reasoning_content from the delta.
// go-openai doesn't expose this field, so we use a type assertion
// against the raw map if available, or check a known embedded field.
func reasoningContent(delta openai.ChatCompletionStreamChoiceDelta) string {
	// The go-openai library may have ReasoningContent field in newer versions.
	// Try to access it via interface checks.
	// For now, this is a placeholder — DeepSeek puts reasoning_content in the
	// delta JSON alongside "content". Since go-openai v1.36.1 doesn't parse it,
	// we return empty. Streaming of regular content still works.
	return ""
}

// toolCallAccum accumulates a tool call from streaming deltas.
type toolCallAccum struct {
	ID        string
	Type      openai.ToolType
	Name      string
	Arguments string
}

// ClearHistory resets the conversation history for a fresh session.
func (a *Agent) ClearHistory() {
	a.history = nil
}

// systemPrompt builds the system prompt dynamically from the available tool definitions,
// similar to how pi constructs its prompt from tool metadata.
func (a *Agent) systemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are an AI coding agent that helps users with programming tasks. ")
	sb.WriteString("You operate in a ReAct (Reasoning + Acting) loop: think about what to do, use tools to act, observe results, and iterate.\n\n")

	// Build the tool list from tool definitions
	sb.WriteString("You have access to the following tools:\n")
	for _, t := range a.tools {
		if t.Function == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	// Collect guidelines from tool definitions and add common ones
	sb.WriteString("\nGuidelines:\n")
	guidelineNum := 1
	for _, t := range a.tools {
		if t.Function == nil {
			continue
		}
		// Look up the ToolDef to get PromptGuidelines
		for _, d := range a.toolDefs {
			if d.Name == t.Function.Name {
				for _, g := range d.PromptGuidelines {
					sb.WriteString(fmt.Sprintf("%d. %s\n", guidelineNum, g))
					guidelineNum++
				}
				break
			}
		}
	}
	sb.WriteString(fmt.Sprintf("%d. Work step by step: think → act → observe → decide\n", guidelineNum))
	guidelineNum++
	sb.WriteString(fmt.Sprintf("%d. Be concise. When done, summarize what you accomplished.\n", guidelineNum))

	sb.WriteString(fmt.Sprintf("\nCurrent working directory: %s", a.cwd))

	return sb.String()
}
