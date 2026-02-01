package tools

import (
	"context"
)

type Tool interface {
	Name() string
	Description() string
	Schema() map[string]interface{}
	Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
}

type Result struct {
	Success bool
	Output  string
	Error   error
}

func NewResult(output string) *Result {
	return &Result{
		Success: true,
		Output:  output,
	}
}

func NewErrorResult(err error) *Result {
	return &Result{
		Success: false,
		Error:   err,
		Output:  err.Error(),
	}
}
