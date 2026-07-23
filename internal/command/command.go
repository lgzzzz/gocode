package command

import (
	"context"
	"strings"
)

// ---- Command Executor Interface ----

// Executor is the interface that all commands must implement.
type Executor interface {
	Name() string                                                        // command name, e.g. "new"
	Description() string                                                 // description, e.g. "开启一轮新的对话"
	Execute(ctx context.Context, args string, env *Env) (*Result, error) // execute the command
}

// ---- Command Environment ----

// Env holds references to the agent and TUI model, which commands can
// freely use during execution.
type Env struct {
	TUI TUIAccess // TUI model access interface
}

// TUIAccess defines the operations a command can perform on the TUI model.
// The method set is intentionally minimal and grows as needed.
type TUIAccess interface {
	Running() bool
	CancelAgent()
	NewSession()         // creates a new session in store and clears TUI history
	OpenSessionBrowser() // activates the interactive session browser
}

// ---- Command Result ----

// Result holds the outcome of a command execution.
type Result struct {
	Message string // success message to display
	Error   error  // execution error
}

// ---- Command Registry ----

// Registry stores all registered commands in order and provides
// lookup and filtering operations.
type Registry struct {
	commands []Executor     // ordered list, preserves registration order
	index    map[string]int // name -> position in commands slice
}

// NewRegistry creates a new empty command registry.
func NewRegistry() *Registry {
	return &Registry{
		index: make(map[string]int),
	}
}

// Register adds a command to the registry. If a command with the same
// name already exists, it is replaced.
func (r *Registry) Register(cmd Executor) {
	if idx, ok := r.index[cmd.Name()]; ok {
		r.commands[idx] = cmd
		return
	}
	r.index[cmd.Name()] = len(r.commands)
	r.commands = append(r.commands, cmd)
}

// Filter returns commands whose names start with the given prefix.
// Matching is case-insensitive prefix matching. Returns results in
// registration order. An empty prefix returns all commands.
func (r *Registry) Filter(prefix string) []Executor {
	if prefix == "" {
		return r.All()
	}
	lower := strings.ToLower(prefix)
	var result []Executor
	for _, cmd := range r.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name()), lower) {
			result = append(result, cmd)
		}
	}
	return result
}

// All returns all registered commands in registration order.
func (r *Registry) All() []Executor {
	result := make([]Executor, len(r.commands))
	copy(result, r.commands)
	return result
}
