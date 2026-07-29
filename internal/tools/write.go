package tools

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteTool creates or overwrites a file with the given content.
// Creates parent directories as needed.
type WriteTool struct {
	tracker *RollbackTracker
}

func (t *WriteTool) Name() string                        { return "write" }
func (t *WriteTool) SetTracker(tracker *RollbackTracker) { t.tracker = tracker }

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteTool) Execute(argsJSON string) (string, error) {
	var args writeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("write: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("write: path is required")
	}
	var oldContent []byte
	existed := false
	if info, err := os.Stat(args.Path); err == nil && !info.IsDir() {
		existed = true
		oldContent, _ = os.ReadFile(args.Path)
	}
	t.tracker.RecordFileWrite(args.Path, oldContent, existed)
	if err := os.MkdirAll(dirOf(args.Path), 0755); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", args.Path, err)
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), args.Path), nil
}
