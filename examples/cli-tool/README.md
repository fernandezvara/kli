# CLI tool example

A command-line application demonstrating kli's command system, middleware pipeline and token authentication.

## What it demonstrates

- **Command system** - Commands, subcommands and aliases
- **Middleware pipeline** - Global, command-specific and custom middleware
- **Authentication** - Token-based auth middleware on the deploy command
- **Help** - Generated help with long/short descriptions
- **Source priority** - Environment and flag source ordering
- **Error handling** - Errors returned by `Execute` with exit codes
- **Command aliases** - Multiple names for the same command

## Usage

### Basic commands
```bash
# Help
go run main.go help
go run main.go --help
go run main.go help deploy

# System status
go run main.go status
go run main.go status --detailed --format json
```

### Deploy (requires ADMIN_TOKEN)
```bash
ADMIN_TOKEN=your-admin-token go run main.go deploy --env staging
ADMIN_TOKEN=your-admin-token go run main.go deploy --env staging --dry-run
ADMIN_TOKEN=your-admin-token go run main.go deploy --env prod --force --skip-tests --branch feature/new-ui
```

`ADMIN_TOKEN` is checked by `tokenAuthMiddleware`, attached to `deploy` only.

### Docker subcommands
```bash
go run main.go docker run --image myapp:v2 --port 8080 --detach
go run main.go docker stop --container-id abc123 --timeout 10s
go run main.go docker logs --container-id abc123 --tail 50 --follow
go run main.go docker status --filter running
```

### Admin commands
```bash
go run main.go admin-users --action list --username alice
go run main.go admin-users --action create --username newuser --role admin
go run main.go admin-shutdown --graceful --delay 60s
```

### Configuration Management
```bash
# Show current configuration
go run main.go config

# Show configuration with secrets
go run main.go config --show-secrets

# Validate configuration only
go run main.go config --validate-only
```

## Command Structure

### Global options
Shared by all commands; they are also available as environment variables (`VERBOSE`, `LOG_LEVEL`, `TIMEOUT`, `ADMIN_TOKEN`, `API_KEY`):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| --verbose | bool | false | Enable verbose logging |
| --log-level | string | info | Logging level (debug/info/warn/error) |
| --timeout | duration | 30s | Operation timeout |

### deploy
`cli-tool deploy [options]` — aliases `dep`, `release`; `ADMIN_TOKEN` required by middleware.

**Flags:**
- `--env` (required): Target environment (dev/staging/prod)
- `--dry-run`: Perform a dry run
- `--skip-tests`: Skip running tests
- `--force`: Force deployment despite checks
- `--branch`: Git branch to deploy (default: main, also env `DEPLOY_BRANCH`)

### docker
`cli-tool docker [subcommand]`

**Subcommands:**
- `run` — `--image` (default myapp:latest), `--port` (1024–65535 via custom validator), `--detach`, `--env`
- `stop` — `--container-id` (required), `--timeout`
- `logs` — `--container-id` (required), `--follow`, `--tail` (0–10000 via custom validator)
- `status` — `--filter`, `--verbose`

### admin-users
`cli-tool admin-users` — `--action` (required: list/create/delete/update), `--username`, `--role` (admin/user/readonly)

### admin-shutdown
`cli-tool admin-shutdown` — `--graceful` (default true), `--delay` (default 30s)

### status
`cli-tool status` — `--detailed`, `--format` (text/json)

### config
`cli-tool config` — `--show-secrets`, `--validate-only`

### help, custom-test
`help` prints a custom help page (aliases `--help`, `-h`); `custom-test` shows how command definitions combine with custom help.

## Middleware pipeline

The middleware pipeline in this example:

### Global Middleware (applies to all commands)
1. **RecoveryMiddleware** - Catches panics and provides clean error messages
2. **TimingMiddleware** - Measures command execution time
3. **LoggingMiddleware** - Logs all command executions (with a custom callback)
4. **MetricsMiddleware** - Collects execution metrics and logs errors (with a custom callback)

### Command-Specific Middleware
- **`tokenAuthMiddleware`** - Custom middleware defined at the bottom of
  `main.go`; applied to the `deploy` command and requires the `ADMIN_TOKEN`
  config value

## Custom Middleware

Middleware are plain functions matching the `kli.CommandMiddleware`
signature (`func(next kli.CommandFunc) kli.CommandFunc`). The
`tokenAuthMiddleware` in `main.go` shows the pattern: wrap the handler, read
configuration from the `CommandContext` (secret-aware via `IsSecret`/
`GetSecret`), and either fail with an error or call `next(ctx)`.

Attach it with `CommandBuilder.Middleware()` - command middleware runs
*after* the command's configuration (including secrets) is resolved into
`ctx.CommandConfig`. `UseMiddlewareForCommands` wraps at the global level
instead, which runs *before* values are resolved, so it cannot inspect
resolved config:

```go
func tokenAuthMiddleware(configKey string) kli.CommandMiddleware {
    return func(next kli.CommandFunc) kli.CommandFunc {
        return func(ctx *kli.CommandContext) error {
            cfg := ctx.GlobalConfig
            if ctx.CommandConfig != nil {
                cfg = ctx.CommandConfig
            }
            // ... look up the token, then:
            ctx.Set("auth_token", token)
            return next(ctx)
        }
    }
}

// Attach it to the command (runs after config resolution):
cfg.Command("deploy").
    Func(deployCommand).
    // ...Config(...) etc...
    Middleware(tokenAuthMiddleware("ADMIN_TOKEN"))
```

## Authentication

`tokenAuthMiddleware("ADMIN_TOKEN")` is attached to `deploy` with `CommandBuilder.Middleware`, so it runs after the command's configuration is resolved and can read `ctx.CommandConfig`. Without `ADMIN_TOKEN` the command fails with `missing authentication token`:

```bash
ADMIN_TOKEN=your-admin-token go run main.go deploy --env staging
```

`ADMIN_TOKEN` and `API_KEY` are defined as global secrets, so other commands can read them the same way when needed.

## Command Aliases

Commands support multiple names for convenience:

```bash
# Deploy command aliases
go run main.go deploy
go run main.go dep
go run main.go release

# Help
go run main.go help
go run main.go --help
go run main.go -h
go run main.go help deploy
```

## Error handling

`Execute` returns errors carrying an exit code; `main` maps them with `kli.ExitCode`:

```bash
# Missing required option
go run main.go deploy
# Shows: --env string (required, oneOf: ['dev', 'staging', 'prod']) -> Not provided

# Invalid option value
go run main.go deploy --env invalid
# Shows: --env string (required, oneOf: ['dev', 'staging', 'prod']) -> value 'invalid' is not one of: [dev staging prod]

# Authentication failure (deploy middleware)
go run main.go deploy --env staging
# Logs: Error in command deploy: missing authentication token (config key: ADMIN_TOKEN)
```

## Help

Help is generated from the command and flag definitions:

```bash
# Global help
go run main.go --help
# Shows: Available commands, global options

# Command help
go run main.go deploy --help
# Shows: Deploy command description, options, subcommands

# Subcommand help
go run main.go admin users --help
# Shows: Users subcommand options and usage
```
