package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	goopenai "github.com/sashabaranov/go-openai"

	"github.com/lgzzzz/gocode/internal/tools"
)

// Logger is the logging interface used by Agent.
// Implementations must be safe for concurrent use.
type Logger interface {
	Printf(format string, v ...interface{})
}

// nopLogger is the default no-op logger.
type nopLogger struct{}

func (nopLogger) Printf(string, ...interface{}) {}

// ---- callback message types ----

// MsgType identifies the kind of callback message.
type MsgType string

const (
	MsgThinkingStream  MsgType = "thinking_stream"  // streaming thinking/reasoning content (not persisted)
	MsgAssistantStream MsgType = "assistant_stream" // streaming assistant content (not persisted)
	MsgThinking        MsgType = "thinking"         // complete thinking/reasoning (persisted)
	MsgAssistant       MsgType = "assistant"        // complete assistant reply (persisted)

	MsgToolCall   MsgType = "tool_call"   // tool call issued
	MsgToolResult MsgType = "tool_result" // tool execution result

	MsgError     MsgType = "error"      // API call error (may be followed by retry)
	MsgRetryWait MsgType = "retry_wait" // waiting before retrying after an error

	MsgUser MsgType = "user"
)

// CallbackMsg is passed to the progress callback during agent execution.
type CallbackMsg struct {
	ID         string  // message ID
	Type       MsgType // event type
	Content    string
	ToolCallID string // tool call ID (set for tool_call and tool_result)
	ToolName   string // tool name (set for tool_call)
	ToolArgs   string // tool arguments JSON (set for tool_call)
	ToolErr    error  // tool execution error (set for tool_result)
}

// Agent implements a ReAct-style loop using OpenAI-compatible function calling.
type Agent struct {
	client          *goopenai.Client
	model           string
	oaiTools        []goopenai.Tool
	toolDefs        []tools.ToolDef // tool definitions for system prompt generation
	toolMap         map[string]tools.ToolExecutor
	cwd             string
	contextMessages []goopenai.ChatCompletionMessage // conversation contextMessages
	logger          Logger                           // optional logger (defaults to no-op)
}

// New creates a new Agent with the given API key, model, and base URL.
// If baseURL is empty, defaults to DeepSeek API.
func New(apiKey, model, baseURL string) *Agent {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	config := goopenai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := goopenai.NewClientWithConfig(config)

	tm, defs := tools.AllTools()

	var oaiTools []goopenai.Tool
	for _, d := range defs {
		oaiTools = append(oaiTools, goopenai.Tool{
			Type: goopenai.ToolTypeFunction,
			Function: &goopenai.FunctionDefinition{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  d.Parameters.(map[string]any),
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
		logger:   nopLogger{},
	}
}

// Run executes the ReAct loop with streaming. The callback receives structured
// CallbackMsg values as events occur.
//
// It maintains conversation history across calls so the LLM can refer to
// previous messages. reasoning_content is round-tripped: returned from the
// API and included back in assistant messages on subsequent requests.
func (a *Agent) Run(ctx context.Context, userMessage string, cb func(CallbackMsg)) {
	// Initialize history with system prompt on first run
	if len(a.contextMessages) == 0 {
		a.contextMessages = []goopenai.ChatCompletionMessage{
			sysMsg(a.systemPrompt()),
		}
	}

	// Append the new user message to history
	a.contextMessages = append(a.contextMessages, userMsg(userMessage))

	messages := a.contextMessages

	for {
		fullContent, fullReasoning, toolCalls, err := a.streamOne(ctx, messages, cb)
		if err != nil {
			return
		}
		// If the model wants to call tools
		if len(toolCalls) > 0 {
			// Build the assistant message that requested tool calls
			assistantMsg := asstMsg(fullContent)
			assistantMsg.ReasoningContent = fullReasoning
			assistantMsg.ToolCalls = toolCalls

			messages = append(messages, assistantMsg)

			for _, tc := range toolCalls {
				toolMsgId := uuid.NewString()
				cb(CallbackMsg{
					Type:       MsgToolCall,
					ID:         toolMsgId,
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
					ToolArgs:   tc.Function.Arguments,
				})

				tool, ok := a.toolMap[tc.Function.Name]
				var result string
				var toolErr error
				if !ok {
					result = fmt.Sprintf("Error: unknown tool '%s'", tc.Function.Name)
					toolErr = fmt.Errorf("unknown tool: %s", tc.Function.Name)
				} else {
					res, err := tool.Execute(tc.Function.Arguments)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
						toolErr = err
					} else {
						result = res
						// Shell tools return nil error even on non-zero exit;
						// detect via result prefix.
						if (tc.Function.Name == "bash" || tc.Function.Name == "powershell") &&
							(strings.HasPrefix(result, "exit ") || strings.HasPrefix(result, "(timed out")) {
							toolErr = fmt.Errorf("%s", strings.SplitN(result, "\n", 2)[0])
						}
					}
				}

				cb(CallbackMsg{
					Type:       MsgToolResult,
					ID:         toolMsgId,
					ToolCallID: tc.ID,
					Content:    result,
					ToolErr:    toolErr,
				})

				messages = append(messages, toolMsg(result, tc.ID))
			}
			continue
		}

		// Final text response — append to history (with reasoning_content)
		assistantMsg := asstMsg(fullContent)
		assistantMsg.ReasoningContent = fullReasoning
		messages = append(messages, assistantMsg)
		a.contextMessages = messages
		return
	}
}

// streamOne runs a single streaming chat completion and returns the accumulated
// content, reasoning, and any tool calls.
//
// It retries up to maxRetries times on transient errors, with exponential
// backoff between attempts.
func (a *Agent) streamOne(
	ctx context.Context,
	messages []goopenai.ChatCompletionMessage,
	cb func(CallbackMsg),
) (content string, reasoning string, toolCalls []goopenai.ToolCall, err error) {
	const maxRetries = 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, reasoning, toolCalls, err = a.streamOneAttempt(ctx, messages, cb)
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
	messages []goopenai.ChatCompletionMessage,
	cb func(CallbackMsg),
) (content string, reasoning string, toolCalls []goopenai.ToolCall, err error) {
	req := goopenai.ChatCompletionRequest{
		Model:           a.model,
		Messages:        messages,
		Tools:           a.oaiTools,
		Stream:          true,
		ReasoningEffort: "max",
	}

	stream, err := a.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", "", nil, fmt.Errorf("create stream: %w", err)
	}
	defer stream.Close()

	var (
		fullContent   strings.Builder
		fullReasoning strings.Builder
		tcMap         = make(map[int]*toolCallAccum) // index -> accumulator
	)
	fullContentId := uuid.NewString()
	fullReasoningId := uuid.NewString()
	for {
		resp, recvErr := stream.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				break
			}
			return "", "", nil, fmt.Errorf("stream recv: %w", recvErr)
		}

		if len(resp.Choices) == 0 {
			continue
		}
		delta := resp.Choices[0].Delta

		// Handle reasoning_content (DeepSeek)
		if delta.ReasoningContent != "" {
			fullReasoning.WriteString(delta.ReasoningContent)
			cb(CallbackMsg{Type: MsgThinkingStream, ID: fullReasoningId, Content: fullReasoning.String()})
		}

		// Handle regular content
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			cb(CallbackMsg{Type: MsgAssistantStream, ID: fullContentId, Content: fullContent.String()})
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
				acc.Type = string(tc.Type)
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
			toolCalls = append(toolCalls, goopenai.ToolCall{
				ID:   acc.ID,
				Type: goopenai.ToolType(acc.Type),
				Function: goopenai.FunctionCall{
					Name:      acc.Name,
					Arguments: acc.Arguments,
				},
			})
		}
	}
	if fullReasoning.Len() != 0 {
		cb(CallbackMsg{Type: MsgThinking, ID: fullReasoningId, Content: fullReasoning.String()})
	}
	if fullContent.Len() != 0 {
		cb(CallbackMsg{Type: MsgAssistant, ID: fullContentId, Content: fullContent.String()})
	}
	return fullContent.String(), fullReasoning.String(), toolCalls, nil
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

// SetLogger sets the logger used by the agent. Passing nil resets to no-op.
func (a *Agent) SetLogger(l Logger) {
	if l == nil {
		l = nopLogger{}
	}
	a.logger = l
}

// ClearContextMessage resets the conversation history for a fresh session.
func (a *Agent) ClearContextMessage() {
	a.contextMessages = nil
}

// SetContextMessage replaces the conversation history with the given messages.
func (a *Agent) SetContextMessage(contextMessages []goopenai.ChatCompletionMessage) {
	a.contextMessages = contextMessages
}

// SystemPrompt returns the system prompt string.
func (a *Agent) SystemPrompt() string {
	return a.systemPrompt()
}

// ---- history reconstruction ----

// HistoryMessage is a simplified representation of a persisted message,
// used by ReconstructHistory to rebuild the OpenAI conversation format.
type HistoryMessage struct {
	MsgType    string // uses MsgXxx constants
	Content    string
	ToolCallID string
	ToolName   string
	ToolArgs   string
}

// ReconstructHistory rebuilds an OpenAI-compatible conversation history
// from persisted messages. thinking messages are merged into the following
// assistant message via ReasoningContent, and tool_call messages are grouped
// into the preceding assistant message as ToolCalls.
func ReconstructHistory(msgs []HistoryMessage, systemPrompt string) []goopenai.ChatCompletionMessage {
	var result []goopenai.ChatCompletionMessage

	// Always start with the system prompt.
	result = append(result, goopenai.ChatCompletionMessage{
		Role:    goopenai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	var pendingAssistant = goopenai.ChatCompletionMessage{
		Role: goopenai.ChatMessageRoleAssistant,
	}
	addPendingAssistant := func() {
		if pendingAssistant.Content != "" || pendingAssistant.ReasoningContent != "" {
			result = append(result, pendingAssistant)
			pendingAssistant = goopenai.ChatCompletionMessage{
				Role: goopenai.ChatMessageRoleAssistant,
			}
		}
	}
	var pendingToolResult []goopenai.ChatCompletionMessage
	addPendingToolResult := func() {
		result = append(result, pendingToolResult...)
		pendingToolResult = nil
	}
	i := 0
	for i < len(msgs) {
		msg := msgs[i]
		switch msg.MsgType {
		case string(MsgUser):
			addPendingAssistant()
			result = append(result, goopenai.ChatCompletionMessage{
				Role:    goopenai.ChatMessageRoleUser,
				Content: msg.Content,
			})
			i++
		case string(MsgThinking):
			if pendingAssistant.ReasoningContent != "" {
				addPendingAssistant()
				addPendingToolResult()
			}
			pendingAssistant.ReasoningContent = msg.Content
			i++
		case string(MsgAssistant):
			if pendingAssistant.Content != "" {
				addPendingAssistant()
				addPendingToolResult()
			}
			pendingAssistant.Content = msg.Content
			i++
		case string(MsgToolCall):
			if pendingAssistant.Content == "" && pendingAssistant.ReasoningContent == "" {
				// 预期之外的情况, 直接报个错
				panic(`pendingAssistant.Content == "" && pendingAssistant.ReasoningContent == ""`)
			}
			pendingAssistant.ToolCalls = append(pendingAssistant.ToolCalls, goopenai.ToolCall{
				ID:   msg.ToolCallID,
				Type: goopenai.ToolTypeFunction,
				Function: goopenai.FunctionCall{
					Name:      msg.ToolName,
					Arguments: msg.ToolArgs,
				},
			})
			i++
		case string(MsgToolResult):
			if len(pendingAssistant.ToolCalls) == 0 {
				// 预期之外的情况, 直接报个错
				panic(`pendingAssistant.ToolCalls == 0`)
			}
			pendingToolResult = append(pendingToolResult, goopenai.ChatCompletionMessage{
				Role:       goopenai.ChatMessageRoleTool,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
			})
			i++

		default:
			i++
		}
	}
	addPendingAssistant()
	addPendingToolResult()
	return result
}

// systemPromptTemplate defines the system prompt structure.
// Placeholders are replaced at runtime:
//
//	{{.ToolsList}}   – available tools with descriptions
//	{{.Guidelines}}  – tool-specific and common usage guidelines
//	{{.CWD}}         – current working directory
//	{{.OS}}          – operating system name
const systemPromptTemplate = `You are an expert coding assistant called GoCode.
You help users by reading files, executing commands, editing code, and writing new files.

Available tools:
{{ToolsList}}

Guidelines:
{{Guidelines}}
Be concise in your responses.
Show file paths clearly when working with files.

Current working directory: {{CWD}}
Current environment: {{OS}}`

// systemPrompt builds the system prompt by rendering the template with
// tool descriptions, guidelines, and environment info.
func (a *Agent) systemPrompt() string {
	prompt := systemPromptTemplate

	prompt = strings.Replace(prompt, "{{ToolsList}}", a.buildToolsPrompt(), 1)
	prompt = strings.Replace(prompt, "{{Guidelines}}", a.buildGuidelinesPrompt(), 1)
	prompt = strings.Replace(prompt, "{{CWD}}", a.cwd, 1)
	prompt = strings.Replace(prompt, "{{OS}}", osName(), 1)

	return prompt
}

// buildToolsPrompt returns the formatted list of available tools.
func (a *Agent) buildToolsPrompt() string {
	var sb strings.Builder
	for _, t := range a.oaiTools {
		snippet := t.Function.Description // fallback to full description
		for _, d := range a.toolDefs {
			if d.Name == t.Function.Name && d.PromptSnippet != "" {
				snippet = d.PromptSnippet
				break
			}
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, snippet))

	}
	return strings.TrimSpace(sb.String())
}

// buildGuidelinesPrompt returns the formatted, deduplicated list of guidelines.
func (a *Agent) buildGuidelinesPrompt() string {
	var sb strings.Builder
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
	return strings.TrimSpace(sb.String())
}

// osName returns a human-readable OS name.
func osName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	default:
		return runtime.GOOS
	}
}

// ---- message constructors ----

func sysMsg(content string) goopenai.ChatCompletionMessage {
	return goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleSystem, Content: content}
}

func userMsg(content string) goopenai.ChatCompletionMessage {
	return goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleUser, Content: content}
}

func asstMsg(content string) goopenai.ChatCompletionMessage {
	return goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleAssistant, Content: content}
}

func toolMsg(content, toolCallID string) goopenai.ChatCompletionMessage {
	return goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleTool, Content: content, ToolCallID: toolCallID}
}
