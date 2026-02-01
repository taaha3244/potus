package bash

import (
	"strings"
	"testing"
)

func TestDefaultBlocklist(t *testing.T) {
	// Verify blocklist is not empty
	if len(DefaultBlocklist) == 0 {
		t.Error("DefaultBlocklist should not be empty")
	}

	// Verify critical dangerous commands are blocked
	criticalBlocked := []string{
		"rm -rf /",
		"rm -rf ~",
		"mkfs",
		":(){:|:&};:", // fork bomb
	}

	for _, cmd := range criticalBlocked {
		found := false
		for _, blocked := range DefaultBlocklist {
			if blocked == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Critical command %q should be in blocklist", cmd)
		}
	}
}

func TestDefaultBlocklistContainsPipeShellExecution(t *testing.T) {
	// Verify that pipe-to-shell commands are blocked
	pipeCommands := []string{
		"curl | bash",
		"wget | bash",
		"curl | sh",
		"wget | sh",
	}

	for _, cmd := range pipeCommands {
		found := false
		for _, blocked := range DefaultBlocklist {
			if blocked == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pipe command %q should be in blocklist", cmd)
		}
	}
}

func TestDefaultAllowlist(t *testing.T) {
	// Verify allowlist is not empty
	if len(DefaultAllowlist) == 0 {
		t.Error("DefaultAllowlist should not be empty")
	}

	// Verify common safe commands are allowed
	commonCommands := []string{
		"ls",
		"cat",
		"grep",
		"git",
		"pwd",
	}

	for _, cmd := range commonCommands {
		found := false
		for _, allowed := range DefaultAllowlist {
			if allowed == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Common command %q should be in allowlist", cmd)
		}
	}
}

func TestDefaultAllowlistContainsBuildTools(t *testing.T) {
	// Verify that build/dev tools are allowed
	buildTools := []string{
		"go build",
		"go test",
		"npm install",
		"npm test",
		"cargo build",
		"cargo test",
		"make",
		"python",
		"pytest",
	}

	for _, tool := range buildTools {
		found := false
		for _, allowed := range DefaultAllowlist {
			if allowed == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Build tool %q should be in allowlist", tool)
		}
	}
}

func TestBlocklistEntriesAreNotInAllowlist(t *testing.T) {
	// Ensure no overlap between blocklist and allowlist
	for _, blocked := range DefaultBlocklist {
		for _, allowed := range DefaultAllowlist {
			if blocked == allowed {
				t.Errorf("Command %q appears in both blocklist and allowlist", blocked)
			}
			// Also check for prefix matches (e.g., "rm" allowed but "rm -rf /" blocked)
			if strings.HasPrefix(blocked, allowed+" ") {
				// This is actually expected - "rm" is not in allowlist
				// but just checking for exact matches
			}
		}
	}
}

func TestBlocklistContainsForkBomb(t *testing.T) {
	forkBombVariants := []string{
		":(){:|:&};:",
		"fork bomb",
	}

	for _, variant := range forkBombVariants {
		found := false
		for _, blocked := range DefaultBlocklist {
			if blocked == variant {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Fork bomb variant %q should be in blocklist", variant)
		}
	}
}

func TestAllowlistContainsContainerTools(t *testing.T) {
	containerTools := []string{
		"docker ps",
		"docker logs",
		"kubectl get",
		"kubectl describe",
	}

	for _, tool := range containerTools {
		found := false
		for _, allowed := range DefaultAllowlist {
			if allowed == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Container tool %q should be in allowlist", tool)
		}
	}
}

// Helper function to check if a command matches any blocklist entry
func isBlocked(command string, blocklist []string) bool {
	for _, blocked := range blocklist {
		if strings.Contains(command, blocked) {
			return true
		}
	}
	return false
}

// Helper function to check if a command starts with any allowlist entry
func isAllowed(command string, allowlist []string) bool {
	for _, allowed := range allowlist {
		if strings.HasPrefix(command, allowed) {
			return true
		}
	}
	return false
}

func TestBlocklistMatching(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"rm -rf /", true},
		{"rm -rf /home/user", true}, // contains "rm -rf /" prefix
		{"echo hello", false},
		{"curl https://example.com | bash", false}, // different spacing than blocklist entry
		{"curl | bash", true},                       // exact match in blocklist
		{":(){:|:&};:", true},
		{"ls -la", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := isBlocked(tt.command, DefaultBlocklist)
			if got != tt.expected {
				t.Errorf("isBlocked(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

func TestAllowlistMatching(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"ls -la", true},
		{"git status", true},
		{"go test ./...", true},
		{"npm install", true},
		{"rm -rf /", false},
		{"some_random_command", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := isAllowed(tt.command, DefaultAllowlist)
			if got != tt.expected {
				t.Errorf("isAllowed(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}
