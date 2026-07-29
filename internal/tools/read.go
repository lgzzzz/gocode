package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReadTool reads contents of a text file and returns it as text.
type ReadTool struct{}

func (t *ReadTool) Name() string                        { return "read" }
func (t *ReadTool) SetTracker(tracker *RollbackTracker) {} // read does not modify anything

type readArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (t *ReadTool) Execute(argsJSON string) (string, error) {
	var args readArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("read: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("read: path is required")
	}
	info, err := os.Stat(args.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.Path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("read: %s is a directory, not a file", args.Path)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.Path, err)
	}
	lines := strings.Split(string(data), "\n")
	if args.Offset > 0 {
		if args.Offset > len(lines) {
			return "", fmt.Errorf("read: offset %d exceeds file length %d lines", args.Offset, len(lines))
		}
		lines = lines[args.Offset-1:]
	}
	if args.Limit > 0 && args.Limit < len(lines) {
		lines = lines[:args.Limit]
	}
	result := strings.Join(lines, "\n")
	if len(result) > 50000 {
		result = result[:50000] + "\n... [truncated]"
	}
	return result, nil
}
