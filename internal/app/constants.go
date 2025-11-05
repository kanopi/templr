package app

// Exit codes for CI-friendly behavior.
const (
	ExitOK            = 0
	ExitGeneral       = 1
	ExitTemplateError = 2
	ExitDataError     = 3
	ExitStrictError   = 4
	ExitGuardSkipped  = 5
	ExitLintWarn      = 6 // lint found warnings (with --fail-on-warn)
	ExitLintError     = 7 // lint found errors
)

// Version is set at build time via -ldflags
var Version string

// GetVersion returns a human-friendly version string.
func GetVersion() string {
	if Version != "" {
		return Version
	}
	return "dev"
}

// Contains checks if a string contains a substring
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
