package shopts

import (
	"strings"
	"testing"
)

const argsSchema = `
	short=u, long=user, type=string;
	short=v, long=verbose, type=flag;
	short=t, long=tag, type=list;
`

func scanArgs(t *testing.T, schemaText string, args ...string) (scanned, map[string][]string) {
	t.Helper()
	s := mustParse(t, schemaText)
	sc := scan(s, args)
	byName := map[string][]string{}
	for e, v := range sc.values {
		byName[e.long] = v
	}
	return sc, byName
}

func TestScanForms(t *testing.T) {
	for _, args := range [][]string{
		{"--user", "alice"}, {"--user=alice"}, {"-u", "alice"}, {"-u=alice"},
	} {
		sc, v := scanArgs(t, argsSchema, args...)
		if len(sc.errs) > 0 || strings.Join(v["user"], "") != "alice" {
			t.Errorf("%q: values %v, errs %v", args, v, sc.errs)
		}
	}
}

func TestScanValueThatLooksLikeOption(t *testing.T) {
	_, v := scanArgs(t, argsSchema, "-u", "--verbose")
	if v["user"][0] != "--verbose" || v["verbose"] != nil {
		t.Fatalf("got %v", v)
	}
}

func TestScanEmptyInlineValue(t *testing.T) {
	_, v := scanArgs(t, argsSchema, "--user=", "-v")
	if v["user"][0] != "" || v["verbose"][0] != "true" {
		t.Fatalf("got %v", v)
	}
}

func TestScanRejectsBareWords(t *testing.T) {
	sc, v := scanArgs(t, argsSchema, "deploy", "-v", "-", "--", "-b", "--user=x")
	want := []string{`unrecognized bare word "deploy"`, `unrecognized bare word "-"`, `unrecognized bare word "-b" after --`, `unrecognized bare word "--user=x" after --`}
	if strings.Join(sc.errs, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", sc.errs, want)
	}
	if v["verbose"][0] != "true" || v["user"] != nil {
		t.Fatalf("got %v", v)
	}
}

func TestScanHelpVersionAnywhere(t *testing.T) {
	for _, args := range [][]string{{"-H"}, {"x", "--help"}, {"--bogus", "-H"}, {"-u", "-H"}} {
		if sc, _ := scanArgs(t, argsSchema, args...); !sc.help || len(sc.errs) > 0 {
			t.Errorf("%q: help not detected", args)
		}
	}
	for _, args := range [][]string{{"-V"}, {"x", "--version"}, {"--bogus", "-V"}} {
		if sc, _ := scanArgs(t, argsSchema, args...); !sc.version {
			t.Errorf("%q: version not detected", args)
		}
	}
	if sc, _ := scanArgs(t, argsSchema, "--", "-H"); sc.help {
		t.Error("-H after -- must be a bare argument")
	}
	if sc, _ := scanArgs(t, argsSchema, "-u", "--", "-H"); !sc.help {
		t.Error("-- as the value of -u is not the end of options; -H after it is help")
	}
	if sc, _ := scanArgs(t, argsSchema, "-v", "--", "-H"); sc.help {
		t.Error("-- after a flag is the end of options; -H after it is a bare argument")
	}
	if sc, _ := scanArgs(t, argsSchema, "-h"); sc.help || len(sc.errs) != 1 {
		t.Error("-h is not help")
	}
}

func TestScanErrors(t *testing.T) {
	cases := []struct {
		schema string
		args   []string
		want   string
	}{
		{argsSchema, []string{"--nope"}, "unknown option --nope"},
		{argsSchema, []string{"-x"}, "unknown option -x"},
		{argsSchema, []string{"-uv"}, "short options cannot be combined: -uv"},
		{argsSchema, []string{"--verbose=true"}, "--verbose does not take a value"},
		{argsSchema, []string{"--user"}, "--user requires a value"},
		{argsSchema, []string{"-u", "a", "--user", "b"}, "--user is given more than once"},
		{argsSchema, []string{"-v", "-v"}, "--verbose is given more than once"},
		{argsSchema, []string{"--nope"}, "unknown option --nope"},
		{argsSchema, []string{"-é"}, "unknown option -é"},
		{argsSchema, []string{"--usér=x"}, "unknown option --usér"},
		{"long=a, type=string;", []string{"stray"}, `unrecognized bare word "stray"`},
		{"long=a, type=string;", []string{"--", "stray"}, `unrecognized bare word "stray" after --`},
	}
	for _, tc := range cases {
		sc, _ := scanArgs(t, tc.schema, tc.args...)
		if len(sc.errs) != 1 || sc.errs[0] != tc.want {
			t.Errorf("%q: got %q, want %q", tc.args, sc.errs, tc.want)
		}
	}
}

func TestScanDashes(t *testing.T) {
	const schema = "long=dry_run, type=flag; long=no_cache, type=flag; long=max_depth, type=int;"
	sc, v := scanArgs(t, schema, "--dry-run", "--no-cache", "--max-depth=3")
	if len(sc.errs) > 0 || v["dry_run"][0] != "true" || v["no_cache"][0] != "true" || v["max_depth"][0] != "3" {
		t.Fatalf("got %v, errs %v", v, sc.errs)
	}
	for _, tc := range []struct{ arg, want string }{
		{"--dry_run", "unknown option --dry_run"},
		{"--no-dry-run", "unknown option --no-dry-run"}, // nothing is implied
		{"--cache", "unknown option --cache"},
		{"--no-cache=1", "--no-cache does not take a value"},
	} {
		if sc, _ := scanArgs(t, schema, tc.arg); len(sc.errs) != 1 || sc.errs[0] != tc.want {
			t.Errorf("%s: got %q, want %q", tc.arg, sc.errs, tc.want)
		}
	}
}

func TestScanListCollects(t *testing.T) {
	_, v := scanArgs(t, argsSchema, "-t", "a", "--tag=b", "-t", "c")
	if strings.Join(v["tag"], ",") != "a,b,c" {
		t.Fatalf("got %v", v["tag"])
	}
}
