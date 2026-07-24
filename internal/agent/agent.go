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

type Logger interface {
	Printf(format string, v ...interface{})
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...interface{}) {}


type MsgType string

const (
	MsgThinkingStream  MsgType = "thinking_stream"
	MsgAssistantStream MsgType = "assistant_stream"
	MsgThinking        MsgType = "thinking"
	MsgAssistant       MsgType = "assistant"

	MsgToolCall   MsgType = "tool_call"
	MsgToolResult MsgType = "tool_result"

	MsgError     MsgType = "error"
	MsgRetryWait MsgType = "retry_wait"

	MsgUser MsgType = "user"
)

type CallbackMsg struct {
	ID         string
	Type       MsgType
	Content    string
	ToolCallID string
	ToolName   string
	ToolArgs   string
	ToolErr    error
}

type Agent struct {
	client          *goopenai.Client
	model           string
	oaiTools        []goopenai.Tool
	toolDefs        []tools.ToolDef
	toolMap         map[string]tools.ToolExecutor
	cwd             string
	contextMessages []goopenai.ChatCompletionMessage
	logger          Logger
}

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

func (a *Agent) Run(ctx context.Context, userMessage string, cb func(CallbackMsg)) {
	if len(a.contextMessages) == 0 {
		a.contextMessages = []goopenai.ChatCompletionMessage{
			sysMsg(a.systemPrompt()),
		}
	}

	a.contextMessages = append(a.contextMessages, userMsg(userMessage))

	messages := a.contextMessages

	for {
		fullContent, fullReasoning, toolCalls, err := a.streamOne(ctx, messages, cb)
		if err != nil {
			return
		}
		if len(toolCalls) > 0 {
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

		assistantMsg := asstMsg(fullContent)
		assistantMsg.ReasoningContent = fullReasoning
		messages = append(messages, assistantMsg)
		a.contextMessages = messages
		return
	}
}

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

		if ctx.Err() != nil {
			cb(CallbackMsg{Type: MsgError, Content: fmt.Sprintf("请求已取消: %v", ctx.Err())})
			return "", "", nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		if attempt == maxRetries {
			cb(CallbackMsg{Type: MsgError, Content: fmt.Sprintf("API 调用失败（已重试 %d 次）: %v", maxRetries, err)})
			return "", "", nil, fmt.Errorf("API call failed after %d retries: %w", maxRetries, err)
		}

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
		tcMap         = make(map[int]*toolCallAccum)
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

		if delta.ReasoningContent != "" {
			fullReasoning.WriteString(delta.ReasoningContent)
			cb(CallbackMsg{Type: MsgThinkingStream, ID: fullReasoningId, Content: fullReasoning.String()})
		}

		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			cb(CallbackMsg{Type: MsgAssistantStream, ID: fullContentId, Content: fullContent.String()})
		}

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

type toolCallAccum struct {
	ID        string
	Type      string
	Name      string
	Arguments string
}

func (a *Agent) Model() string { return a.model }

func (a *Agent) SetLogger(l Logger) {
	if l == nil {
		l = nopLogger{}
	}
	a.logger = l
}

func (a *Agent) ClearContextMessage() {
	a.contextMessages = nil
}

func (a *Agent) SetContextMessage(contextMessages []goopenai.ChatCompletionMessage) {
	a.contextMessages = contextMessages
}

func (a *Agent) SystemPrompt() string {
	return a.systemPrompt()
}


type HistoryMessage struct {
	MsgType    string
	Content    string
	ToolCallID string
	ToolName   string
	ToolArgs   string
}

func ReconstructHistory(msgs []HistoryMessage, systemPrompt string) []goopenai.ChatCompletionMessage {
	var result []goopenai.ChatCompletionMessage

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

func (a *Agent) systemPrompt() string {
	prompt := systemPromptTemplate

	prompt = strings.Replace(prompt, "{{ToolsList}}", a.buildToolsPrompt(), 1)
	prompt = strings.Replace(prompt, "{{Guidelines}}", a.buildGuidelinesPrompt(), 1)
	prompt = strings.Replace(prompt, "{{CWD}}", a.cwd, 1)
	prompt = strings.Replace(prompt, "{{OS}}", osName(), 1)

	return prompt
}

func (a *Agent) buildToolsPrompt() string {
	var sb strings.Builder
	for _, t := range a.oaiTools {
		snippet := t.Function.Description
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
