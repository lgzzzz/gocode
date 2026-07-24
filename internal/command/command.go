package command

import (
	"context"
	"strings"
)


type Executor interface {
	Name() string
	Description() string                                                 // description, e.g. "开启一轮新的对话"
	Execute(ctx context.Context, args string, env *Env) (*Result, error)
}


type Env struct {
	TUI TUIAccess
}

type TUIAccess interface {
	Running() bool
	CancelAgent()
	NewSession()
	OpenSessionBrowser()
}


type Result struct {
	Message string
	Error   error
}


type Registry struct {
	commands []Executor
	index    map[string]int
}

func NewRegistry() *Registry {
	return &Registry{
		index: make(map[string]int),
	}
}

func (r *Registry) Register(cmd Executor) {
	if idx, ok := r.index[cmd.Name()]; ok {
		r.commands[idx] = cmd
		return
	}
	r.index[cmd.Name()] = len(r.commands)
	r.commands = append(r.commands, cmd)
}

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

func (r *Registry) All() []Executor {
	result := make([]Executor, len(r.commands))
	copy(result, r.commands)
	return result
}
