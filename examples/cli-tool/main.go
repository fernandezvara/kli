// CLI Tool Example - Full-Featured Command Line Application
// Demonstrates: command system, middleware pipeline, authentication, help customization
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fernandezvara/kli"
)

// Custom validator for log tail lines - ensures reasonable range
func logTailValidator(value any) error {
	if tail, ok := value.(int64); ok {
		if tail < 0 {
			return fmt.Errorf("tail lines cannot be negative, got %d", tail)
		}
		if tail > 10000 {
			return fmt.Errorf("tail lines too large (max 10000), got %d", tail)
		}
		return nil
	}
	return fmt.Errorf("tail must be an integer, got %T", value)
}

// Custom validator for port numbers - ensures non-privileged ports for security
func portSecurityValidator(value any) error {
	// this can be achived with Range(1024, 65535) but it will show generic out of bounds error
	// and this helps us provide more specific error messages
	if port, ok := value.(int64); ok {
		if port < 1024 {
			return fmt.Errorf("privileged ports (1-1023) not allowed for security, got %d", port)
		}
		if port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535, got %d", port)
		}
		return nil
	}
	return fmt.Errorf("port must be an integer, got %T", value)
}

func main() {
	cfg := kli.New()

	// Global configuration (available to all commands)
	cfg.Define("VERBOSE").
		Bool().
		Env("VERBOSE").
		Flag("verbose").
		Default(false).
		Description("Enable verbose logging")

	cfg.Define("LOG_LEVEL").
		String().
		Env("LOG_LEVEL").
		Flag("log-level").
		Default("info").
		OneOf("debug", "info", "warn", "error").
		Description("Logging level")

	cfg.Define("TIMEOUT").
		Duration().
		Env("TIMEOUT").
		Flag("timeout").
		Default(30 * time.Second).
		Description("Operation timeout")

	// Not Required(): the token is only enforced by tokenAuthMiddleware
	// on commands that need it (deploy), not for every command invocation
	cfg.Define("ADMIN_TOKEN").
		String().
		Env("ADMIN_TOKEN").
		Secret().
		Description("Admin authentication token")

	cfg.Define("API_KEY").
		String().
		Env("API_KEY").
		Secret().
		Description("API authentication key")

	// Set up global middleware
	setupGlobalMiddleware(cfg)

	// Set up commands
	setupCommands(cfg)

	// Execute with unified API
	if err := cfg.Execute(os.Args); err != nil {
		os.Exit(1)
	}
}

// setupGlobalMiddleware configures middleware that applies to all commands
func setupGlobalMiddleware(cfg *kli.Config) {
	// Recovery middleware - catches panics
	cfg.UseMiddleware(kli.RecoveryMiddleware())

	// Timing middleware - measures command execution time
	cfg.UseMiddleware(kli.TimingMiddleware())

	// Logging middleware - logs all command executions via a callback
	cfg.UseMiddleware(kli.LoggingMiddleware(func(ctx *kli.CommandContext, duration time.Duration) {
		log.Printf("Command %s completed in %v", ctx.Command, duration)
	}))

	// Metrics middleware - collects execution metrics via a callback
	// (also doubles as the error-logging hook: err is non-nil on failures)
	cfg.UseMiddleware(kli.MetricsMiddleware(func(ctx *kli.CommandContext, duration time.Duration, err error) {
		status := "success"
		if err != nil {
			status = "error"
			log.Printf("Error in command %s: %v", ctx.Command, err)
		}
		log.Printf("Metrics: command=%s duration=%v status=%s", ctx.Command, duration, status)
	}))
}

// setupCommands defines all commands with their configurations and middleware
func setupCommands(cfg *kli.Config) {
	// Deploy command
	cfg.Command("deploy").
		Func(deployCommand).
		ShortHelp("Deploy application").
		LongHelp(`Deploy the application to the specified environment.

Usage: cli-tool deploy [options]

This command handles the complete deployment process including:
- Building the application
- Running tests
- Deploying to target environment
- Health checks`).
		Aliases("dep", "release").
		Config(func(cc *kli.CommandConfig) {
			cc.Define("ENVIRONMENT").
				String().
				Flag("env").
				Required().
				OneOf("dev", "staging", "prod").
				Description("Target environment")

			cc.Define("DRY_RUN").
				Bool().
				Flag("dry-run").
				Default(false).
				Description("Perform a dry run without making changes")

			cc.Define("SKIP_TESTS").
				Bool().
				Flag("skip-tests").
				Default(false).
				Description("Skip running tests before deployment")

			cc.Define("FORCE").
				Bool().
				Flag("force").
				Default(false).
				Description("Force deployment even if checks fail")

			cc.Define("BRANCH").
				String().
				Flag("branch").
				Env("DEPLOY_BRANCH").
				Default("main").
				Description("Git branch to deploy")
		}).
		// Deploy-specific middleware: require the ADMIN_TOKEN config value.
		// Attached via .Middleware() (not UseMiddlewareForCommands) so it runs
		// AFTER command config is processed and ctx.CommandConfig is populated
		Middleware(tokenAuthMiddleware("ADMIN_TOKEN"))

	// Docker command with subcommands example
	dockerCmd := cfg.Command("docker").
		Func(dockerCommand).
		ShortHelp("Docker container operations").
		LongHelp(`Manage Docker containers for the application.

This command group provides subcommands for container operations:
- docker run: Start new containers
- docker stop: Stop running containers
- docker logs: View container logs
- docker status: Check container status`)

	// Run subcommand
	dockerCmd.SubCommand("run").
		Func(dockerRunCommand).
		ShortHelp("Run Docker container").
		LongHelp("Start a new Docker container with the specified configuration.").
		Config(func(cc *kli.CommandConfig) {
			cc.Define("IMAGE").
				String().
				Flag("image").
				Required().
				Default("myapp:latest").
				Description("Docker image to run")

			cc.Define("PORT").
				Int64().
				Flag("port").
				Default(8080).
				Range(1, 65535).
				Custom("port_security", portSecurityValidator).
				Description("Port number (non-privileged ports only: 1024-65535)")

			cc.Define("DETACH").
				Bool().
				Flag("detach").
				Default(false).
				Description("Run container in detached mode")

			cc.Define("ENVIRONMENT").
				String().
				Flag("env").
				Default("dev").
				OneOf("dev", "staging", "prod").
				Description("Environment for the container")
		})

	// Stop subcommand
	dockerCmd.SubCommand("stop").
		Func(dockerStopCommand).
		ShortHelp("Stop Docker container").
		LongHelp("Stop a running Docker container gracefully.").
		Config(func(cc *kli.CommandConfig) {
			cc.Define("CONTAINER_ID").
				String().
				Flag("container-id").
				Required().
				Description("Container ID to stop")

			cc.Define("TIMEOUT").
				Duration().
				Flag("timeout").
				Default(30 * time.Second).
				Description("Graceful shutdown timeout")
		})

	// Logs subcommand
	dockerCmd.SubCommand("logs").
		Func(dockerLogsCommand).
		ShortHelp("View container logs").
		LongHelp("Display logs from a running Docker container.").
		Config(func(cc *kli.CommandConfig) {
			cc.Define("CONTAINER_ID").
				String().
				Flag("container-id").
				Required().
				Description("Container ID to view logs from")

			cc.Define("FOLLOW").
				Bool().
				Flag("follow").
				Default(false).
				Description("Follow log output")

			cc.Define("TAIL").
				Int64().
				Flag("tail").
				Default(100).
				Custom("log_tail_range", logTailValidator).
				Description("Number of lines to show from the end (0-10000)")
		})

	// Status subcommand
	dockerCmd.SubCommand("status").
		Func(dockerStatusCommand).
		ShortHelp("Check container status").
		LongHelp("Show the status of all application containers.").
		Config(func(cc *kli.CommandConfig) {
			cc.Define("FILTER").
				String().
				Flag("filter").
				Description("Filter containers by status or name")

			cc.Define("VERBOSE").
				Bool().
				Flag("verbose").
				Default(false).
				Description("Show detailed container information")
		})

	// Admin commands with authentication
	cfg.Command("admin-users").
		Func(adminUsersCommand).
		ShortHelp("Manage users").
		LongHelp(`Administrative command for user management.

Requires ADMIN_TOKEN environment variable for authentication.`).
		Config(func(cc *kli.CommandConfig) {
			cc.Define("ACTION").
				String().
				Flag("action").
				Required().
				OneOf("list", "create", "delete", "update").
				Description("User action to perform")

			cc.Define("USERNAME").
				String().
				Flag("username").
				Description("Target username")

			cc.Define("ROLE").
				String().
				Flag("role").
				OneOf("admin", "user", "readonly").
				Default("user").
				Description("User role")
		})

	// Note: Authentication middleware would be added here in production

	cfg.Command("admin-shutdown").
		Func(adminShutdownCommand).
		ShortHelp("Shutdown the service").
		LongHelp(`Administrative command to shutdown the service.

Requires ADMIN_TOKEN environment variable for authentication.`).
		Config(func(cc *kli.CommandConfig) {
			cc.Define("GRACEFUL").
				Bool().
				Flag("graceful").
				Default(true).
				Description("Perform graceful shutdown")

			cc.Define("DELAY").
				Duration().
				Flag("delay").
				Default(30 * time.Second).
				Description("Delay before shutdown")
		})

	// Status commands with API authentication
	cfg.Command("status").
		Func(statusCommand).
		ShortHelp("Show system status").
		LongHelp(`Show comprehensive system status including API and database.

Requires API_KEY environment variable for authentication.`).
		Config(func(cc *kli.CommandConfig) {
			cc.Define("DETAILED").
				Bool().
				Flag("detailed").
				Default(false).
				Description("Show detailed status")

			cc.Define("FORMAT").
				String().
				Flag("format").
				Default("text").
				OneOf("text", "json").
				Description("Output format")
		})

	// Note: API authentication middleware would be added here in production

	cfg.Command("config").
		Func(configCommand).
		ShortHelp("Manage configuration").
		LongHelp(`Configuration management commands.

View, validate, and manage application configuration.`).
		Config(func(cc *kli.CommandConfig) {
			cc.Define("SHOW_SECRETS").
				Bool().
				Flag("show-secrets").
				Default(false).
				Description("Show secret values (use with caution)")

			cc.Define("VALIDATE_ONLY").
				Bool().
				Flag("validate-only").
				Default(false).
				Description("Only validate configuration")
		})

		// Help command with custom help
	cfg.Command("help").
		Func(helpCommand).
		ShortHelp("Show help").
		LongHelp(`Display comprehensive help for the CLI tool.

This command provides detailed help information for all available commands.
You can get help for specific commands by using 'cli-tool help <command>'.

Available commands include:
- deploy: Deploy the application to different environments
- docker: Manage Docker containers (run, stop, logs, status)
- admin-users: Manage user accounts and permissions
- admin-shutdown: Safely shutdown the service
- status: Show comprehensive system status
- config: Manage configuration settings

Use 'cli-tool <command> --help' for detailed command-specific help including
all available options, environment variables, and examples.`).
		Aliases("--help", "-h")

	// Test command with custom help
	cfg.Command("custom-test").
		Func(func(ctx *kli.CommandContext) error {
			fmt.Println("Custom test command executed!")
			return nil
		}).
		ShortHelp("Test custom help functionality").
		LongHelp(`This is a test command that demonstrates the custom help functionality.

When you run this command with --help, you'll see this custom LongHelp text
instead of the standard template-based help output.

The custom help system allows developers to provide detailed, narrative-style
help that better explains the command's purpose and usage patterns.

Features demonstrated:
- LongHelp text display
- Integration with validation errors when present
- Template-based formatting for consistency`).
		Config(func(cc *kli.CommandConfig) {
			cc.Define("TEST_VALUE").
				String().
				Flag("test-value").
				Required().
				Description("Required test value for validation testing")
		})
}

// Command implementations

func deployCommand(ctx *kli.CommandContext) error {
	environment, err := kli.Get[string](ctx, "ENVIRONMENT")
	if err != nil {
		return fmt.Errorf("failed to get ENVIRONMENT: %w", err)
	}

	dryRun, err := kli.Get[bool](ctx, "DRY_RUN")
	if err != nil {
		return fmt.Errorf("failed to get DRY_RUN: %w", err)
	}

	skipTests, err := kli.Get[bool](ctx, "SKIP_TESTS")
	if err != nil {
		return fmt.Errorf("failed to get SKIP_TESTS: %w", err)
	}

	force, err := kli.Get[bool](ctx, "FORCE")
	if err != nil {
		return fmt.Errorf("failed to get FORCE: %w", err)
	}

	branch, err := kli.Get[string](ctx, "BRANCH")
	if err != nil {
		return fmt.Errorf("failed to get BRANCH: %w", err)
	}

	fmt.Printf("=== Deploy Command ===\n")
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Dry Run: %v\n", dryRun)
	fmt.Printf("Skip Tests: %v\n", skipTests)
	fmt.Printf("Force: %v\n", force)
	fmt.Printf("Branch: %s\n", branch)

	if dryRun {
		fmt.Printf("DRY RUN: No actual deployment performed\n")
	} else {
		fmt.Printf("Deployment would be performed here...\n")
	}

	return nil
}

func adminUsersCommand(ctx *kli.CommandContext) error {
	action, err := kli.Get[string](ctx, "ACTION")
	if err != nil {
		return fmt.Errorf("failed to get ACTION: %w", err)
	}

	username, err := kli.Get[string](ctx, "USERNAME")
	if err != nil {
		return fmt.Errorf("failed to get USERNAME: %w", err)
	}

	role, err := kli.Get[string](ctx, "ROLE")
	if err != nil {
		return fmt.Errorf("failed to get ROLE: %w", err)
	}

	fmt.Printf("=== Admin Users Command ===\n")
	fmt.Printf("Action: %s\n", action)
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Role: %s\n", role)

	switch action {
	case "list":
		fmt.Printf("Users: admin, user1, user2\n")
	case "create":
		fmt.Printf("User %s created with role %s\n", username, role)
	case "delete":
		fmt.Printf("User %s deleted\n", username)
	case "update":
		fmt.Printf("User %s updated with role %s\n", username, role)
	}

	return nil
}

func adminShutdownCommand(ctx *kli.CommandContext) error {
	graceful, err := kli.Get[bool](ctx, "GRACEFUL")
	if err != nil {
		return fmt.Errorf("failed to get GRACEFUL: %w", err)
	}

	delay, err := kli.Get[time.Duration](ctx, "DELAY")
	if err != nil {
		return fmt.Errorf("failed to get DELAY: %w", err)
	}

	fmt.Printf("=== Admin Shutdown Command ===\n")
	fmt.Printf("Graceful: %v\n", graceful)
	fmt.Printf("Delay: %v\n", delay)

	if graceful {
		fmt.Printf("Initiating graceful shutdown in %v...\n", delay)
	} else {
		fmt.Printf("Initiating immediate shutdown...\n")
	}

	return nil
}

func statusCommand(ctx *kli.CommandContext) error {
	detailed, err := kli.Get[bool](ctx, "DETAILED")
	if err != nil {
		return fmt.Errorf("failed to get DETAILED: %w", err)
	}

	format, err := kli.Get[string](ctx, "FORMAT")
	if err != nil {
		return fmt.Errorf("failed to get FORMAT: %w", err)
	}

	fmt.Printf("=== Status Command ===\n")
	fmt.Printf("Detailed: %v\n", detailed)
	fmt.Printf("Format: %s\n", format)
	fmt.Printf("API Status: Healthy\n")
	fmt.Printf("Response Time: 45ms\n")
	fmt.Printf("Uptime: 7 days, 14 hours\n")

	if detailed {
		fmt.Printf("Database: Connected\n")
		fmt.Printf("Cache: Active\n")
		fmt.Printf("Queue: Normal\n")
	}

	return nil
}

func configCommand(ctx *kli.CommandContext) error {
	showSecrets, err := kli.Get[bool](ctx, "SHOW_SECRETS")
	if err != nil {
		return fmt.Errorf("failed to get SHOW_SECRETS: %w", err)
	}

	validateOnly, err := kli.Get[bool](ctx, "VALIDATE_ONLY")
	if err != nil {
		return fmt.Errorf("failed to get VALIDATE_ONLY: %w", err)
	}

	fmt.Printf("=== Config Command ===\n")
	fmt.Printf("Show Secrets: %v\n", showSecrets)
	fmt.Printf("Validate Only: %v\n", validateOnly)

	if !validateOnly {
		fmt.Printf("Current Configuration:\n")
		fmt.Printf("  LOG_LEVEL: info\n")
		fmt.Printf("  TIMEOUT: 30s\n")
		if showSecrets {
			fmt.Printf("  ADMIN_TOKEN: [REDACTED]\n")
			fmt.Printf("  API_KEY: [REDACTED]\n")
		}
	}

	return nil
}

func helpCommand(ctx *kli.CommandContext) error {
	fmt.Printf("=== Help Command ===\n")
	fmt.Printf("CLI Tool - Comprehensive Command Management\n\n")
	fmt.Printf("Available Commands:\n")
	fmt.Printf("  deploy        Deploy application\n")
	fmt.Printf("  docker        Docker container operations\n")
	fmt.Printf("    run         Run Docker container\n")
	fmt.Printf("    stop        Stop Docker container\n")
	fmt.Printf("    logs        View container logs\n")
	fmt.Printf("    status      Check container status\n")
	fmt.Printf("  admin-users   Manage users\n")
	fmt.Printf("  admin-shutdown Shutdown the service\n")
	fmt.Printf("  status        Show system status\n")
	fmt.Printf("  config        Manage configuration\n")
	fmt.Printf("  help          Show this help\n\n")
	fmt.Printf("Use 'cli-tool <command> --help' for command-specific help\n")
	fmt.Printf("Use 'cli-tool docker <subcommand> --help' for docker subcommand help\n")
	return nil
}

// Docker command implementations

func dockerCommand(ctx *kli.CommandContext) error {
	// This function handles the docker command itself
	// Subcommands will be handled by their respective functions
	fmt.Printf("=== Docker Command ===\n")
	fmt.Printf("Use 'cli-tool docker <subcommand> --help' for available subcommands\n")
	fmt.Printf("Available subcommands: run, stop, logs, status\n")
	return nil
}

func dockerRunCommand(ctx *kli.CommandContext) error {
	image, err := kli.Get[string](ctx, "IMAGE")
	if err != nil {
		return fmt.Errorf("failed to get IMAGE: %w", err)
	}

	port, err := kli.Get[int64](ctx, "PORT")
	if err != nil {
		return fmt.Errorf("failed to get PORT: %w", err)
	}

	detach, err := kli.Get[bool](ctx, "DETACH")
	if err != nil {
		return fmt.Errorf("failed to get DETACH: %w", err)
	}

	environment, err := kli.Get[string](ctx, "ENVIRONMENT")
	if err != nil {
		return fmt.Errorf("failed to get ENVIRONMENT: %w", err)
	}

	fmt.Printf("=== Docker Run Command ===\n")
	fmt.Printf("Image: %s\n", image)
	fmt.Printf("Port: %d\n", port)
	fmt.Printf("Detach: %v\n", detach)
	fmt.Printf("Environment: %s\n", environment)

	if detach {
		fmt.Printf("Starting container %s in detached mode on port %d\n", image, port)
		fmt.Printf("Container ID: abc123def456\n")
	} else {
		fmt.Printf("Starting container %s on port %d (interactive)\n", image, port)
		fmt.Printf("Container ready! Use 'docker logs abc123def456' to view logs\n")
	}

	return nil
}

func dockerStopCommand(ctx *kli.CommandContext) error {
	containerID, err := kli.Get[string](ctx, "CONTAINER_ID")
	if err != nil {
		return fmt.Errorf("failed to get CONTAINER_ID: %w", err)
	}

	timeout, err := kli.Get[time.Duration](ctx, "TIMEOUT")
	if err != nil {
		return fmt.Errorf("failed to get TIMEOUT: %w", err)
	}

	fmt.Printf("=== Docker Stop Command ===\n")
	fmt.Printf("Container ID: %s\n", containerID)
	fmt.Printf("Timeout: %v\n", timeout)

	fmt.Printf("Stopping container %s gracefully (timeout: %v)\n", containerID, timeout)
	fmt.Printf("Container stopped successfully\n")

	return nil
}

func dockerLogsCommand(ctx *kli.CommandContext) error {
	containerID, err := kli.Get[string](ctx, "CONTAINER_ID")
	if err != nil {
		return fmt.Errorf("failed to get CONTAINER_ID: %w", err)
	}

	follow, err := kli.Get[bool](ctx, "FOLLOW")
	if err != nil {
		return fmt.Errorf("failed to get FOLLOW: %w", err)
	}

	tail, err := kli.Get[int64](ctx, "TAIL")
	if err != nil {
		return fmt.Errorf("failed to get TAIL: %w", err)
	}

	fmt.Printf("=== Docker Logs Command ===\n")
	fmt.Printf("Container ID: %s\n", containerID)
	fmt.Printf("Follow: %v\n", follow)
	fmt.Printf("Tail: %d lines\n", tail)

	fmt.Printf("Container logs for %s (showing last %d lines):\n", containerID, tail)
	fmt.Printf("2024-01-15 10:30:01 [INFO] Application starting on port 8080\n")
	fmt.Printf("2024-01-15 10:30:02 [INFO] Database connection established\n")
	fmt.Printf("2024-01-15 10:30:03 [INFO] Server ready to accept connections\n")
	fmt.Printf("2024-01-15 10:30:04 [INFO] Health check passed\n")

	if follow {
		fmt.Printf("Following logs (Ctrl+C to stop)...\n")
		fmt.Printf("2024-01-15 10:30:05 [INFO] Request processed: GET /health\n")
		fmt.Printf("2024-01-15 10:30:06 [INFO] Request processed: GET /api/users\n")
	}

	return nil
}

// tokenAuthMiddleware is a custom CommandMiddleware: it requires the named
// config value to be set before the command runs. Middleware are plain
// func(next CommandFunc) CommandFunc - wrap the handler, inspect the
// CommandContext, and either return an error or call next(ctx).
func tokenAuthMiddleware(configKey string) kli.CommandMiddleware {
	return func(next kli.CommandFunc) kli.CommandFunc {
		return func(ctx *kli.CommandContext) error {
			var token string

			// Secrets live in whichever config defines the key:
			// command-scoped first, then global
			secretDefined := false
			for _, cfg := range []*kli.Config{ctx.CommandConfig, ctx.GlobalConfig} {
				if cfg != nil && cfg.IsSecret(configKey) {
					secretDefined = true
					token = cfg.GetSecret(configKey).String()
					break
				}
			}

			// Fall back to a regular (non-secret) config value;
			// Get already resolves command config vs global internally
			if !secretDefined {
				token, _ = kli.Get[string](ctx, configKey)
			}

			if token == "" {
				return fmt.Errorf("missing authentication token (config key: %s)", configKey)
			}

			// Make the token available to downstream middleware/commands
			ctx.Set("auth_token", token)

			return next(ctx)
		}
	}
}

func dockerStatusCommand(ctx *kli.CommandContext) error {
	filter, err := kli.Get[string](ctx, "FILTER")
	if err != nil {
		return fmt.Errorf("failed to get FILTER: %w", err)
	}

	verbose, err := kli.Get[bool](ctx, "VERBOSE")
	if err != nil {
		return fmt.Errorf("failed to get VERBOSE: %w", err)
	}

	fmt.Printf("=== Docker Status Command ===\n")
	fmt.Printf("Filter: %s\n", filter)
	fmt.Printf("Verbose: %v\n", verbose)

	fmt.Printf("Container Status:\n")
	fmt.Printf("CONTAINER ID   IMAGE          STATUS    PORTS\n")
	fmt.Printf("abc123def456   myapp:latest   Up 2m    0.0.0.0:8080->8080/tcp\n")
	fmt.Printf("def789ghi012   nginx:latest   Up 5m    0.0.0.0:80->80/tcp\n")
	fmt.Printf("ghi345jkl678   postgres:13    Up 10m   0.0.0.0:5432->5432/tcp\n")

	if verbose {
		fmt.Printf("\nDetailed Container Information:\n")
		fmt.Printf("Container: abc123def456\n")
		fmt.Printf("  Image: myapp:latest\n")
		fmt.Printf("  Status: Up 2 minutes\n")
		fmt.Printf("  Ports: 8080/tcp -> 0.0.0.0:8080\n")
		fmt.Printf("  Environment: dev\n")
		fmt.Printf("  Restart Policy: always\n")
		fmt.Printf("  Mounts: /app/data -> /var/lib/myapp\n")
	}

	return nil
}
