# Web server example

A configuration-only application: no subcommands, just configuration resolved from environment variables, flags, a config file and defaults. It uses the empty string command — `cfg.Command("")` runs when no command word is given.

## What it demonstrates

- Empty string command as the default action
- Multiple configuration sources: environment variables, flags, file keys, defaults
- Secrets held in guarded memory
- Validation: ranges, regular expressions, required fields, durations, slices
- Help generated from the definitions

## How it works

```bash
./web-server                    # runs the default command
./web-server --port 9000        # flags of the default command
./web-server --help             # help for the default command
DATABASE_URL=... ./web-server   # values from the environment
```

Configuration is declared in two places in `main.go`:

- global `cfg.Define` calls cover every key the program reads, including env-only and file-only keys,
- the command's `Config(func(cc))` re-declares the keys that take flags, so `--port`, `--host` and `--log-level` parse against the command.

`kli.Get` resolves a key in the command's definitions first and falls back to the global ones; the global secrets are read back through `ctx.GlobalConfig.GetSecret`.

A command-level `cc.Define` replaces the global definition of the same key for that command — constraints such as `Range` or `Required` are not inherited, so the command definitions below repeat the essentials and intentionally skip the rest.

## Usage

```bash
go run main.go                                                # run with defaults

PORT=3000 LOG_LEVEL=debug go run main.go                      # environment variables
go run main.go --port 3000 --host 0.0.0.0 --log-level debug   # flags

go run main.go --help        # help for the default command
go run main.go --full-help   # extended help
```

Required values come from the environment:

```bash
DATABASE_URL="postgres://user:pass@localhost/db" \
JWT_SIGNING_KEY="your-32-character-secret-key-here" \
BASE_URL="https://example.com" \
go run main.go
```

## Configuration

| Key | Type | Sources | Notes |
|-----|------|---------|-------|
| PORT | int64 | env, flag, file, default 8080 | range 1–65535 |
| HOST | string | env, flag, file, default localhost | |
| BASE_URL | string | env, flag | required, must match `^https?://` |
| DATABASE_URL | string | env | required, secret, min length 10 |
| REDIS_URL | string | env | secret |
| LOG_LEVEL | string | env, flag, default info | one of debug/info/warn/error |
| ACCESS_TOKEN_TTL | duration | env, default 15m | between 1m and 24h |
| CORS_ORIGINS | []string | env, flag, default `http://localhost:3000` | comma-separated |
| JWT_SIGNING_KEY | string | env | required, secret, min length 32 |
| ENVIRONMENT | string | env, flag, default development | one of development/staging/production |
| MAX_CONNECTIONS | int64 | env, default 100 | range 1–1000 |
| ENABLE_METRICS | bool | env, flag, default true | |
| LOG_PERMS | string | env, default `0640` | octal `^0[0-7]{3}$` |

## Code Structure

The empty string command pattern:

```go
// Add empty string command for config-only mode
cfg.Command("").
    Func(func(ctx *kli.CommandContext) error {
        // Type-safe access to configuration
        port, _ := kli.Get[int64](ctx, "PORT")
        host, _ := kli.Get[string](ctx, "HOST")
        logLevel, _ := kli.Get[string](ctx, "LOG_LEVEL")
        
        fmt.Printf("Web Server Starting!\n")
        fmt.Printf("   Port: %d\n", port)
        fmt.Printf("   Host: %s\n", host)
        fmt.Printf("   Log Level: %s\n", logLevel)
        
        // Check for secrets
        if dbSecret := ctx.GlobalConfig.GetSecret("DATABASE_URL"); dbSecret.IsSet() {
            fmt.Printf("   Database: %s\n", maskSecret(dbSecret.String()))
        }
        
        return nil
    }).
    ShortHelp("Start the web server").
    LongHelp("Starts the web server with the specified configuration.").
    Config(func(cc *kli.CommandConfig) {
        // Add configuration to the default command
        cc.Define("PORT").Int64().Env("PORT").Flag("port").Default(int64(8080))
        cc.Define("HOST").String().Env("HOST").Flag("host").Default("localhost")
        // ... more configuration
    })
```

## Errors

Invalid or missing configuration prints a usage line and the per-key problems to stderr:

```bash
$ go run main.go
Usage: main [options]

Starts the web server with the specified configuration.
...

Configuration errors:
  --base-url string (required, pattern: ^https?://, env: BASE_URL) -> Not provided
```

## Secrets

The command function reads secrets through `ctx.GlobalConfig.GetSecret` (they are defined globally); `IsSet`, `Size`, `String` and `Bytes` expose them without printing the raw value — `maskSecret` in `main.go` is a small helper for display.
