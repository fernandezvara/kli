package kli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type harness struct {
	cfg            *Config
	stdout, stderr *bytes.Buffer
}

func newHarness(env map[string]string) *harness {
	h := &harness{cfg: New(), stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	h.cfg.SetName("app").SetIO(strings.NewReader("input\n"), h.stdout, h.stderr).
		SetEnv(func(k string) string { return env[k] })
	return h
}

func (h *harness) run(args ...string) error {
	return h.cfg.Execute(append([]string{"app"}, args...))
}

func TestExitCodes(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("ok").Func(func(*CommandContext) error { return nil })
	h.cfg.Command("fail").Func(func(*CommandContext) error { return errors.New("boom") })
	h.cfg.Command("three").Func(func(*CommandContext) error { return Exit(3, errors.New("custom")) })
	h.cfg.Command("silent").Func(func(*CommandContext) error { return Exit(4, nil) })
	h.cfg.Command("wrapped").Func(func(*CommandContext) error {
		return errors.Join(errors.New("context"), Exit(5, errors.New("inner")))
	})
	h.cfg.Command("need").Func(func(*CommandContext) error { return nil }).Config(func(cc *CommandConfig) {
		cc.Define("REALM").String().Flag("realm").Required().Description("the realm")
	})

	tests := []struct {
		name     string
		args     []string
		code     int
		reported bool
		stderr   string
	}{
		{"success", []string{"ok"}, 0, false, ""},
		{"runtime error", []string{"fail"}, 1, false, ""},
		{"chosen code", []string{"three"}, 3, false, ""},
		{"silent chosen code", []string{"silent"}, 4, false, ""},
		{"code inside a chain", []string{"wrapped"}, 5, false, ""},
		{"unknown command", []string{"nope"}, 2, false, ""},
		{"missing required flag", []string{"need"}, 2, true, "--realm"},
		{"unknown flag", []string{"need", "--realm", "r", "--bogus"}, 2, true, "bogus"},
		{"flag without value", []string{"need", "--realm"}, 2, true, "realm"},
		{"help is success", []string{"--help"}, 0, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.stdout.Reset()
			h.stderr.Reset()
			err := h.run(tt.args...)
			if got := ExitCode(err); got != tt.code {
				t.Fatalf("exit code = %d, want %d (err %v)", got, tt.code, err)
			}
			if IsReported(err) != tt.reported {
				t.Errorf("reported = %v, want %v", IsReported(err), tt.reported)
			}
			if !strings.Contains(h.stderr.String(), tt.stderr) {
				t.Errorf("stderr = %q, want it to contain %q", h.stderr.String(), tt.stderr)
			}
		})
	}
	if ExitCode(nil) != 0 {
		t.Error("ExitCode(nil) != 0")
	}
}

func TestConfigOnlyAppUsageError(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Define("PORT").Int64().Flag("port").Required().Description("port")
	err := h.run()
	if ExitCode(err) != 2 || !IsReported(err) || !strings.Contains(h.stderr.String(), "--port") {
		t.Fatalf("code %d reported %v stderr %q", ExitCode(err), IsReported(err), h.stderr.String())
	}
}

func TestEnvIsInjected(t *testing.T) {
	t.Setenv("REALM_FROM_OS", "os-realm")
	h := newHarness(map[string]string{"REALM_FROM_ENV": "env-realm"})
	var got, fromOS string
	h.cfg.Command("show").Func(func(ctx *CommandContext) error {
		got = MustGet[string](ctx, "REALM")
		fromOS = MustGet[string](ctx, "OTHER")
		if ctx.Getenv("REALM_FROM_ENV") != "env-realm" || ctx.Getenv("REALM_FROM_OS") != "" {
			t.Errorf("ctx.Getenv does not use the injected environment")
		}
		return nil
	}).Config(func(cc *CommandConfig) {
		cc.Define("REALM").String().Env("REALM_FROM_ENV").Required().Description("realm")
		cc.Define("OTHER").String().Env("REALM_FROM_OS").Default("none").Description("other")
	})
	if err := h.run("show"); err != nil {
		t.Fatal(err)
	}
	if got != "env-realm" {
		t.Errorf("REALM = %q, want the injected value", got)
	}
	if fromOS != "none" {
		t.Errorf("OTHER = %q: the process environment must not be read once SetEnv is used", fromOS)
	}
}

func TestStreamsAreInjected(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("echo").Func(func(ctx *CommandContext) error {
		var b [16]byte
		n, _ := ctx.Stdin().Read(b[:])
		_, _ = ctx.Stdout().Write(b[:n])
		_, _ = ctx.Stderr().Write([]byte("err\n"))
		return nil
	}).ShortHelp("echo stdin")
	if err := h.run("echo"); err != nil {
		t.Fatal(err)
	}
	if h.stdout.String() != "input\n" || h.stderr.String() != "err\n" {
		t.Errorf("stdout %q stderr %q", h.stdout, h.stderr)
	}
	h.stdout.Reset()
	if err := h.run("--help"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stdout.String(), "app") || !strings.Contains(h.stdout.String(), "echo stdin") {
		t.Errorf("help went elsewhere or lacks the program name: %q", h.stdout)
	}
}

func TestBoolFlags(t *testing.T) {
	h := newHarness(nil)
	var yes, dry, other bool
	var env string
	h.cfg.Command("run").Func(func(ctx *CommandContext) error {
		yes = MustGet[bool](ctx, "YES")
		dry = MustGet[bool](ctx, "DRY")
		other = MustGet[bool](ctx, "OTHER")
		env = MustGet[string](ctx, "ENV")
		return nil
	}).Config(func(cc *CommandConfig) {
		cc.Define("YES").Bool().Flag("yes").Default(false).Description("yes")
		cc.Define("DRY").Bool().Flag("dry-run").Default(true).Description("dry")
		cc.Define("OTHER").Bool().Flag("other").Default(false).Description("other")
		cc.Define("ENV").String().Flag("env").Default("").Description("env")
	})
	tests := []struct {
		args            []string
		yes, dry, other bool
		env             string
	}{
		{[]string{"run"}, false, true, false, ""},
		{[]string{"run", "--yes"}, true, true, false, ""},
		{[]string{"run", "--yes", "--env", "prod"}, true, true, false, "prod"},
		{[]string{"run", "--env", "prod", "--yes"}, true, true, false, "prod"},
		{[]string{"run", "--dry-run=false"}, false, false, false, ""},
		{[]string{"run", "--yes=true", "--other"}, true, true, true, ""},
	}
	for _, tt := range tests {
		yes, dry, other, env = false, false, false, ""
		if err := h.run(tt.args...); err != nil {
			t.Fatalf("%v: %v (stderr %q)", tt.args, err, h.stderr)
		}
		if yes != tt.yes || dry != tt.dry || other != tt.other || env != tt.env {
			t.Errorf("%v: yes=%v dry=%v other=%v env=%q", tt.args, yes, dry, other, env)
		}
	}
	h.stderr.Reset()
	err := h.run("run", "--yes=maybe")
	if ExitCode(err) != 2 || !strings.Contains(h.stderr.String(), "yes") {
		t.Errorf("--yes=maybe: code %d stderr %q", ExitCode(err), h.stderr)
	}
}

func TestHelpWord(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("user").ShortHelp("manage users").
		SubCommand("list").Func(func(*CommandContext) error { return nil }).ShortHelp("list the users")
	h.cfg.Command("plain").Func(func(*CommandContext) error { return nil }).ShortHelp("a plain command")

	for _, args := range [][]string{{"help"}, {"--help"}, {"help", "user"}, {"user", "help"}, {"user", "--help"}, {"plain", "help"}} {
		h.stdout.Reset()
		if err := h.run(args...); err != nil || ExitCode(err) != 0 {
			t.Fatalf("%v: %v", args, err)
		}
		if h.stdout.Len() == 0 {
			t.Errorf("%v: no help printed", args)
		}
	}
	h.stdout.Reset()
	_ = h.run("user", "help")
	if !strings.Contains(h.stdout.String(), "list the users") {
		t.Errorf("`user help` should list the subcommands: %q", h.stdout)
	}
}

func TestGroupAndStrayArguments(t *testing.T) {
	h := newHarness(nil)
	var positional []string
	u := h.cfg.Command("user").ShortHelp("manage users")
	u.SubCommand("list").ShortHelp("list").Func(func(ctx *CommandContext) error {
		positional = ctx.Positional()
		return nil
	}).Config(func(cc *CommandConfig) {
		cc.Define("REALM").String().Flag("realm").Default("r").Description("realm")
	})

	// A word that names no subcommand is a usage error, not help.
	err := h.run("user", "zzz")
	if ExitCode(err) != 2 || IsReported(err) || err == nil || !strings.Contains(err.Error(), `"zzz"`) {
		t.Errorf("user zzz: code %d reported %v err %v", ExitCode(err), IsReported(err), err)
	}
	// A group alone prints its help.
	h.stdout.Reset()
	if err := h.run("user"); err != nil || !strings.Contains(h.stdout.String(), "list") {
		t.Errorf("user: err %v out %q", err, h.stdout)
	}
	// Leftover non-flag arguments are visible to the command.
	if err := h.run("user", "list", "--realm", "x", "extra", "more"); err != nil {
		t.Fatal(err)
	}
	if len(positional) != 2 || positional[0] != "extra" {
		t.Errorf("Positional() = %v", positional)
	}
	if err := h.run("user", "list", "--realm=x"); err != nil || len(positional) != 0 {
		t.Errorf("no stray arguments: err %v positional %v", err, positional)
	}
}

func TestUsageLineNamesTheProgramAndCommand(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("user").ShortHelp("manage users").
		SubCommand("list").ShortHelp("list").Func(func(*CommandContext) error { return nil })
	if err := h.run("user", "list", "--help"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stdout.String(), "Usage: app user list [options]") {
		t.Errorf("usage line: %q", h.stdout)
	}
}

func TestGlobalDefinitionsUseInjectedEnv(t *testing.T) {
	t.Setenv("APP_URL", "from-os")
	h := newHarness(map[string]string{"APP_URL": "from-injected"})
	h.cfg.Define("URL").String().Env("APP_URL").Flag("url").Default("").Description("server")
	var got string
	h.cfg.Command("show").Func(func(ctx *CommandContext) error {
		got = MustGet[string](ctx, "URL")
		return nil
	})
	if err := h.run("show"); err != nil {
		t.Fatal(err)
	}
	if got != "from-injected" {
		t.Errorf("URL = %q", got)
	}
	if err := h.run("show", "--url", "from-flag"); err != nil || got != "from-flag" {
		t.Errorf("flag should win: %q %v", got, err)
	}
}

func TestHelpSummariesAndDefaults(t *testing.T) {
	h := newHarness(nil)
	g := h.cfg.Command("group").ShortHelp("group summary").LongHelp("group details\nsecond line")
	g.SubCommand("leaf").ShortHelp("leaf summary").LongHelp("leaf details\nmore details").
		Func(func(*CommandContext) error { return nil }).
		Config(func(cc *CommandConfig) {
			cc.Define("NAME").String().Flag("name").Default("").Description("optional name")
			cc.Define("MODE").String().Flag("mode").Default("fast").Description("mode")
		})

	for args, want := range map[string][]string{
		"--help":       {"group summary"},
		"group --help": {"leaf summary"},
	} {
		h.stdout.Reset()
		if err := h.run(strings.Fields(args)...); err != nil {
			t.Fatal(err)
		}
		for _, w := range want {
			if !strings.Contains(h.stdout.String(), w) {
				t.Errorf("%s: missing %q in %q", args, w, h.stdout)
			}
		}
		if strings.Contains(h.stdout.String(), "more details") || (args == "--help" && strings.Contains(h.stdout.String(), "second line")) {
			t.Errorf("%s: lists should show summaries only: %q", args, h.stdout)
		}
	}

	h.stdout.Reset()
	if err := h.run("group", "leaf", "--help"); err != nil {
		t.Fatal(err)
	}
	out := h.stdout.String()
	for _, w := range []string{"leaf summary", "leaf details", "more details", "--mode string (default: fast)"} {
		if !strings.Contains(out, w) {
			t.Errorf("leaf help lacks %q: %q", w, out)
		}
	}
	if strings.Contains(out, "default: )") || strings.Contains(out, "default: ,") {
		t.Errorf("an empty default should not be shown: %q", out)
	}
}

func TestCommandWithoutFlagsIsStrict(t *testing.T) {
	h := newHarness(nil)
	var positional []string
	h.cfg.Command("plain").Func(func(ctx *CommandContext) error {
		positional = ctx.Positional()
		return nil
	})
	err := h.run("plain", "--bogus")
	if ExitCode(err) != 2 || !IsReported(err) || !strings.Contains(h.stderr.String(), "bogus") {
		t.Errorf("unknown flag: code %d reported %v stderr %q", ExitCode(err), IsReported(err), h.stderr)
	}
	if err := h.run("plain", "extra", "more"); err != nil || len(positional) != 2 || positional[0] != "extra" {
		t.Errorf("positional = %v, err %v", positional, err)
	}
	if err := h.run("plain"); err != nil || len(positional) != 0 {
		t.Errorf("positional = %v, err %v", positional, err)
	}
}

func TestUnknownCommandMessage(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("").Func(func(*CommandContext) error { return nil })
	h.cfg.Command("start").Func(func(*CommandContext) error { return nil })
	err := h.run("strat")
	if ExitCode(err) != 2 || err == nil || !strings.Contains(err.Error(), `unknown command: "strat"`) || !strings.Contains(err.Error(), "Did you mean: start?") {
		t.Errorf("close name: %v", err)
	}
	err = h.run("completely-different")
	if err == nil || strings.Contains(err.Error(), "Did you mean") {
		t.Errorf("no suggestion should mean no \"Did you mean\": %v", err)
	}
}

func TestGlobalHelpWithDefaultCommand(t *testing.T) {
	h := newHarness(nil)
	h.cfg.Command("").ShortHelp("run the server").Func(func(*CommandContext) error { return nil })
	h.cfg.Command("check").ShortHelp("check the config").Func(func(*CommandContext) error { return nil })
	for _, a := range []string{"--help", "help"} {
		h.stdout.Reset()
		if err := h.run(a); err != nil || !strings.Contains(h.stdout.String(), "check the config") {
			t.Errorf("%s: err %v out %q", a, err, h.stdout)
		}
	}
}
