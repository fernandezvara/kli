// cli/flag_alias_test.go
package kli

import (
	"slices"
	"strings"
	"testing"
)

func TestDefinitionBuilderMultipleFlags(t *testing.T) {
	cfg := New()
	cfg.Define("FILE").Flag("f").Flag("file").Flag("").Flag("f").Flag("input")

	got := cfg.definitions["FILE"].flags
	if want := []string{"f", "file", "input"}; !slices.Equal(got, want) {
		t.Errorf("flags = %v, want %v", got, want)
	}

	cfg.Define("NONE").Flag("")
	if got := cfg.definitions["NONE"].flags; len(got) != 0 {
		t.Errorf("an empty flag name should be ignored, got %q", got)
	}
}

func TestFlagParser_MultipleFlags(t *testing.T) {
	defs := map[string]*Definition{
		"file":    {key: "file", valueType: TypeString, flags: []string{"f", "file", "input"}},
		"verbose": {key: "verbose", valueType: TypeBool, flags: []string{"v", "verbose"}},
	}

	tests := []struct {
		name    string
		args    []string
		file    string
		verbose string
	}{
		{name: "none"},
		{name: "first names", args: []string{"-f", "a", "-v"}, file: "a", verbose: "true"},
		{name: "second names", args: []string{"--file", "b", "--verbose"}, file: "b", verbose: "true"},
		{name: "third name with equals", args: []string{"--input=c"}, file: "c"},
		{name: "bool alias set to false", args: []string{"--verbose=false"}, verbose: "false"},
		{name: "last one wins", args: []string{"-f", "a", "--file", "b", "-v", "--verbose=false"}, file: "b", verbose: "false"},
		{name: "last one wins reversed", args: []string{"--file", "b", "-f", "a"}, file: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := newFlagParser().ParseCommand(tt.args, defs)
			if err != nil || len(parsed.Errors) > 0 {
				t.Fatalf("ParseCommand() errors: %v %v", err, parsed.Errors)
			}
			if got := *parsed.Values["file"]; got != tt.file {
				t.Errorf("file = %q, want %q", got, tt.file)
			}
			if got := *parsed.Values["verbose"]; got != tt.verbose {
				t.Errorf("verbose = %q, want %q", got, tt.verbose)
			}
		})
	}
}

func TestFlagParser_ErrorForAnyFlagName(t *testing.T) {
	defs := map[string]*Definition{
		"FILE": {key: "FILE", valueType: TypeString, flags: []string{"f", "file"}},
	}
	parser := newFlagParser()

	for _, arg := range []string{"-f", "--file"} {
		parsed, _ := parser.ParseCommand([]string{arg}, defs)
		errs := parser.ConvertFlagErrorsToConfigErrors(parsed.Errors, defs)
		if len(errs) != 1 {
			t.Fatalf("%s: expected 1 error, got %v", arg, errs)
		}
		if errs[0].Key != "FILE" || errs[0].Display != "-f --file string" {
			t.Errorf("%s: got key %q display %q", arg, errs[0].Key, errs[0].Display)
		}
	}
}

func TestFormatFlagNames(t *testing.T) {
	tests := []struct {
		flags []string
		want  string
	}{
		{[]string{"file"}, "--file"},
		{[]string{"f"}, "-f"},
		{[]string{"f", "file"}, "-f --file"},
		{[]string{"file", "f"}, "-f --file"},
		{[]string{"file", "f", "input", "i"}, "-f -i --file --input"},
	}

	for _, tt := range tests {
		if got := formatFlagNames(tt.flags); got != tt.want {
			t.Errorf("formatFlagNames(%v) = %q, want %q", tt.flags, got, tt.want)
		}
	}
}

// newAliasHarness builds an app whose "read" command has a flag with two names.
func newAliasHarness(got *string) *harness {
	h := newHarness(nil)
	h.cfg.Command("read").
		Func(func(ctx *CommandContext) error {
			v, err := Get[string](ctx, "FILE")
			*got = v
			return err
		}).
		Config(func(cc *CommandConfig) {
			cc.Define("FILE").String().Flag("f").Flag("file").Env("APP_FILE").
				Required().Description("input file")
		})
	return h
}

func TestMultipleFlagsExecute(t *testing.T) {
	for _, args := range [][]string{{"read", "-f", "a.txt"}, {"read", "--file", "a.txt"}} {
		var got string
		h := newAliasHarness(&got)
		if err := h.run(args...); err != nil {
			t.Fatalf("%v: %v (stderr: %s)", args, err, h.stderr.String())
		}
		if got != "a.txt" {
			t.Errorf("%v: FILE = %q, want a.txt", args, got)
		}
	}
}

func TestMultipleFlagsListedOnceInHelp(t *testing.T) {
	var got string
	h := newAliasHarness(&got)
	if err := h.run("read", "--full-help"); err != nil {
		t.Fatalf("help: %v", err)
	}
	help := h.stdout.String() + h.stderr.String()

	if !strings.Contains(help, "  -f --file string (required, env: APP_FILE)\n") {
		t.Errorf("help should list both names on one line:\n%s", help)
	}
	for _, name := range []string{"-f ", "--file"} {
		if n := strings.Count(help, name); n != 1 {
			t.Errorf("%q should appear once, got %d:\n%s", name, n, help)
		}
	}
}

func TestFlagParser_ErrorForHyphenatedName(t *testing.T) {
	defs := map[string]*Definition{
		"DRY": {key: "DRY", valueType: TypeString, flags: []string{"n", "dry-run"}},
	}
	parser := newFlagParser()

	tests := []struct {
		arg, key, display string
	}{
		{"--dry-run", "DRY", "-n --dry-run string"},
		{"--no-such-flag", "unknown_flag", "--no-such-flag string"},
	}

	for _, tt := range tests {
		parsed, _ := parser.ParseCommand([]string{tt.arg}, defs)
		errs := parser.ConvertFlagErrorsToConfigErrors(parsed.Errors, defs)
		if len(errs) != 1 {
			t.Fatalf("%s: expected 1 error, got %v", tt.arg, errs)
		}
		if errs[0].Key != tt.key || errs[0].Display != tt.display {
			t.Errorf("%s: got key %q display %q", tt.arg, errs[0].Key, errs[0].Display)
		}
	}
}

// A flag name used twice fails every run, whatever the arguments, and no
// command runs.
func TestDuplicateFlagNameIsReported(t *testing.T) {
	ran := false
	run := func(*CommandContext) error { ran = true; return nil }
	define := func(cfg *Config) {
		cfg.Define("ALPHA").String().Flag("a").Flag("alpha")
		cfg.Define("BETA").Bool().Flag("beta").Flag("a")
	}

	tests := []struct {
		name  string
		args  []string
		setup func(h *harness)
	}{
		{"command flags", []string{"go"}, func(h *harness) {
			h.cfg.Command("go").Func(run).Config(func(cc *CommandConfig) { define(cc.Config) })
		}},
		{"global and command flags", []string{"go"}, func(h *harness) {
			h.cfg.Define("ALPHA").String().Flag("a")
			h.cfg.Command("go").Func(run).Config(func(cc *CommandConfig) {
				cc.Define("BETA").String().Flag("a")
			})
		}},
		{"global flags, command without its own", []string{"go"}, func(h *harness) {
			define(h.cfg)
			h.cfg.Command("go").Func(run)
		}},
		{"app without commands", nil, func(h *harness) { define(h.cfg) }},
		{"help", []string{"--help"}, func(h *harness) {
			h.cfg.Command("go").Func(run).Config(func(cc *CommandConfig) { define(cc.Config) })
		}},
		{"another command", []string{"other"}, func(h *harness) {
			h.cfg.Command("other").Func(run)
			h.cfg.Command("go").SubCommand("sub").Func(run).
				Config(func(cc *CommandConfig) { define(cc.Config) })
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran = false
			h := newHarness(nil)
			tt.setup(h)

			err := h.run(tt.args...)
			if ExitCode(err) != ExitFailure {
				t.Errorf("exit code = %d, want %d (err %v)", ExitCode(err), ExitFailure, err)
			}
			if ran {
				t.Error("command should not run")
			}
			want := `flag -a duplicated on "ALPHA" and "BETA"`
			if n := strings.Count(h.stderr.String(), want); n != 1 {
				t.Errorf("stderr should contain %q once, got %d:\n%s", want, n, h.stderr.String())
			}
		})
	}
}

func TestHelpFlagSynonyms(t *testing.T) {
	for _, arg := range []string{"-h", "--h", "-help", "--help", "-full-help", "--full-help"} {
		for _, args := range [][]string{{arg}, {"read", arg}} {
			var got string
			h := newAliasHarness(&got)

			if err := h.run(args...); err != nil {
				t.Errorf("%v: %v (stderr: %s)", args, err, h.stderr.String())
			}
			if !strings.Contains(h.stdout.String(), "read") {
				t.Errorf("%v: expected help, got %q", args, h.stdout.String())
			}
		}
	}

	for _, arg := range []string{"-full-help", "--full-help"} {
		if !isFullHelpFlag(arg) {
			t.Errorf("isFullHelpFlag(%q) = false", arg)
		}
	}
}

func TestHelpFlagInAnyPosition(t *testing.T) {
	var ran bool
	run := func(*CommandContext) error { ran = true; return nil }
	config := func(cc *CommandConfig) {
		cc.Define("NAME").String().Flag("n").Flag("name").Default("").Description("a name")
		cc.Define("FORCE").Bool().Flag("force").Default(false).Description("force it")
	}
	newApp := func(withDefault bool) *harness {
		h := newHarness(nil)
		user := h.cfg.Command("user").ShortHelp("manage users")
		user.SubCommand("list").ShortHelp("list the users").Func(run).Config(config)
		h.cfg.Command("read").ShortHelp("read things").Func(run).Config(config)
		if withDefault {
			h.cfg.Command("").ShortHelp("default").Func(run).Config(config)
		}
		return h
	}

	const read, list = "Usage: app read", "Usage: app user list"
	tests := []struct {
		args        string
		withDefault bool
		help        string // expected in the output; empty means the command runs
	}{
		{args: "read -h --force", help: read},
		{args: "read --force -help --name x", help: read},
		{args: "read --h --unknown", help: read},
		{args: "read stray --help more", help: read},
		{args: "read --name=x -h --force", help: read},
		{args: "read --full-help --force", help: read},
		{args: "read --name -h", help: read},
		{args: "user list -h --force", help: list},
		{args: "user -h list", help: list},
		{args: "-h user list", help: list},
		{args: "help user list", help: list},
		{args: "--force -h read", help: read},
		{args: "--force -h", help: "read things"},
		{args: "--force -h", withDefault: true, help: "read things"},
		{args: "--help --force", withDefault: true, help: "read things"},
		{args: "read --name -h --force"},
		{args: "read -n --help --force"},
		{args: "read -- -h x"},
		{args: "read help me"},
		{args: "--name -h --force", withDefault: true},
	}

	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			ran = false
			h := newApp(tt.withDefault)
			if err := h.run(strings.Fields(tt.args)...); err != nil {
				t.Fatalf("%v (stderr: %s)", err, h.stderr.String())
			}
			if wantRun := tt.help == ""; ran != wantRun {
				t.Errorf("command ran = %v, want %v", ran, wantRun)
			}
			if !strings.Contains(h.stdout.String(), tt.help) {
				t.Errorf("output should contain %q, got %q", tt.help, h.stdout.String())
			}
		})
	}
}

func TestReservedFlagNamesFailOnRun(t *testing.T) {
	run := func(*CommandContext) error { return nil }

	for _, name := range []string{"h", "help"} {
		define := func(cc *CommandConfig) { cc.Define("HOST").String().Flag("host").Flag(name) }
		setups := map[string]func(h *harness){
			"global": func(h *harness) { h.cfg.Define("HOST").String().Flag(name) },
			"command": func(h *harness) {
				h.cfg.Command("serve").Func(run).Config(define)
			},
			"subcommand": func(h *harness) {
				h.cfg.Command("db").SubCommand("serve").Func(run).Config(define)
			},
		}

		for where, setup := range setups {
			// Any run fails, including help and unrelated commands
			for _, args := range [][]string{{"other"}, {"--help"}, nil} {
				h := newHarness(nil)
				h.cfg.Command("other").Func(run)
				setup(h)

				err := h.run(args...)
				if ExitCode(err) != ExitFailure || !IsReported(err) {
					t.Errorf("%s %q %v: code %d, err %v", where, name, args, ExitCode(err), err)
				}
				want := "help and/or h cannot be set as flag"
				if !strings.Contains(h.stderr.String(), want) {
					t.Errorf("%s %q %v: stderr %q", where, name, args, h.stderr.String())
				}
			}
		}
	}
}

func TestInvalidBoolReportedAlike(t *testing.T) {
	const reason = "invalid bool: maybe (use true/1/yes or false/0/no)"
	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"flag", nil, []string{"go", "--dry-run=maybe"}},
		{"alias", nil, []string{"go", "-n=maybe"}},
		{"env", map[string]string{"DRY": "maybe"}, []string{"go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(tt.env)
			h.cfg.Command("go").Func(func(*CommandContext) error { return nil }).Config(func(cc *CommandConfig) {
				cc.Define("DRY_RUN").Bool().Flag("dry-run").Flag("n").Env("DRY")
			})
			if err := h.run(tt.args...); err == nil {
				t.Fatal("expected an error")
			}
			out := h.stderr.String()
			if !strings.Contains(out, "-n --dry-run bool") || !strings.Contains(out, "-> "+reason) {
				t.Errorf("unexpected output:\n%s", out)
			}
			if strings.Contains(out, "unknown_flag") {
				t.Errorf("flag not matched to its definition:\n%s", out)
			}
		})
	}
}

func TestBoolValuesFromEverySource(t *testing.T) {
	values := map[string]bool{"true": true, "1": true, "yes": true, "false": false, "0": false, "no": false}
	for raw, want := range values {
		sources := map[string]struct {
			env  map[string]string
			args []string
			def  any
		}{
			"flag":    {nil, []string{"go", "--dry-run=" + raw}, !want},
			"env":     {map[string]string{"DRY": raw}, []string{"go"}, !want},
			"default": {nil, []string{"go"}, raw},
		}
		for source, tt := range sources {
			t.Run(source+" "+raw, func(t *testing.T) {
				var got bool
				h := newHarness(tt.env)
				h.cfg.Command("go").Func(func(ctx *CommandContext) error {
					got = MustGet[bool](ctx, "DRY_RUN")
					return nil
				}).Config(func(cc *CommandConfig) {
					cc.Define("DRY_RUN").Bool().Flag("dry-run").Env("DRY").Default(tt.def)
				})
				if err := h.run(tt.args...); err != nil {
					t.Fatalf("unexpected error: %v\n%s", err, h.stderr.String())
				}
				if got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			})
		}
	}
}
