# kli

A Go library for building command-line applications. A configuration key is declared once — with its type, flag, environment variable, file key, default value and validation rules — and resolved automatically from every source. Commands and subcommands get their own flags and generated help.

- Typed access through generics: `kli.Get[T]` and `kli.MustGet[T]`
- Flags, environment variables and config files (JSON, YAML, TOML) with a configurable source priority
- Validation: ranges, enums, regular expressions, length and item bounds, durations, custom checks
- Secrets held in guarded memory and masked in dumps and help output
- Commands, subcommands, aliases and a middleware pipeline
- `Execute` returns errors that carry exit codes; it never calls `os.Exit`
- Environment lookup and standard I/O are injectable for in-process testing

## Contents

- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Commands](#commands)
- [Exit codes, environment and I/O](#exit-codes-environment-and-io)
- [Middleware](#middleware)
- [Error handling](#error-handling)
- [Help output](#help-output)
- [Examples](#examples)
- [Benchmarks](#benchmarks)
- [API reference](#api-reference)
- [Install](#install)
- [License](#license)

## Quick start

### Configuration-only application

For services, daemons and tools that need configuration without commands. `cfg.Command("")` registers a default command that runs when no command word is given:

```go
package main

import (
    "fmt"
    "os"

    "github.com/fernandezvara/kli"
)

func main() {
    cfg := kli.New()

    // An empty name makes this the default command.
    cfg.Command("").
        Func(func(ctx *kli.CommandContext) error {
            port := kli.MustGet[int64](ctx, "PORT")
            fmt.Printf("Server starting on port %d\n", port)

            if s := ctx.CommandConfig.GetSecret("DATABASE_URL"); s.IsSet() {
                fmt.Printf("Database configured (%d bytes)\n", s.Size())
            }
            return nil
        }).
        ShortHelp("Start the server").
        Config(func(cc *kli.CommandConfig) {
            cc.Define("PORT").
                Int64().
                Env("PORT").
                Flag("port").
                Default(int64(8080)).
                Range(1, 65535).
                Description("HTTP server port")

            cc.Define("DATABASE_URL").
                String().
                Env("DATABASE_URL").
                Required().
                Secret().
                Description("Database connection string")
        })

    err := cfg.Execute(os.Args)
    if err != nil && !kli.IsReported(err) {
        fmt.Fprintln(os.Stderr, err)
    }
    os.Exit(kli.ExitCode(err))
}
```

```bash
DATABASE_URL=postgres://localhost/db ./app               # start with defaults
DATABASE_URL=postgres://localhost/db ./app --port 9000   # flag overrides the default
./app --help                                             # help for the default command
```

The default command supports everything a named command does — its own definitions, validation, secrets, help and middleware — so configuration-only applications use the same machinery as multi-command tools.

### Command-based application

For tools with multiple commands. Each command declares its own configuration inside `Config`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/fernandezvara/kli"
)

func main() {
    cfg := kli.New()

    // Global configuration
    cfg.Define("VERBOSE").
        Bool().
        Flag("verbose").
        Default(false).
        Description("Enable verbose output")

    // Define commands with their own configuration
    cfg.Command("deploy").
        Func(deployCommand).
        ShortHelp("Deploy the application").
        LongHelp("Deploy the application to the specified environment.").
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
                Description("Show what would be deployed")
        })

    cfg.Command("status").
        Func(statusCommand).
        ShortHelp("Show application status").
        Aliases("st", "info")

    err := cfg.Execute(os.Args)
    if err != nil && !kli.IsReported(err) {
        fmt.Fprintln(os.Stderr, err)
    }
    os.Exit(kli.ExitCode(err))
}

func deployCommand(ctx *kli.CommandContext) error {
    env := kli.MustGet[string](ctx, "ENVIRONMENT")
    dryRun := kli.MustGet[bool](ctx, "DRY_RUN")

    if dryRun {
        fmt.Printf("Would deploy to %s (dry run)\n", env)
    } else {
        fmt.Printf("Deploying to %s\n", env)
    }
    return nil
}

func statusCommand(ctx *kli.CommandContext) error {
    fmt.Println("Application is running")
    return nil
}
```

Definitions on `cfg` itself are global: shared by every command. Their flags are parsed from the arguments **before** the command word (`myapp --verbose deploy ...`), and `kli.Get` resolves a key in the command's definitions first, then in the global ones.

## Configuration

### Types

Seven types:

```go
cc.Define("NAME").String().Default("app")
cc.Define("PORT").Int64().Default(int64(8080))
cc.Define("RATE").Float64().Default(100.0)
cc.Define("ENABLED").Bool().Default(true)
cc.Define("TIMEOUT").Duration().Default(30 * time.Second)
cc.Define("TAGS").StringSlice().Default([]string{"v1", "api"})
cc.Define("NUMBERS").Int64Slice().Default([]int64{1, 2, 3})
```

### Validation

```go
cc.Define("PORT").
    Int64().
    Range(1, 65535).
    Required()

cc.Define("EMAIL").
    String().
    Regexp(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).
    MinLength(5).
    MaxLength(100)

cc.Define("LOG_LEVEL").
    String().
    OneOf("debug", "info", "warn", "error").
    Default("info")
```

`Custom(name, check)` adds a validation function of your own.

### Configuration files

JSON, YAML and TOML are supported. `File(key)` maps a definition to a key in the file:

```go
cc.Define("PORT").
    Int64().
    Flag("port").
    File("server_port").
    Default(int64(8080))

cfg.LoadFile("config.json")                 // one file
cfg.LoadFiles("config.json", "local.json")  // later files override earlier ones
cfg.LoadFileFromEnv("CONFIG_FILE")          // file path taken from an env var
```

```json
{
  "server_port": 3000,
  "database_url": "postgres://localhost/mydb"
}
```

### Source priority

Values resolve in priority order — `Flag > Environment > File > Default` by default. A higher-priority source replaces a lower one silently: a flag overriding an environment variable is normal operation, not something to warn about.

Change the order globally or per definition:

```go
cfg.SetDefaultPriority(kli.PriorityFileEnvFlagDefault)

cc.Define("PORT").
    Int64().
    Flag("port").
    Env("PORT").
    Default(int64(8080)).
    Priority(kli.PriorityEnvFlagDefault)
```

Presets: `PriorityFlagEnvFileDefault`, `PriorityFlagEnvDefault`, `PriorityEnvFlagDefault`, `PriorityFileEnvFlagDefault`, `PriorityDefaultOnly`. A priority that references a source the definition doesn't declare is reported as a configuration error.

### Secrets

`Secret()` stores the value in guarded memory instead of the values map; it is masked in `Dump()`, and `Get` refuses to return it:

```go
cc.Define("API_KEY").
    String().
    Env("API_KEY").
    Required().
    Secret().
    Description("API authentication key")

// Inside a command — CommandConfig for command keys, GlobalConfig for global ones:
s := ctx.CommandConfig.GetSecret("API_KEY")
if s.IsSet() {
    fmt.Printf("key configured (%d bytes)\n", s.Size())
    // s.String() / s.Bytes() return the value when needed
}
```

`cfg.Destroy()` wipes all stored secrets.

## Commands

### Subcommands and aliases

```go
docker := cfg.Command("docker").ShortHelp("Docker operations")

docker.SubCommand("run").
    Func(dockerRun).
    ShortHelp("Run a container")

docker.SubCommand("stop").
    Func(dockerStop).
    ShortHelp("Stop a container")

cfg.Command("start").
    Func(start).
    ShortHelp("Start the service").
    Aliases("run", "up")
```

### Boolean flags

`Bool()` definitions are switches: `--dry-run` sets the value to true, `--dry-run=false` sets it to false. A boolean flag never consumes the next argument.

### Positional arguments

There are no positional-argument declarations: pass values as flags or environment variables. Arguments left after a command's flags are available as `ctx.Positional()`, so a command can use or reject stray arguments.

### Help

`app --help`, `app help`, `app help <command>`, `app <command> --help` and `app <group> help` all print help. A word after a group command that is not one of its subcommands (`app user zzz`) is a usage error.

## Exit codes, environment and I/O

`Execute` never exits the process. It returns an error, and `kli.ExitCode(err)` gives the exit code to end with:

| Situation | Code |
|---|---|
| Success, or help shown | `0` (`kli.ExitOK`) |
| A command returned an error | `1` (`kli.ExitFailure`) |
| Unknown command or flag, missing or invalid flag value, unknown subcommand | `2` (`kli.ExitUsage`) |
| A command returned `kli.Exit(code, err)` | `code` |

```go
func main() {
    cfg := kli.New()
    // ... definitions and commands ...
    err := cfg.Execute(os.Args)
    if err != nil && !kli.IsReported(err) {
        fmt.Fprintln(os.Stderr, err) // configuration errors were already printed
    }
    os.Exit(kli.ExitCode(err))
}

// in a command:
return kli.Exit(3, errors.New("deploy failed")) // exit code 3
```

Everything the library reads or writes can be replaced, which makes commands testable in-process:

```go
cfg.SetName("myapp").                                  // program name in usage lines
    SetEnv(func(k string) string { return env[k] }).   // instead of os.Getenv
    SetIO(stdin, stdout, stderr)                       // help -> stdout, errors -> stderr

// in a command: ctx.Getenv("X"), ctx.Stdin(), ctx.Stdout(), ctx.Stderr()
```

## Middleware

Add cross-cutting concerns to your commands:

```go
// Global middleware - applies to all commands
cfg.UseMiddleware(kli.RecoveryMiddleware())
cfg.UseMiddleware(kli.LoggingMiddleware(func(ctx *kli.CommandContext, d time.Duration) {
    log.Printf("%s completed in %v", ctx.Command, d)
}))
cfg.UseMiddleware(kli.MetricsMiddleware(collectMetrics))

// Command-specific middleware
cfg.UseMiddlewareForCommands([]string{"admin", "shutdown"}, authMiddleware)

// Custom middleware - plain functions matching CommandMiddleware
// (see a full example in examples/cli-tool: tokenAuthMiddleware)
func authMiddleware(next kli.CommandFunc) kli.CommandFunc {
    return func(ctx *kli.CommandContext) error {
        // ...validate...
        return next(ctx)
    }
}
```

Built-in middleware:

| Middleware | Description |
| ------ | ----------- |
| `RecoveryMiddleware()` | Recover from panics in commands |
| `LoggingMiddleware(func(ctx, duration))` | Log command execution with timing |
| `MetricsMiddleware(func(ctx, duration, err))` | Collect command metrics |
| `AuthMiddleware(func(ctx) error)` | Validate authentication before execution |
| `TimingMiddleware()` | Measure and store execution timing in the context |
| `ConditionalMiddleware(cond, mw)` | Apply middleware only when a condition holds |

## Error handling

Usage and configuration errors print to stderr together with the usage line; the exit code is 2:

```bash
$ DATABASE_URL=x ./app --port 99999
Usage: app [options]

Configuration errors:
  --port int64 (default: 8080, valid: 1-65535, env: PORT) -> value 99999 is greater than maximum 65535
```

## Help output

Global help:

```bash
$ myapp --help
Usage: myapp <command> [options]

Available commands:

  deploy       Deploy the application to the specified environment.
  status       (aliases: st, info) Show application status


Use 'myapp <command> --help' for command-specific help
```

Command help:

```bash
$ myapp deploy --help
Usage: myapp deploy [options]

Deploy the application to the specified environment.

Flags:
  --dry-run bool
        Show what would be deployed
  --env string (required, oneOf: ['dev', 'staging', 'prod'])
        Target environment
```

## Examples

Three programs under `examples/`:

- **web-server** — configuration-only application using the empty string command: environment variables, flags, file loading, secrets, validation
- **cli-tool** — multi-command tool with subcommands, aliases and a middleware pipeline including token authentication
- **conversions** — exercises `Get[T]` conversions for every supported type across every source

```bash
cd examples/web-server && go run main.go --help
cd examples/cli-tool && go run main.go help
cd examples/conversions && go run main.go
```

## Benchmarks

`benchmark_test.go` contains the benchmarks:

```bash
go test -run=^$ -bench=. -benchmem
```

## API reference

### Definition builder

Returned by `cfg.Define(key)` and `cc.Define(key)`:

| Method | Description |
| ------ | ----------- |
| `String()`, `Int64()`, `Float64()`, `Bool()`, `Duration()`, `StringSlice()`, `Int64Slice()` | Set value type |
| `Env(name)` | Environment variable name |
| `Flag(name)` | Command-line flag name |
| `File(key)` | Config-file key name |
| `Default(value)` | Default value |
| `Delimiter(d)` | Delimiter for parsing slice flag values |
| `Required()` | Mark as required |
| `Secret()` | Guarded memory; masked in output |
| `Description(text)` | Text for help |
| `Priority(p)` | Per-definition source priority |
| `Min(n)`, `Max(n)`, `Range(min, max)` | Numeric bounds |
| `OneOf(values...)` | Enum validation (string values) |
| `Regexp(pattern)` | Regex validation |
| `MinLength(n)`, `MaxLength(n)` | String length bounds |
| `MinItems(n)`, `MaxItems(n)` | Slice item-count bounds |
| `MinDuration(d)`, `MaxDuration(d)` | Duration bounds |
| `Custom(name, check)` | Custom validation function |

### Config

| Method | Description |
| ------ | ----------- |
| `New()` | Create a Config |
| `Define(key)` | Define a global configuration key |
| `Command(name)` | Define a command (`""` is the default command) |
| `Execute(args)` | Route and run; returns an error carrying an exit code |
| `SetName(name)` | Program name shown in usage lines |
| `SetEnv(fn)` | Replace the environment lookup (default `os.Getenv`) |
| `SetIO(in, out, err)` | Replace stdin, stdout, stderr |
| `SetDefaultPriority(p)` | Global default source priority |
| `LoadFile(path)`, `LoadFiles(paths...)`, `LoadFileFromEnv(env)` | Load configuration files |
| `GetSecret(key)`, `HasSecret(key)`, `IsSecret(key)` | Secret access |
| `Has(key)`, `Keys()` | Definition lookup |
| `Dump()` | All values (secrets masked) |
| `Destroy()` | Wipe all secrets |
| `GenerateHelp()` | Generated help text |
| `ShowGlobalHelp()`, `ShowCommandHelp(name)` | Print help |
| `UseMiddleware(fn)`, `UseMiddlewareForCommands(names, fn)`, `UseMiddlewareForSubcommands(cmd, names, fn)` | Register middleware |

### Command builder

| Method | Description |
| ------ | ----------- |
| `Func(fn)` | Command function |
| `ShortHelp(text)`, `LongHelp(text)` | Help text |
| `Aliases(names...)` | Command aliases |
| `Config(fn)` | Command-specific definitions via `cc.Define` |
| `SubCommand(name)` | Add a subcommand |
| `Middleware(fn)` | Command-specific middleware |

### CommandContext

| Member | Description |
| ------ | ----------- |
| `Args`, `Command`, `SubCommand` | Invocation details |
| `GlobalConfig`, `CommandConfig` | Resolved configuration |
| `Positional()` | Arguments left after flag parsing |
| `Getenv(name)` | Environment lookup through `SetEnv` |
| `Stdin()`, `Stdout()`, `Stderr()` | Streams from `SetIO` |
| `Set(k, v)`, `GetData(k)` | Middleware data sharing |

### Package functions and exit codes

| Function | Description |
| ------ | ----------- |
| `Get[T](ctx, key)` | Typed lookup (returns `T`, `error`) |
| `MustGet[T](ctx, key)` | Typed lookup or panic |
| `Exit(code, err)` | Error that makes `ExitCode` report `code` |
| `ExitCode(err)` | Exit code for the error `Execute` returned |
| `IsReported(err)` | Whether the library already printed the error |
| `ExitOK`, `ExitFailure`, `ExitUsage` | Exit codes 0, 1, 2 |

## Install

```bash
go get github.com/fernandezvara/kli
```

## License

MIT
