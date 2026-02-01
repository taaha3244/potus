package bash

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestExecutorTool_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping bash tests on Windows")
	}

	tool := NewExecutorTool(".", 5*time.Second, nil, DefaultBlocklist)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "simple echo",
			params:  map[string]interface{}{"command": "echo hello"},
			wantErr: false,
		},
		{
			name:    "ls current directory",
			params:  map[string]interface{}{"command": "ls"},
			wantErr: false,
		},
		{
			name:    "command not found",
			params:  map[string]interface{}{"command": "nonexistentcommand12345"},
			wantErr: true,
		},
		{
			name:    "blocked command",
			params:  map[string]interface{}{"command": "rm -rf /"},
			wantErr: true,
		},
		{
			name:    "missing command parameter",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("expected error result")
			}

			if !tt.wantErr && !result.Success {
				t.Errorf("unexpected error: %v", result.Error)
			}
		})
	}
}

func TestExecutorTool_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping bash tests on Windows")
	}

	tool := NewExecutorTool(".", 100*time.Millisecond, nil, nil)

	params := map[string]interface{}{
		"command": "sleep 5",
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("expected timeout error")
	}
}

func TestExecutorTool_Allowlist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping bash tests on Windows")
	}

	allowlist := []string{"echo", "ls"}
	tool := NewExecutorTool(".", 5*time.Second, allowlist, nil)

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowed command",
			command: "echo test",
			wantErr: false,
		},
		{
			name:    "disallowed command",
			command: "cat /etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{"command": tt.command}
			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("expected error due to allowlist")
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tool := NewExecutorTool(".", 5*time.Second, nil, DefaultBlocklist)

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"safe command", "ls -la", false},
		{"blocked rm -rf /", "rm -rf /", true},
		{"blocked rm -rf / with path", "rm -rf /home/user", true}, // Contains "rm -rf /"
		{"blocked curl pipe exact", "curl | bash", true},          // Exact blocklist entry
		{"blocked wget pipe exact", "wget | sh", true},            // Exact blocklist entry
		{"blocked fork bomb", ":(){:|:&};:", true},
		{"curl with url not blocked", "curl http://example.com", false}, // URL doesn't match blocklist
		{"safe echo", "echo hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.validateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
