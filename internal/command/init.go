package command

import "context"

// initPrompt is the preset prompt sent to the LLM to generate AGENTS.md.
const initPrompt = `请分析当前项目并生成一个 AGENTS.md 文件放在项目根目录下。

AGENTS.md 文件的作用是：当 AI 助手在此项目中工作时，该文件会作为系统提示词的一部分被自动加载，帮助 AI 更好地理解项目。

请先用工具探索项目结构和代码（阅读关键文件、查看目录结构等），然后生成 AGENTS.md 文件。

AGENTS.md 应包含以下内容：
1. 项目概述（项目名称、用途、技术栈）
2. 项目目录结构说明
3. 构建与运行说明（如何编译、运行、测试）
4. 编码规范与约定（代码风格、命名约定等）
5. 关键模块说明
6. 注意事项或特殊约定

请根据实际项目内容来写，不要编造不存在的信息。使用中文编写 AGENTS.md。`

type InitCommand struct{}

func (c *InitCommand) Name() string        { return "init" }
func (c *InitCommand) Description() string { return "分析项目并生成 AGENTS.md 文件" }

func (c *InitCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	return &Result{
		Message:    "正在分析项目并生成 AGENTS.md ...",
		AgentInput: initPrompt,
	}, nil
}
