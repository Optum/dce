package shell

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
	"time"
)

// ExecInput includes options for executing a shell command
type ExecInput struct {
	Name    string   // Command to execute
	Args    []string // Arguments to pass to the command
	Dir     string   // Working directory
	Timeout float64  // Max execution time (seconds) of the command
}

// Exec executes a shell command
// and returns Stdout
func Exec(input *ExecInput) (string, error) {
	// Create a context, in order to enforce a Timeout on the command.
	// See https://medium.com/@vCabbage/go-timeout-commands-with-os-exec-commandcontext-ba0c861ed738
	// and https://siadat.github.io/post/context
	var ctx context.Context
	var cancel context.CancelFunc
	if input.Timeout == 0 {
		// If no Timeout is configured, use and empty context
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Duration(input.Timeout)*time.Second,
		)
	}

	// Cleanup context, on completion
	defer cancel()

	// Configure the shell command
	cmd := exec.CommandContext(ctx, input.Name, input.Args...)
	if input.Dir != "" {
		cmd.Dir = input.Dir
	}

	// Capture stdout/stderr in Buffer objects
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()

	// Check if the command timed out
	cmdStr := strings.Join(
		append([]string{input.Name}, input.Args...), " ",
	)
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.Wrap(ctx.Err(),
			"Command timed out: "+cmdStr+": "+stderr.String(),
		)
	}

	// Check for command errors
	if err != nil {
		return "", errors.Wrap(err,
			"Command failed: "+cmdStr+"\n"+stderr.String(),
		)
	}

	return stdout.String(), nil
}
