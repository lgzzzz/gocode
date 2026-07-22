package compoent

// ---- Component interface ----
//
// Component is the unified interface for all message renderers.
// Each message type (user, thinking, assistant, tool, error, system)
// implements this interface to provide its own rendering logic.

// Component represents a renderable message component.
type Component interface {
	// Type returns the kind of component (e.g. "user", "thinking", "assistant").
	Type() string
	// MsgID returns a unique identifier for this component, used for in-place updates.
	// Empty string means the component is not updatable.
	MsgID() string
	// Content returns the current content of the component.
	Content() string
	// Render returns the component rendered as a string for the given terminal width.
	Render(width int) string
	// SetContent updates the component's content in place.
	SetContent(content string)
}
