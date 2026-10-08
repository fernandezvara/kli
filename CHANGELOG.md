# Changelog

## v0.4.0

### Changed

- **Breaking**: the library was renamed from `cli` to `kli`. The module path is now `github.com/fernandezvara/kli` and the package name is `kli`. Update your imports and replace `cli.` qualifiers with `kli.` (or keep `cli.` by aliasing the import: `cli "github.com/fernandezvara/kli"`).

## v0.3.0

Same release as v0.2.1; the tag was created without changes.

## v0.2.1

### Changed

- Command and subcommand lists show each command's `ShortHelp` (else the first line of its `LongHelp`); a command's own help shows its `ShortHelp` followed by its `LongHelp`.
- A command that declares no flags now rejects unknown flags (usage error, code 2) and exposes its leftover arguments through `Positional()`; global flags are still accepted.
- `app --help` and `app help` list the commands when the app has a default command next to other commands (a config-only app still shows the default command's help).
- Unknown-command errors omit "Did you mean" when nothing is close, and never suggest the default (unnamed) command.
- Help no longer shows `(default: )` for optional string flags with an empty default.

## v0.2.0

Made for programs that must control their exit code and be tested in-process.

### Added

- `kli.Exit(code, err)`, `kli.ExitCode(err)`, `kli.IsReported(err)` and `ExitOK`/`ExitFailure`/`ExitUsage`.
- `Config.SetEnv`, `Config.SetIO` and `Config.SetName`; `CommandContext.Getenv`, `Stdin`, `Stdout`, `Stderr`.
- `CommandContext.Positional()`: the arguments left after a command's flags.
- `help` as a word (`app help`, `app help <command>`, `app <command> help`).
- Real boolean flags: `--flag` and `--flag=false`.

### Changed

- **`Execute` never calls `os.Exit`.** Usage errors (unknown command or flag, missing or invalid flag value) return exit code 2 instead of exiting with 1; command errors are 1. `CommandResult.Handle` was removed.
- **Boolean flags no longer take the next argument**: `--verbose false` is now `--verbose` plus a leftover argument; write `--verbose=false`.
- Usage lines name the program and the full command: `Usage: app user list [options]`.
- A word after a group command that is not one of its subcommands is a usage error instead of showing help.
- Help is written to the configured stdout (default `os.Stdout`), errors to the configured stderr.

### Fixed

- README: subcommands are declared with `SubCommand`, not `Config(... cc.Command ...)`.
