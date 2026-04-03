package shopts

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

const sampleSchema = `
short=u;long=username;required=true;type=string;help=Username;minLength=3;
short=p;long=pass;required=true;type=string;help=Password;minLength=6;
short=v;long=verbose;required=false;type=flag;help=Verbose;
`

func TestRun_Help(t *testing.T) {
	var buf bytes.Buffer
	if err := Run([]string{"shopts", sampleSchema, "--help"}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Usage: shopts") {
		t.Fatalf("expected help output, got: %q", buf.String())
	}
}

func TestRun_ValidationAndOutput(t *testing.T) {
	var buf bytes.Buffer
	err := Run([]string{"shopts", sampleSchema, "-u", "alice", "-p", "s3cret", "-v"}, &buf)
	if err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "GO_SHOPTS_username") {
		t.Fatalf("expected key output, got: %q", got)
	}
	if !strings.Contains(got, "alice") {
		t.Fatalf("expected value output, got: %q", got)
	}
}

func TestParseSchema_Invalid(t *testing.T) {
	_, err := parseSchema("short=x;type=invalid;long=foo;")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// ---------------------------------------------------------------------------
// Schema parsing tests
// ---------------------------------------------------------------------------

func TestParseSchema_Empty(t *testing.T) {
	_, err := parseSchema("")
	if err == nil {
		t.Fatal("expected error for empty schema")
	}
}

func TestParseSchema_MissingSemicolon(t *testing.T) {
	_, err := parseSchema("long=foo;type=string")
	if err == nil {
		t.Fatal("expected error for missing trailing semicolon")
	}
}

func TestParseSchema_MissingLong(t *testing.T) {
	_, err := parseSchema("short=x;type=string;")
	if err == nil {
		t.Fatal("expected error for missing long name")
	}
}

func TestParseSchema_MissingType(t *testing.T) {
	_, err := parseSchema("long=foo;")
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestParseSchema_DuplicateShort(t *testing.T) {
	schema := "short=a;long=foo;type=string;\nshort=a;long=bar;type=string;"
	_, err := parseSchema(schema)
	if err == nil {
		t.Fatal("expected error for duplicate short flag")
	}
}

func TestParseSchema_DuplicateLong(t *testing.T) {
	schema := "short=a;long=foo;type=string;\nshort=b;long=foo;type=string;"
	_, err := parseSchema(schema)
	if err == nil {
		t.Fatal("expected error for duplicate long name")
	}
}

func TestParseSchema_RequiredWithDefault(t *testing.T) {
	_, err := parseSchema("long=foo;type=string;required=true;default=bar;")
	if err == nil {
		t.Fatal("expected error for required + default")
	}
}

func TestParseSchema_EnumRequiresEnumType(t *testing.T) {
	_, err := parseSchema("long=foo;type=string;enum=a,b;")
	if err == nil {
		t.Fatal("expected error for enum on non-enum type")
	}
}

func TestParseSchema_EnumTypeMissingValues(t *testing.T) {
	_, err := parseSchema("long=foo;type=enum;")
	if err == nil {
		t.Fatal("expected error for enum type without values")
	}
}

func TestParseSchema_FlagRejectsStringConstraints(t *testing.T) {
	_, err := parseSchema("long=foo;type=flag;minLength=1;")
	if err == nil {
		t.Fatal("expected error for flag with minLength")
	}
}

func TestParseSchema_NumericRejectsStringConstraints(t *testing.T) {
	cases := []struct {
		name   string
		schema string
	}{
		{"int+minLength", "long=n;type=int;minLength=1;"},
		{"float+maxLength", "long=n;type=float;maxLength=10;"},
		{"bool+pattern", "long=b;type=bool;pattern=^true$;"},
		{"int+pattern", "long=n;type=int;pattern=^[0-9]+$;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSchema(tc.schema)
			if err == nil {
				t.Fatal("expected error for string constraint on numeric type")
			}
		})
	}
}

func TestParseSchema_MinLengthGreaterThanMaxLength(t *testing.T) {
	_, err := parseSchema("long=foo;type=string;minLength=10;maxLength=3;")
	if err == nil {
		t.Fatal("expected error for minLength > maxLength")
	}
}

func TestParseSchema_MinItemsMaxItems(t *testing.T) {
	_, err := parseSchema("long=foo;type=list;minItems=5;maxItems=2;")
	if err == nil {
		t.Fatal("expected error for minItems > maxItems")
	}
}

func TestParseSchema_ItemsOnNonList(t *testing.T) {
	_, err := parseSchema("long=foo;type=string;minItems=1;")
	if err == nil {
		t.Fatal("expected error for minItems on non-list type")
	}
}

func TestParseSchema_InvalidDefault(t *testing.T) {
	_, err := parseSchema("long=foo;type=int;default=abc;")
	if err == nil {
		t.Fatal("expected error for invalid default")
	}
}

func TestParseSchema_QuotedValues(t *testing.T) {
	schema := `long=mode;type=enum;enum="dev,prod,test";help="Mode; selects env";`
	entries, err := parseSchema(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %v", entries[0].Enum)
	}
	if entries[0].Help != "Mode; selects env" {
		t.Fatalf("unexpected help: %q", entries[0].Help)
	}
}

func TestParseSchema_Dedent(t *testing.T) {
	schema := `
        long=foo;type=string;
        long=bar;type=int;
    `
	entries, err := parseSchema(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestParseSchema_PatternCompiles(t *testing.T) {
	schema := `long=email;type=string;pattern=^[^@]+@[^@]+$;`
	entries, err := parseSchema(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].CompiledPattern == nil {
		t.Fatal("expected compiled pattern")
	}
}

func TestParseSchema_InvalidPattern(t *testing.T) {
	_, err := parseSchema(`long=foo;type=string;pattern=[invalid;`)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestParseSchema_FailureWithoutPattern(t *testing.T) {
	_, err := parseSchema(`long=foo;type=string;failure=bad format;`)
	if err == nil {
		t.Fatal("expected error for failure without pattern")
	}
}

// ---------------------------------------------------------------------------
// Arg parsing tests
// ---------------------------------------------------------------------------

func TestParseArgs_LongEquals(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	vals, err := parseArgs([]string{"--name=alice"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["name"] != "alice" {
		t.Fatalf("expected alice, got %q", vals["name"])
	}
}

func TestParseArgs_ShortEquals(t *testing.T) {
	schema, _ := parseSchema("short=n;long=name;type=string;")
	vals, err := parseArgs([]string{"-n=bob"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["name"] != "bob" {
		t.Fatalf("expected bob, got %q", vals["name"])
	}
}

func TestParseArgs_ShortSeparate(t *testing.T) {
	schema, _ := parseSchema("short=n;long=name;type=string;")
	vals, err := parseArgs([]string{"-n", "charlie"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["name"] != "charlie" {
		t.Fatalf("expected charlie, got %q", vals["name"])
	}
}

func TestParseArgs_Flag(t *testing.T) {
	schema, _ := parseSchema("short=v;long=verbose;type=flag;")
	vals, err := parseArgs([]string{"-v"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["verbose"] != "true" {
		t.Fatalf("expected true, got %q", vals["verbose"])
	}
}

func TestParseArgs_FlagRejectsValue(t *testing.T) {
	schema, _ := parseSchema("short=v;long=verbose;type=flag;")
	_, err := parseArgs([]string{"--verbose=yes"}, schema)
	if err == nil {
		t.Fatal("expected error for flag with inline value")
	}
}

func TestParseArgs_DoubleDash(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	vals, err := parseArgs([]string{"--name=x", "--"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["name"] != "x" {
		t.Fatalf("expected x, got %q", vals["name"])
	}
}

func TestParseArgs_DoubleDashWithTrailingArg(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	_, err := parseArgs([]string{"--name=x", "--", "extra"}, schema)
	if err == nil {
		t.Fatal("expected error for positional arg after --")
	}
}

func TestParseArgs_UnknownOption(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	_, err := parseArgs([]string{"--unknown=x"}, schema)
	if err == nil {
		t.Fatal("expected error for unknown option")
	}
}

func TestParseArgs_PositionalRejected(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	_, err := parseArgs([]string{"positional"}, schema)
	if err == nil {
		t.Fatal("expected error for positional arg")
	}
}

func TestParseArgs_ShortBundleRejected(t *testing.T) {
	schema, _ := parseSchema("short=a;long=aa;type=flag;\nshort=b;long=bb;type=flag;")
	_, err := parseArgs([]string{"-ab"}, schema)
	if err == nil {
		t.Fatal("expected error for short option bundle")
	}
}

func TestParseArgs_MissingValue(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	_, err := parseArgs([]string{"--name"}, schema)
	if err == nil {
		t.Fatal("expected error for option without value")
	}
}

func TestParseArgs_List(t *testing.T) {
	schema, _ := parseSchema("short=t;long=tags;type=list;")
	vals, err := parseArgs([]string{"-t", "a", "-t", "b", "-t", "c"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["tags"] != "a,b,c" {
		t.Fatalf("expected a,b,c, got %q", vals["tags"])
	}
}

func TestParseArgs_ListCustomDelimiter(t *testing.T) {
	t.Setenv("GO_SHOPTS_LIST_DELIM", ":")
	schema, _ := parseSchema("long=tags;type=list;")
	vals, err := parseArgs([]string{"--tags=a", "--tags=b"}, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["tags"] != "a:b" {
		t.Fatalf("expected a:b, got %q", vals["tags"])
	}
}

func TestParseArgs_ListMinItems(t *testing.T) {
	schema, _ := parseSchema("long=tags;type=list;minItems=2;")
	_, err := parseArgs([]string{"--tags=a"}, schema)
	if err == nil {
		t.Fatal("expected error for too few list items")
	}
}

func TestParseArgs_ListMaxItems(t *testing.T) {
	schema, _ := parseSchema("long=tags;type=list;maxItems=1;")
	_, err := parseArgs([]string{"--tags=a", "--tags=b"}, schema)
	if err == nil {
		t.Fatal("expected error for too many list items")
	}
}

// ---------------------------------------------------------------------------
// Validation tests
// ---------------------------------------------------------------------------

func TestValidate_RequiredMissing(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;required=true;")
	problems := validateParsedValues(schema, map[string]string{})
	if len(problems) == 0 {
		t.Fatal("expected validation error for missing required option")
	}
}

func TestValidate_RequiredEmptyString(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;required=true;")
	problems := validateParsedValues(schema, map[string]string{"name": ""})
	if len(problems) == 0 {
		t.Fatal("expected validation error for empty required option")
	}
}

func TestValidate_NewlineRejected(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	problems := validateParsedValues(schema, map[string]string{"name": "a\nb"})
	if len(problems) == 0 {
		t.Fatal("expected validation error for newline in value")
	}
}

func TestValidate_NulRejected(t *testing.T) {
	schema, _ := parseSchema("long=name;type=string;")
	problems := validateParsedValues(schema, map[string]string{"name": "a\x00b"})
	if len(problems) == 0 {
		t.Fatal("expected validation error for NUL in value")
	}
}

func TestValidateValue_Int(t *testing.T) {
	entry := schemaEntry{Long: "n", Type: "int"}
	if err := validateValue(entry, "42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateValue(entry, "abc"); err == nil {
		t.Fatal("expected error for non-int")
	}
}

func TestValidateValue_Float(t *testing.T) {
	entry := schemaEntry{Long: "f", Type: "float"}
	if err := validateValue(entry, "3.14"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateValue(entry, "abc"); err == nil {
		t.Fatal("expected error for non-float")
	}
}

func TestValidateValue_Bool(t *testing.T) {
	entry := schemaEntry{Long: "b", Type: "bool"}
	for _, v := range []string{"true", "false", "1", "0", "TRUE"} {
		if err := validateValue(entry, v); err != nil {
			t.Fatalf("unexpected error for %q: %v", v, err)
		}
	}
	if err := validateValue(entry, "maybe"); err == nil {
		t.Fatal("expected error for non-bool")
	}
}

func TestValidateValue_Enum(t *testing.T) {
	entry := schemaEntry{Long: "e", Type: "enum", Enum: []string{"a", "b", "c"}}
	if err := validateValue(entry, "a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateValue(entry, "d"); err == nil {
		t.Fatal("expected error for invalid enum value")
	}
}

func TestValidateValue_MinMaxLength(t *testing.T) {
	min, max := 3, 10
	entry := schemaEntry{Long: "s", Type: "string", MinLength: &min, MaxLength: &max}
	if err := validateValue(entry, "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateValue(entry, "ab"); err == nil {
		t.Fatal("expected error for too short")
	}
	if err := validateValue(entry, "12345678901"); err == nil {
		t.Fatal("expected error for too long")
	}
}

func TestValidateValue_Pattern(t *testing.T) {
	re := regexp.MustCompile(`^\w+@\w+\.\w+$`)
	entry := schemaEntry{Long: "e", Type: "string", Pattern: re.String(), CompiledPattern: re}
	if err := validateValue(entry, "a@b.c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateValue(entry, "notanemail"); err == nil {
		t.Fatal("expected error for pattern mismatch")
	}
}

func TestValidateValue_PatternWithFailureMessage(t *testing.T) {
	re := regexp.MustCompile(`^\d+$`)
	entry := schemaEntry{Long: "e", Type: "string", Pattern: re.String(), Failure: "must be numeric", CompiledPattern: re}
	err := validateValue(entry, "abc")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must be numeric") {
		t.Fatalf("expected custom failure message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Resolved value tests
// ---------------------------------------------------------------------------

func TestResolvedValue_Provided(t *testing.T) {
	entry := schemaEntry{Long: "name", Type: "string"}
	val, emit := resolvedValue(entry, map[string]string{"name": "alice"})
	if !emit || val != "alice" {
		t.Fatalf("expected alice/true, got %q/%v", val, emit)
	}
}

func TestResolvedValue_Default(t *testing.T) {
	entry := schemaEntry{Long: "name", Type: "string", Default: "bob"}
	val, emit := resolvedValue(entry, map[string]string{})
	if !emit || val != "bob" {
		t.Fatalf("expected bob/true, got %q/%v", val, emit)
	}
}

func TestResolvedValue_FlagDefault(t *testing.T) {
	entry := schemaEntry{Long: "verbose", Type: "flag"}
	val, emit := resolvedValue(entry, map[string]string{})
	if !emit || val != "false" {
		t.Fatalf("expected false/true, got %q/%v", val, emit)
	}
}

func TestResolvedValue_NotEmitted(t *testing.T) {
	entry := schemaEntry{Long: "name", Type: "string"}
	_, emit := resolvedValue(entry, map[string]string{})
	if emit {
		t.Fatal("expected no emission for optional without default")
	}
}

// ---------------------------------------------------------------------------
// shVarName tests
// ---------------------------------------------------------------------------

func TestShVarName_Basic(t *testing.T) {
	name, err := shVarName("foo", "PREFIX_", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "PREFIX_foo" {
		t.Fatalf("expected PREFIX_foo, got %q", name)
	}
}

func TestShVarName_Upcase(t *testing.T) {
	name, err := shVarName("foo-bar", "P_", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "P_FOO_BAR" {
		t.Fatalf("expected P_FOO_BAR, got %q", name)
	}
}

func TestShVarName_HyphenSanitized(t *testing.T) {
	name, err := shVarName("my-opt", "X_", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "X_my_opt" {
		t.Fatalf("expected X_my_opt, got %q", name)
	}
}

func TestShVarName_Empty(t *testing.T) {
	_, err := shVarName("", "P_", false)
	if err == nil {
		t.Fatal("expected error for empty long name")
	}
}

// ---------------------------------------------------------------------------
// Helper tests
// ---------------------------------------------------------------------------

func TestSplitEnum(t *testing.T) {
	got := splitEnum(`a,b,c`)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestSplitEnum_Escaped(t *testing.T) {
	got := splitEnum(`a\,b,c`)
	if len(got) != 2 || got[0] != "a,b" || got[1] != "c" {
		t.Fatalf("unexpected: %v", got)
	}
}

func TestWantsHelp(t *testing.T) {
	if !wantsHelp([]string{"-h"}) {
		t.Fatal("expected true for -h")
	}
	if !wantsHelp([]string{"--help"}) {
		t.Fatal("expected true for --help")
	}
	if wantsHelp([]string{"--name=x"}) {
		t.Fatal("expected false")
	}
}

func TestSplitFields_BackslashOutsideQuotes(t *testing.T) {
	// Backslash outside quotes should be literal, not an escape character.
	fields, err := splitFields(`pattern=\d+;long=foo;`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d: %v", len(fields), fields)
	}
	if fields[0] != `pattern=\d+` {
		t.Fatalf("expected pattern=\\d+, got %q", fields[0])
	}
}
