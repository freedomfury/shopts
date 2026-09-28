package shopts

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// TestMain clears GO_SHOPTS_ settings from the environment, so a developer's
// exported settings cannot change what the tests see.
func TestMain(m *testing.M) {
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); strings.HasPrefix(key, "GO_SHOPTS_") {
			_ = os.Unsetenv(key)
		}
	}
	os.Exit(m.Run())
}

func run(t *testing.T, schemaText string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(append([]string{"shopts", schemaText}, args...), &out, &errOut, "v1.2.3")
	return out.String(), errOut.String(), code
}

const deploySchema = `
	short=e, long=env, type=enum, enum="dev,prod", required=true, help=Target;
	short=v, long=verbose, type=flag, help=Verbose output;
	short=p, long=port, type=int, min=1, max=65535, default=8080, help=Port;
`

func TestRunOutput(t *testing.T) {
	out, errOut, code := run(t, deploySchema, "-e", "prod", "--port=0443")
	if code != ExitOK {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	want := "SHOPTS_ENV\tprod\nSHOPTS_VERBOSE\tfalse\nSHOPTS_PORT\t443\n"
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
	if errOut != "" {
		t.Fatalf("unexpected stderr %q", errOut)
	}
}

func TestRunExplicitFlags(t *testing.T) {
	// Supporting both spellings takes two entries, and gives two variables.
	const schema = "long=dry_run, type=flag; long=no_dry_run, type=flag;"
	for _, tc := range []struct {
		args []string
		out  string
	}{
		{nil, "SHOPTS_DRY_RUN\tfalse\nSHOPTS_NO_DRY_RUN\tfalse\n"},
		{[]string{"--dry-run"}, "SHOPTS_DRY_RUN\ttrue\nSHOPTS_NO_DRY_RUN\tfalse\n"},
		{[]string{"--no-dry-run"}, "SHOPTS_DRY_RUN\tfalse\nSHOPTS_NO_DRY_RUN\ttrue\n"},
	} {
		if out, errOut, code := run(t, schema, tc.args...); code != ExitOK || out != tc.out {
			t.Errorf("%q: code %d, out %q, stderr %q", tc.args, code, out, errOut)
		}
	}
}

func TestRunNotEmitted(t *testing.T) {
	out, _, code := run(t, "long=a, type=string; long=b, type=list; long=c, type=flag;")
	if code != ExitOK || out != "SHOPTS_C\tfalse\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestRunDefaults(t *testing.T) {
	out, _, code := run(t, `long=n, type=int, default=007; long=b, type=bool, default=T; long=l, type=list, default="x,y";`)
	if code != ExitOK || out != "SHOPTS_N\t7\nSHOPTS_B\ttrue\nSHOPTS_L\tx,y\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestRunErrorsWriteNothingToStdout(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		args   []string
		code   int
		stderr string
	}{
		{"missing required option", deploySchema, nil, ExitArgs, "error: missing required option --env"},
		{"bad value", deploySchema, []string{"-e", "qa"}, ExitArgs, "invalid value for --env: must be one of: dev, prod"},
		{"bare word", deploySchema, []string{"-e", "dev", "web"}, ExitArgs, `error: unrecognized bare word "web"`},
		{"after --", deploySchema, []string{"-e", "dev", "--", "-x"}, ExitArgs, `error: unrecognized bare word "-x" after --`},
		{"bad list item", "long=f, type=list, pattern=[a-z]+;", []string{"--f", "ok", "--f", "UP"}, ExitArgs, "invalid value for --f: must match"},
		{"list too few", "long=t, type=list, minItems=2;", []string{"--t", "a"}, ExitArgs, "--t needs at least 2 items, got 1"},
		{"list too many", "long=t, type=list;", slices.Repeat([]string{"--t=x"}, 101), ExitArgs, "--t allows at most 100 items, got 101"},
		{"required list implicit min", "long=t, type=list, required=true;", nil, ExitArgs, "missing required option --t"},
		{"required empty", "long=t, type=string, required=true;", []string{"--t="}, ExitArgs, "--t requires a non-empty value"},
		{"required list empty", "long=t, type=list, required=true;", []string{"--t="}, ExitArgs, "--t requires a non-empty value"},
		{"newline", "long=t, type=string;", []string{"--t=a\nb"}, ExitArgs, "must not contain a newline"},
		{"all errors reported", deploySchema, []string{"--nope", "-e", "qa", "-p", "x"}, ExitArgs, "unknown option --nope\nerror: invalid value for --env"},
		{"schema error", "long=a, type=nope;", nil, ExitSchema, "shopts: schema line 1, col 14"},
		{"schema error before help", "long=a, type=nope;", []string{"-H"}, ExitSchema, "schema line"},
		{"help", deploySchema, []string{"-H"}, ExitStop, "Usage: [OPTIONS]\n"},
		{"help anywhere", deploySchema, []string{"web", "--bogus", "--help"}, ExitStop, "Options:"},
		{"version anywhere", deploySchema, []string{"web", "-V"}, ExitStop, "shopts v1.2.3\n"},
		{"missing value reported once", "long=a, type=string, required=true;", []string{"--a"}, ExitArgs, "error: --a requires a value\nUsage:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, code := run(t, tc.schema, tc.args...)
			if code != tc.code {
				t.Fatalf("code %d, want %d; stderr %q", code, tc.code, errOut)
			}
			if out != "" {
				t.Fatalf("stdout must be empty, got %q", out)
			}
			if !strings.Contains(errOut, tc.stderr) {
				t.Fatalf("stderr %q does not contain %q", errOut, tc.stderr)
			}
		})
	}
}

func TestRunArgErrorFormat(t *testing.T) {
	t.Setenv("GO_SHOPTS_NAME", "deploy.sh")
	_, errOut, _ := run(t, deploySchema, "-e", "qa")
	want := "deploy.sh: invalid value for --env: must be one of: dev, prod\n" +
		"Usage: deploy.sh [OPTIONS]\n" +
		"Try 'deploy.sh --help' for more information.\n"
	if errOut != want {
		t.Fatalf("got:\n%s\nwant:\n%s", errOut, want)
	}
}

func TestRunHelp(t *testing.T) {
	t.Setenv("GO_SHOPTS_NAME", "deploy.sh")
	_, errOut, code := run(t, deploySchema+`
		long=dry_run, type=flag, help=Show what would happen, description="Prints each step
		  without running it.";
	`, "--help")
	if code != ExitStop {
		t.Fatalf("code %d", code)
	}
	want := `Usage: deploy.sh [OPTIONS]

Options:
  -e, --env <value>    Target; enum; required; allowed: dev, prod
  -v, --verbose        Verbose output; flag
  -p, --port <value>   Port; int; default: 8080; min: 1; max: 65535
      --dry-run        Show what would happen; flag
                       Prints each step
                       without running it.
  -H, --help           Show this help
`
	if errOut != want {
		t.Fatalf("got:\n%s\nwant:\n%s", errOut, want)
	}
}

func TestRunToolLevel(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run([]string{"shopts"}, &out, &errOut, "v1"); code != ExitSchema || !strings.Contains(errOut.String(), "missing schema") {
		t.Fatalf("no schema: code %d, stderr %q", code, errOut.String())
	}
	for _, arg := range []string{"-V", "--version"} {
		errOut.Reset()
		if code := Run([]string{"shopts", arg}, &out, &errOut, "v1"); code != ExitStop || errOut.String() != "shopts v1\n" {
			t.Fatalf("%s: code %d, stderr %q", arg, code, errOut.String())
		}
	}
	for _, arg := range []string{"-H", "--help"} {
		errOut.Reset()
		if code := Run([]string{"shopts", arg}, &out, &errOut, "v1"); code != ExitStop || !strings.HasPrefix(errOut.String(), "usage: shopts SCHEMA [ARGS...]\n") {
			t.Fatalf("%s: code %d, stderr %q", arg, code, errOut.String())
		}
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty, got %q", out.String())
	}
}

func TestRunSettings(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
		out  string
	}{
		{"prefix", map[string]string{"GO_SHOPTS_PREFIX": "OPT_"}, []string{"--userName=a"}, "OPT_USERNAME\ta\n"},
		{"empty prefix", map[string]string{"GO_SHOPTS_PREFIX": ""}, []string{"--userName=a"}, "USERNAME\ta\n"},
		{"no upcase", map[string]string{"GO_SHOPTS_UPCASE": "0"}, []string{"--userName=a"}, "SHOPTS_userName\ta\n"},
		{"out delim", map[string]string{"GO_SHOPTS_OUT_DELIM": "="}, []string{"--userName=a"}, "SHOPTS_USERNAME=a\n"},
		{"list delim", map[string]string{"GO_SHOPTS_LIST_DELIM": ":"}, []string{"--tags=a", "--tags=b"}, "SHOPTS_TAGS\ta:b\n"},
		{"named null", map[string]string{"GO_SHOPTS_OUT_DELIM": "{{ null }}"}, []string{"--userName=a\tb"}, "SHOPTS_USERNAME\x00a\tb\n"},
		{"named tab, any spacing", map[string]string{"GO_SHOPTS_OUT_DELIM": "{{tab}}"}, []string{"--userName=a"}, "SHOPTS_USERNAME\ta\n"},
		{"named list delim", map[string]string{"GO_SHOPTS_LIST_DELIM": "{{ null }}"}, []string{"--tags=a", "--tags=b"}, "SHOPTS_TAGS\ta\x00b\n"},
		{"any string", map[string]string{"GO_SHOPTS_OUT_DELIM": "AAA"}, []string{"--userName=a"}, "SHOPTS_USERNAMEAAAa\n"},
		{"value contains the delimiter", nil, []string{"--userName=a\tb"}, "SHOPTS_USERNAME\ta\tb\n"},
		{"dashes by default", nil, []string{"--max-depth=2"}, "SHOPTS_MAX_DEPTH\t2\n"},
		{"no dashes", map[string]string{"GO_SHOPTS_DASH": "0"}, []string{"--max_depth=2"}, "SHOPTS_MAX_DEPTH\t2\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			out, errOut, code := run(t, "long=userName, type=string; long=tags, type=list; long=max_depth, type=int;", tc.args...)
			if code != ExitOK || out != tc.out {
				t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
			}
		})
	}

	bad := []struct{ key, val, want string }{
		{"GO_SHOPTS_PREFIX", "GO_SHOPTS_X", "must not start with GO_SHOPTS_"},
		{"GO_SHOPTS_PREFIX", "1BAD", "not a valid shell variable prefix"},
		{"GO_SHOPTS_UPCASE", "maybe", "must be 1 or 0"},
		{"GO_SHOPTS_DASH", "maybe", "must be 1 or 0"},
		{"GO_SHOPTS_OUT_DELIM", "\n", "must not contain a newline"},
		{"GO_SHOPTS_OUT_DELIM", "{{ nope }}", `GO_SHOPTS_OUT_DELIM: unknown delimiter "{{ nope }}" (named delimiters: {{ tab }}, {{ null }})`},
		{"GO_SHOPTS_LIST_DELIM", "a\nb", "must not contain a newline"},
	}
	for _, tc := range bad {
		t.Run(tc.key+"="+tc.val, func(t *testing.T) {
			t.Setenv(tc.key, tc.val)
			out, errOut, code := run(t, "long=a, type=string;")
			if code != ExitFailure || out != "" || !strings.Contains(errOut, tc.want) {
				t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
			}
		})
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestRunStopPrecedence(t *testing.T) {
	cases := []struct {
		name   string
		env    map[string]string
		schema string
		args   []string
		code   int
		stderr string
	}{
		{"version ignores a broken schema", nil, "long=a, type=nope;", []string{"-V"}, ExitStop, "shopts v1.2.3\n"},
		{"version ignores bad settings", map[string]string{"GO_SHOPTS_DASH": "maybe", "GO_SHOPTS_PREFIX": "1BAD"}, deploySchema, []string{"--version"}, ExitStop, "shopts v1.2.3\n"},
		{"first of -V/-H wins: version", nil, deploySchema, []string{"-V", "-H"}, ExitStop, "shopts v1.2.3\n"},
		{"first of -H/-V wins: help", nil, deploySchema, []string{"-H", "-V"}, ExitStop, "Usage:"},
		{"version after -- as a value", nil, "long=a, type=string;", []string{"--a", "--", "-V"}, ExitStop, "shopts v1.2.3\n"},
		{"help ignores output settings", map[string]string{"GO_SHOPTS_PREFIX": "1BAD", "GO_SHOPTS_OUT_DELIM": "{{ nope }}"}, deploySchema, []string{"--help"}, ExitStop, "Usage:"},
		{"help needs GO_SHOPTS_DASH", map[string]string{"GO_SHOPTS_DASH": "maybe"}, deploySchema, []string{"--help"}, ExitFailure, "GO_SHOPTS_DASH"},
		{"help needs the schema", nil, "long=a, type=nope;", []string{"--help"}, ExitSchema, "schema line"},
		{"output settings checked without help", map[string]string{"GO_SHOPTS_PREFIX": "1BAD"}, deploySchema, []string{"web", "-e", "dev"}, ExitFailure, "GO_SHOPTS_PREFIX"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			out, errOut, code := run(t, tc.schema, tc.args...)
			if code != tc.code || out != "" || !strings.Contains(errOut, tc.stderr) {
				t.Fatalf("code %d (want %d), out %q, stderr %q", code, tc.code, out, errOut)
			}
		})
	}
}

func TestRunPanicExitsOne(t *testing.T) {
	var errOut bytes.Buffer
	// A nil stdout makes the final write panic.
	code := Run([]string{"shopts", "long=a, type=flag;"}, nil, &errOut, "v1")
	if code != ExitFailure || !strings.Contains(errOut.String(), "shopts: internal error:") {
		t.Fatalf("code %d, stderr %q", code, errOut.String())
	}
}

func TestRunWriteError(t *testing.T) {
	var errOut bytes.Buffer
	code := Run([]string{"shopts", "long=a, type=flag;"}, failWriter{}, &errOut, "v1")
	if code != ExitFailure || !strings.Contains(errOut.String(), "broken pipe") {
		t.Fatalf("code %d, stderr %q", code, errOut.String())
	}
}
