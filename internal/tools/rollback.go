package tools

import (
	"fmt"
	"os"
	"sync"
)

// FileChange records a file modification for potential rollback.
type FileChange struct {
	Path       string
	OldContent []byte // nil means the file did not exist before (newly created)
	Existed    bool   // whether the file existed before modification
}

// ShellCommand records an executed shell command that may have side effects.
type ShellCommand struct {
	Command string
	Output  string
}

// RollbackTracker records tool side effects (file modifications and shell commands)
// during an agent interaction. It can restore modified files to their original state.
type RollbackTracker struct {
	mu            sync.Mutex
	fileChanges   []FileChange
	seenPaths     map[string]bool // tracks which paths have already been recorded (first modification wins)
	shellCommands []ShellCommand
}

// NewRollbackTracker creates a new empty tracker.
func NewRollbackTracker() *RollbackTracker {
	return &RollbackTracker{
		seenPaths: make(map[string]bool),
	}
}

// RecordFileWrite records that a file is about to be modified. oldContent should be
// the file's content before modification. If the file didn't exist, oldContent should be nil.
// Only the first modification to each path is recorded; subsequent writes to the same path are ignored.
func (t *RollbackTracker) RecordFileWrite(path string, oldContent []byte, existed bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.seenPaths[path] {
		return // already tracked the original state
	}
	t.seenPaths[path] = true
	t.fileChanges = append(t.fileChanges, FileChange{
		Path:       path,
		OldContent: oldContent,
		Existed:    existed,
	})
}

// RecordShellCommand records a shell command that was executed.
func (t *RollbackTracker) RecordShellCommand(command, output string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	t.shellCommands = append(t.shellCommands, ShellCommand{
		Command: command,
		Output:  output,
	})
}

// Rollback restores all modified files to their original state.
// Returns lists of restored file paths, shell commands that were executed (cannot auto-rollback),
// and any errors encountered during restoration.
func (t *RollbackTracker) Rollback() (restoredFiles []string, shellCommands []ShellCommand, errors []error) {
	if t == nil {
		return nil, nil, nil
	}
	restoredFiles, shellCommands, errors = t.rollBack()
	t.Clear()
	return restoredFiles, shellCommands, errors
}

func (t *RollbackTracker) rollBack() (restoredFiles []string, shellCommands []ShellCommand, errors []error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Restore files in reverse order (just in case there were multiple changes to the same file,
	// though we only track the first change per file)
	for i := len(t.fileChanges) - 1; i >= 0; i-- {
		fc := t.fileChanges[i]
		var err error
		if fc.Existed && fc.OldContent != nil {
			err = os.WriteFile(fc.Path, fc.OldContent, 0644)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to restore %s: %w", fc.Path, err))
			} else {
				restoredFiles = append(restoredFiles, fc.Path)
			}
		} else if !fc.Existed {
			err = os.Remove(fc.Path)
			if err != nil && !os.IsNotExist(err) {
				errors = append(errors, fmt.Errorf("failed to delete new file %s: %w", fc.Path, err))
			} else {
				restoredFiles = append(restoredFiles, fc.Path+" (deleted)")
			}
		}
	}

	shellCommands = make([]ShellCommand, len(t.shellCommands))
	copy(shellCommands, t.shellCommands)
	return restoredFiles, shellCommands, errors
}

// HasChanges returns true if there are any tracked changes to rollback.
func (t *RollbackTracker) HasChanges() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.fileChanges) > 0 || len(t.shellCommands) > 0
}

// Clear resets the tracker, discarding all recorded changes.
func (t *RollbackTracker) Clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fileChanges = nil
	t.seenPaths = make(map[string]bool)
	t.shellCommands = nil
}

// SetTrackerOnAll sets the rollback tracker on all tool executors.
func SetTrackerOnAll(toolMap map[string]ToolExecutor, tracker *RollbackTracker) {
	for _, tool := range toolMap {
		tool.SetTracker(tracker)
	}
}
