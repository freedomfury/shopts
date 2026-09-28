package shopts

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, text string) *schema {
	t.Helper()
	s, err := parseSchema(text, true)
	if err != nil {
		t.Fatalf("parseSchema: %v", err)
	}
	return s
}

func schemaErr(t *testing.T, text, want string) {
	t.Helper()
	_, err := parseSchema(text, true)
	if err == nil {
		t.Fatalf("expected schema error containing %q, got none", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestQuotingRule(t *testing.T) {
	cases := []struct {
		field, want string
	}{
		{`pattern="^\d{2,4}$"`, `^\d{2,4}$`},
		{`help="Say \"hi\", then go"`, `Say "hi", then go`},
		{`pattern="^C:\\\\"`, `^C:\\`},
		{`pattern=^\d+$`, `^\d+$`},
		{`help="a\nb"`, `a\nb`},
		{`help="semi; colon, comma"`, `semi; colon, comma`},
		{`help=  padded  `, `padded`},
		{`help=`, ``},
		{`help=Use "x, y" form`, `Use "x, y" form`},
		{`help=a "x;y" b`, `a "x;y" b`},
		{`help=say "\"hi\", ok" now`, `say "\"hi\", ok" now`},
		{"help=\n  \"quoted on the next line\"", `quoted on the next line`},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			raws, err := newLexer("long=x, " + tc.field + ";").entries()
			if err != nil {
				t.Fatal(err)
			}
			if got := raws[0].fields[1].value; got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSchemaFormat(t *testing.T) {
	s := mustParse(t, `
		short=e, long=env, type=enum, enum="dev,prod", required=true, help=Target;
		short=v,long=verbose,type=flag,help=Verbose output;
		long=multi,
		  type=string,
		  description="spans
		  lines";
	`)
	if len(s.entries) != 3 {
		t.Fatalf("got %d entries", len(s.entries))
	}
	if got := s.entries[0].enum; strings.Join(got, "|") != "dev|prod" {
		t.Fatalf("enum = %q", got)
	}
	if s.entries[1].help != "Verbose output" {
		t.Fatalf("help = %q", s.entries[1].help)
	}
}

func TestSchemaErrorPositions(t *testing.T) {
	cases := []struct {
		name, schema, want string
	}{
		{"unknown field", "long=a, type=string;\n  long=b, type=string, patern=x;", `schema line 2, col 24: unknown field "patern"`},
		{"indented schema keeps original columns", "\n    long=a, type=strng;", `schema line 2, col 18: type: "strng" is not a type`},
		{"missing semicolon", "long=a, type=string;\nlong=b, type=int", `schema line 2, col 1: entry is missing its terminating ';'`},
		{"unterminated quote", `long=a, type=string, help="oops;`, `schema line 1, col 27: quoted value is never closed`},
		{"unclosed quote in bare value", `long=a, type=string, help=Use "x, y form;`, `schema line 1, col 31: quote in the value of "help" is never closed`},
		{"junk after quote", `long=a, type=string, help="x" y;`, `schema line 1, col 31: expected ',' or ';'`},
		{"missing equals", `long=a, type;`, `schema line 1, col 13: expected '=' after field name "type"`},
		{"empty field", `long=a,, type=string;`, `schema line 1, col 8: expected a field name`},
		{"empty bare value, value on next line", "long=a, type=string, help=\n  pattern=[a-z]+;", `schema line 1, col 27: value of "help" is missing on this line`},
		{"bare value over lines", "long=a, type=string, help=one\nlong=b, type=int;", `schema line 1, col 30: unquoted value of "help" runs onto the next line`},
		{"invalid UTF-8 at its column", "long=a, type=string, help=caf\xe9;", "schema line 1, col 30: schema is not valid UTF-8"},
		{"invalid UTF-8 as the first byte", "\xfflong=a, type=string;", "schema line 1, col 1: schema is not valid UTF-8"},
		{"byte order mark", "\uFEFFlong=a, type=string;", "schema line 1, col 1: schema starts with a byte order mark"},
		{"end of schema on an indented blank line", "\n      long=a, type=string,\n    ", "schema line 3, col 5: expected a field name, found end of schema"},
		{"cross-field check at field", "long=a, type=string,\n  required=true, default=x;", `schema line 2, col 26: option "a": required and default cannot both be set`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { schemaErr(t, tc.schema, tc.want) })
	}
}

func TestSchemaRules(t *testing.T) {
	cases := []struct {
		name, schema, want string
	}{
		{"empty", "  \n ", "schema is empty"},
		{"no long", "type=string;", "no 'long' field"},
		{"no type", "long=a;", "no 'type' field"},
		{"bad type", "long=a, type=str;", `"str" is not a type`},
		{"long hyphen", "long=dry-run, type=flag;", "must start with a letter"},
		{"long digit", "long=1a, type=flag;", "must start with a letter"},
		{"long underscore", "long=_a, type=flag;", "must start with a letter"},
		{"long go_shopts reserved", "long=go_shopts_x, type=flag;", "names starting with go_shopts belong to shopts"},
		{"long GO_SHOPTS reserved, any case", "long=Go_Shopts, type=flag;", "names starting with go_shopts belong to shopts"},
		{"help on two lines", "long=a, type=flag, help=\"one\ntwo\";", "help: must be a single line"},
		{"failure on two lines", "long=a, type=string, pattern=x, failure=\"one\ntwo\";", "failure: must be a single line"},
		{"count with leading zero", "long=a, type=list, maxItems=01;", `must be a whole number of at least 0, got "01"`},
		{"count with sign", "long=a, type=string, minLength=+2;", `must be a whole number of at least 0, got "+2"`},
		{"positional is gone", "long=a, type=string, positional=1;", `unknown field "positional"`},
		{"long help reserved", "long=help, type=flag;", "reserved"},
		{"long version reserved", "long=version, type=flag;", "reserved"},
		{"short H reserved", "long=a, short=H, type=flag;", "reserved"},
		{"short V reserved", "long=a, short=V, type=flag;", "reserved"},
		{"short too long", "long=a, short=ab, type=flag;", "single letter or digit"},
		{"dup short", "long=a, short=a, type=flag; long=b, short=a, type=flag;", `short flag "a" is used twice`},
		{"dup long", "long=a, type=flag; long=a, type=flag;", `long name "a" is used twice`},
		{"long differs only in case", "long=Env, type=string; long=env, type=string;", `long names "Env" and "env" differ only in case`},
		{"unclosed enum quote", `long=m, type=enum, enum="\"a,b";`, "quoted item is never closed"},
		{"unclosed list default quote", `long=l, type=list, default="\"a,b";`, "quoted item is never closed"},
		{"dup field", "long=a, type=flag, long=b;", `field "long" is given twice`},
		{"required not bool", "long=a, type=string, required=yes;", "must be true or false"},
		{"flag required", "long=a, type=flag, required=true;", "cannot be required"},
		{"flag default", "long=a, type=flag, default=false;", "a flag is true when given and false when not, so it cannot have a default"},
		{"enum needs values", "long=a, type=enum;", "needs an enum field"},
		{"enum on string", "long=a, type=string, enum=x;", `field "enum" does not apply to type string`},
		{"enum empty item", `long=a, type=enum, enum="a,,b";`, "must not be empty"},
		{"enum dup item", `long=a, type=enum, enum="a,b,a";`, `"a" is listed twice`},
		{"pattern on int", "long=a, type=int, pattern=x;", `field "pattern" does not apply to type int`},
		{"minLength on list", "long=a, type=list, minLength=1;", `field "minLength" does not apply to type list`},
		{"maxLength zero", "long=a, type=string, maxLength=0;", "at least 1"},
		{"minLength > maxLength", "long=a, type=string, minLength=3, maxLength=2;", "minLength 3 is greater than maxLength 2"},
		{"min on string", "long=a, type=string, min=1;", `field "min" does not apply to type string`},
		{"min not int", "long=a, type=int, min=1.5;", "must be a valid int"},
		{"min > max", "long=a, type=float, min=2, max=1.5;", "min 2 is greater than max 1.5"},
		{"items on string", "long=a, type=string, minItems=1;", `does not apply to type string`},
		{"minItems > maxItems", "long=a, type=list, minItems=3, maxItems=2;", "minItems 3 is greater than maxItems 2"},
		{"failure without pattern", "long=a, type=string, failure=x;", "failure is set but there is no pattern"},
		{"bad regex", "long=a, type=string, pattern=(;", "invalid regex"},
		{"regex cannot escape anchoring", "long=a, type=string, pattern=[0-9]+)|(.*;", "invalid regex: error parsing regexp: unexpected ): `[0-9]+)|(.*`"},
		{"unknown validator", "long=a, type=string, pattern={{ Nope }};", `unknown validator "Nope"`},
		{"malformed validator name", "long=a, type=string, pattern={{ Port_Number }};", `unknown validator "Port_Number"`},
		{"malformed name in define", "define=X, pattern={{ Port_Number }}; long=a, type=flag;", "must be a regex"},
		{"bad default", "long=a, type=int, default=abc;", `default "abc": must be a valid integer`},
		{"default out of range", "long=a, type=int, max=10, default=11;", "must be at most 10"},
		{"default fails pattern", "long=a, type=string, pattern=[a-z]+, default=ABC;", "must match the pattern"},
		{"list default item fails", `long=a, type=list, pattern=[a-z]+, default="ok,NO";`, `default item "NO"`},
		{"list default too many", `long=a, type=list, maxItems=1, default="a,b";`, "default allows at most 1 items, got 2"},
		{"minItems above default max", "long=a, type=list, minItems=150;", "minItems 150 is greater than maxItems 100 (the default)"},
		{"required list with maxItems=0", "long=a, type=list, required=true, maxItems=0;", "minItems 1 (required lists need at least 1 item) is greater than maxItems 0"},
		{"no options", "define=X, pattern=x;", "schema has no options"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { schemaErr(t, tc.schema, tc.want) })
	}
}

func TestDefine(t *testing.T) {
	s := mustParse(t, `
		long=ticket, type=string, pattern={{ Ticket }};
		define=Ticket, pattern=[A-Z]+-[0-9]+, failure=must look like ABC-123;
	`)
	e := s.entries[0]
	if e.match == nil || !e.match.Check("ABC-123") || e.match.Check("abc-123") {
		t.Fatal("defined pattern not applied")
	}
	if _, err := validate(e, "x"); err == nil || err.Error() != "must look like ABC-123" {
		t.Fatalf("expected define's failure message, got %v", err)
	}

	cases := []struct {
		name, schema, want string
	}{
		{"builtin name", "define=SemVer, pattern=x; long=a, type=flag;", "is a built-in validator"},
		{"twice", "define=X, pattern=x; define=X, pattern=y; long=a, type=flag;", `pattern "X" is defined twice`},
		{"bad name", "define=my_pat, pattern=x; long=a, type=flag;", "must start with a letter and contain only letters and digits"},
		{"no pattern", "define=X; long=a, type=flag;", `define "X" has no pattern`},
		{"other field", "define=X, pattern=x, help=h; long=a, type=flag;", `field "help" is not allowed in a define entry`},
		{"misspelled field", "define=X, patern=x; long=a, type=flag;", `field "patern" is not allowed in a define entry`},
		{"failure on two lines", "define=X, pattern=x, failure=\"one\ntwo\"; long=a, type=flag;", "failure: must be a single line"},
		{"field twice", "define=X, pattern=[a-z]+, pattern=[0-9]+; long=a, type=flag;", `field "pattern" is given twice`},
		{"define twice", "define=X, define=Y, pattern=x; long=a, type=flag;", `field "define" is given twice`},
		{"template", "define=X, pattern={{ SemVer }}; long=a, type=flag;", "must be a regex"},
		{"bad regex", "define=X, pattern=(; long=a, type=flag;", "invalid regex"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { schemaErr(t, tc.schema, tc.want) })
	}
}

func TestSpell(t *testing.T) {
	s := mustParse(t, "long=dry_run, type=flag; long=no_cache, type=flag; long=max_depth, type=int;")
	check := func(want ...string) {
		t.Helper()
		for i, e := range s.entries {
			if e.name != want[i] {
				t.Errorf("%s: got %q, want %q", e.long, e.name, want[i])
			}
		}
	}
	check("dry-run", "no-cache", "max-depth")
	s.spell(false)
	check("dry_run", "no_cache", "max_depth")
}

func TestSplitItems(t *testing.T) {
	cases := map[string]string{
		"a,b,c":          "a|b|c",
		" a , b ":        "a|b",
		`"a,b",c`:        "a,b|c",
		`"say \"hi\"",x`: `say "hi"|x`,
		`"  padded  ",x`: "  padded  |x",
		`a"x"b,c`:        `a"x"b|c`,
	}
	for in, want := range cases {
		got, err := splitItems(in)
		if err != nil || strings.Join(got, "|") != want {
			t.Errorf("splitItems(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for in, want := range map[string]string{
		`"a,b`:   "quoted item is never closed",
		`"a"b,c`: `expected ',' after quoted item "a"`,
	} {
		if _, err := splitItems(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("splitItems(%q): got %v, want error containing %q", in, err, want)
		}
	}
}
