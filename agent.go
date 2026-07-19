package main

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// Agent implements a ReAct-style loop using OpenAI-compatible function calling.
type Agent struct {
	client  *openai.Client
	model   string
	tools   []openai.Tool
	toolMap map[string]ToolExecutor
	cwd     string
}

func NewAgent(apiKey, model, baseURL string) *Agent {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	client := openai.NewClientWithConfig(cfg)

	tm, defs := allTools()

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
		client:  client,
		model:   model,
		tools:   oaiTools,
		toolMap: tm,
		cwd:     cwd,
	}
}

// Run executes the ReAct loop. The callback receives progress messages (tool calls, results, final response).
func (a *Agent) Run(ctx context.Context, userMessage string, cb func(string)) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: a.systemPrompt(),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		},
	}

	const maxIter = 30

	for iter := 0; iter < maxIter; iter++ {
		resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: messages,
			Tools:    a.tools,
		})
		if err != nil {
			return "", fmt.Errorf("API error: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]

		// Show reasoning text if present alongside tool calls
		if choice.Message.Content != "" {
			cb("💭 " + choice.Message.Content)
		}

		// If the model wants to call tools
		if choice.FinishReason == openai.FinishReasonToolCalls && len(choice.Message.ToolCalls) > 0 {
			// Append the assistant message (with tool calls)
			messages = append(messages, choice.Message)

			for _, tc := range choice.Message.ToolCalls {
				name := tc.Function.Name
				args := tc.Function.Arguments

				cb(fmt.Sprintf("🔧 %s(%s)", name, truncate(args, 200)))

				tool, ok := a.toolMap[name]
				var result string
				if !ok {
					result = fmt.Sprintf("Error: unknown tool '%s'", name)
				} else {
					res, err := tool.Execute(args)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					} else {
						result = res
					}
				}

				cb(fmt.Sprintf("   → %s", truncate(result, 400)))

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			continue
		}

		// Final text response
		content := choice.Message.Content
		messages = append(messages, choice.Message)
		cb("🤖 " + content)
		return content, nil
	}

	return "", fmt.Errorf("max iterations (%d) reached without final answer", maxIter)
}

func (a *Agent) systemPrompt() string {
	return fmt.Sprintf(`You are an AI coding agent that helps users with programming tasks. You operate in a ReAct (Reasoning + Acting) loop: think about what to do, use tools to act, observe results, and iterate.

You have access to four tools:
- read: Read file contents
- write: Create or overwrite files
- edit: Make precise text replacements in files
- bash: Execute shell commands

Guidelines:
1. Before making changes, use read to understand existing code
2. Use edit for precise, small changes; use write only for new files or complete rewrites
3. When edit fails because oldText is not unique, read the file around the target area and try again with more context
4. Use bash for commands like ls, grep, find, go build, go test, git, etc.
5. Work step by step: think → act → observe → decide
6. Be concise. When done, summarize what you accomplished.

Current working directory: %s`, a.cwd)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
