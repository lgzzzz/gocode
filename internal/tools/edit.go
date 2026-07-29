package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EditTool edits a file by replacing an exact text match with new text.
// oldText must be unique in the file.
type EditTool struct {
	tracker *RollbackTracker
}

func (t *EditTool) Name() string                        { return "edit" }
func (t *EditTool) SetTracker(tracker *RollbackTracker) { t.tracker = tracker }

type editArgs struct {
	Path    string `json:"path"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

func (t *EditTool) Execute(argsJSON string) (string, error) {
	var args editArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("edit: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("edit: path is required")
	}
	if args.OldText == "" {
		return "", fmt.Errorf("edit: oldText is required")
	}
	info, err := os.Stat(args.Path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("edit: %s is a directory, not a file", args.Path)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}

	// Record original file state for rollback
	t.tracker.RecordFileWrite(args.Path, data, true)

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	count := strings.Count(content, args.OldText)
	if count == 0 {
		return "", fmt.Errorf("edit: oldText not found in %s", args.Path)
	}
	if count > 1 {
		return "", fmt.Errorf("edit: oldText matches %d times in %s — must be unique", count, args.Path)
	}
	newContent := strings.Replace(content, args.OldText, args.NewText, 1)
	if err := os.WriteFile(args.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("edit %s: %w", args.Path, err)
	}
	return fmt.Sprintf("Edited %s: replaced 1 occurrence", args.Path), nil
}
