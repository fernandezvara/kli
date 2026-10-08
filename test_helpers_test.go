package kli

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("pipe close failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("stdout capture failed: %v", err)
	}

	return buf.String()
}

// newTestCommand creates a Command for tests (replaces removed NewCommand export)
func newTestCommand(name string) *Command {
	return &Command{
		Name:        name,
		Definitions: make(map[string]*Definition),
		SubCommands: make(map[string]*Command),
		Middleware:  make([]CommandMiddleware, 0),
	}
}

func captureLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
	}()

	fn()

	return buf.String()
}

// Test command helper functions
func startCommand(ctx *CommandContext) error {
	fmt.Printf("Starting service\n")
	return nil
}

func stopCommand(ctx *CommandContext) error {
	fmt.Printf("Stopping service\n")
	return nil
}

func restartCommand(ctx *CommandContext) error {
	fmt.Printf("Restarting service\n")
	return nil
}

func startServerCommand(ctx *CommandContext) error {
	fmt.Printf("Starting server\n")
	return nil
}

func startWorkerCommand(ctx *CommandContext) error {
	fmt.Printf("Starting worker\n")
	return nil
}

func echoCommand(ctx *CommandContext) error {
	for _, arg := range ctx.Args {
		fmt.Printf("Echo: %s\n", arg)
	}
	return nil
}

func testCommand(ctx *CommandContext) error {
	fmt.Printf("Test command executed\n")
	return nil
}

func loggingMiddleware(next CommandFunc) CommandFunc {
	return func(ctx *CommandContext) error {
		fmt.Printf("Logging: Executing %s\n", ctx.Command)
		return next(ctx)
	}
}

func authMiddleware(next CommandFunc) CommandFunc {
	return func(ctx *CommandContext) error {
		fmt.Printf("Auth: Checking permissions for %s\n", ctx.Command)
		return next(ctx)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(strings.HasPrefix(s, substr) || strings.HasSuffix(s, substr) || strings.Contains(s, substr))))
}
