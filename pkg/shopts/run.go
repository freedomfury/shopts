// Package shopts parses a script's arguments against a schema and writes one
// KEY<TAB>VALUE line per option to stdout. That output and the exit code are
// the whole contract.
package shopts

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Exit codes.
const (
	ExitOK      = 0 // parsed; stdout holds KEY<TAB>VALUE lines
	ExitFailure = 1 // bad GO_SHOPTS_ setting, write error
	ExitSchema  = 2 // schema error: the script author's mistake
	ExitArgs    = 3 // argument error: the user's mistake
	ExitStop    = 7 // help or version was printed; the caller should stop
)

// Run parses argv (program, schema, args...) and returns the exit code.
// Nothing is written to stdout unless parsing succeeds.
//
// Checks run in this order, and the first to fail decides the exit code:
// version (7), the settings help shows (1), schema (2), help (7), the other
// settings (1), arguments (3).
func Run(argv []string, stdout, stderr io.Writer, version string) (code int) {
	fail := func(code int, format string, args ...any) int {
		_, _ = fmt.Fprintf(stderr, "shopts: "+format+"\n", args...)
		return code
	}
	// An internal error must not look like a schema error, which is the
	// exit code Go gives an unrecovered panic.
	defer func() {
		if r := recover(); r != nil {
			code = fail(ExitFailure, "internal error: %v", r)
		}
	}()
	if len(argv) < 2 {
		return fail(ExitSchema, "missing schema\nusage: shopts SCHEMA [ARGS...]")
	}
	printVersion := func() int {
		if _, err := fmt.Fprintf(stderr, "shopts %s\n", version); err != nil {
			return ExitFailure
		}
		return ExitStop
	}
	// -V/--version needs neither settings nor the schema. The first of
	// -H/-V before "--" decides; help needs the schema, so it waits.
	// (A "--" that is an option's value is handled again by scan.)
	for _, arg := range argv[1:] {
		if arg == "--" || arg == "-H" || arg == "--help" {
			break
		}
		if arg == "-V" || arg == "--version" {
			return printVersion()
		}
	}
	// Without a schema, -H/--help describes shopts itself. A schema never
	// starts with '-', so this is unambiguous.
	if argv[1] == "-H" || argv[1] == "--help" {
		if _, err := fmt.Fprintln(stderr, "usage: shopts SCHEMA [ARGS...]\n\n"+
			"Parses ARGS against SCHEMA and prints one KEY<TAB>VALUE line per option.\n"+
			"See https://github.com/freedomfury/shopts"); err != nil {
			return ExitFailure
		}
		return ExitStop
	}

	cfg, err := loadConfig()
	if err != nil {
		return fail(ExitFailure, "%v", err)
	}
	s, err := parseSchema(argv[1])
	if err != nil {
		return fail(ExitSchema, "%v", err)
	}
	s.spell(cfg.dash)

	// 1. Scan argv into raw values.
	sc := scan(s, argv[2:])
	if sc.help {
		if _, err := io.WriteString(stderr, helpText(s, cfg.name)); err != nil {
			return ExitFailure
		}
		return ExitStop
	}
	if sc.version {
		return printVersion()
	}
	if err := cfg.loadOutput(); err != nil {
		return fail(ExitFailure, "%v", err)
	}

	errs := sc.errs
	var out strings.Builder
	for _, e := range s.entries {
		// 2. Resolve defaults and absent flags.
		items, ok := resolve(e, sc.values[e])
		if !ok {
			if e.required {
				errs = append(errs, "missing required "+kind(e)+" "+displayName(e))
			}
			continue
		}

		// 3. Validate every value.
		if e.typ == "list" {
			if lo, hi := e.itemLimits(); len(items) < lo {
				errs = append(errs, fmt.Sprintf("%s needs at least %d items, got %d", displayName(e), lo, len(items)))
			} else if len(items) > hi {
				errs = append(errs, fmt.Sprintf("%s allows at most %d items, got %d", displayName(e), hi, len(items)))
			}
		}
		if e.required && strings.Join(items, "") == "" { // every item empty
			errs = append(errs, displayName(e)+" requires a non-empty value")
			continue
		}
		bad := false
		for i, item := range items {
			v, err := validate(e, item)
			if err != nil {
				errs = append(errs, fmt.Sprintf("invalid value for %s: %v", displayName(e), err))
				bad = true
				continue
			}
			items[i] = v
		}
		if bad {
			continue
		}

		// 4. Emit, in schema order.
		out.WriteString(cfg.varName(e.long))
		out.WriteString(cfg.outDelim)
		out.WriteString(strings.Join(items, cfg.listDelim))
		out.WriteByte('\n')
	}

	if len(errs) > 0 {
		prefix := "error: "
		if cfg.name != "" {
			prefix = cfg.name + ": "
		}
		var b strings.Builder
		for _, e := range errs {
			b.WriteString(prefix + e + "\n")
		}
		fmt.Fprintf(&b, "Usage: %s\n", usageLine(s, cfg.name))
		if cfg.name != "" {
			fmt.Fprintf(&b, "Try '%s --help' for more information.\n", cfg.name)
		} else {
			b.WriteString("Try --help for more information.\n")
		}
		_, _ = io.WriteString(stderr, b.String())
		return ExitArgs
	}
	if _, err := io.WriteString(stdout, out.String()); err != nil {
		return fail(ExitFailure, "writing output: %v", err)
	}
	return ExitOK
}

// resolve returns the raw values for e: what was given, else its default,
// else false for a flag. ok is false when there is nothing to emit.
func resolve(e *entry, given []string) (items []string, ok bool) {
	switch {
	case len(given) > 0:
		return append([]string(nil), given...), true
	case e.def != nil && e.typ == "list":
		items, _ := splitItems(*e.def) // checked when the schema was parsed
		return items, true
	case e.def != nil:
		return []string{*e.def}, true
	case e.typ == "flag":
		return []string{"false"}, true
	}
	return nil, false
}

func kind(e *entry) string {
	if e.positional != 0 {
		return "argument"
	}
	return "option"
}

// config holds the GO_SHOPTS_ settings.
type config struct {
	prefix    string
	upcase    bool
	dash      bool
	outDelim  string
	listDelim string
	name      string
}

var prefixRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// loadConfig reads the settings help shows: the program name and how long
// names are typed. The rest is read by loadOutput, so a bad output setting
// does not block help.
func loadConfig() (config, error) {
	cfg := config{
		prefix: "SHOPTS_",
		upcase: true,
		dash:   true,
		name:   os.Getenv("GO_SHOPTS_NAME"),
	}
	return cfg, envBool("GO_SHOPTS_DASH", &cfg.dash)
}

// loadOutput reads the settings that shape the output.
func (cfg *config) loadOutput() error {
	var err error
	if cfg.outDelim, err = delimiter("GO_SHOPTS_OUT_DELIM", "\t"); err != nil {
		return err
	}
	if cfg.listDelim, err = delimiter("GO_SHOPTS_LIST_DELIM", ","); err != nil {
		return err
	}
	if p, ok := os.LookupEnv("GO_SHOPTS_PREFIX"); ok {
		cfg.prefix = p // an explicit empty prefix is allowed
	}
	if strings.HasPrefix(cfg.prefix, "GO_SHOPTS_") {
		return fmt.Errorf("GO_SHOPTS_PREFIX %q must not start with GO_SHOPTS_", cfg.prefix)
	}
	if cfg.prefix != "" && !prefixRE.MatchString(cfg.prefix) {
		return fmt.Errorf("GO_SHOPTS_PREFIX %q is not a valid shell variable prefix", cfg.prefix)
	}
	return envBool("GO_SHOPTS_UPCASE", &cfg.upcase)
}

// envBool reads a boolean setting into dst; unset or empty leaves dst as is.
func envBool(key string, dst *bool) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%s %q must be 1 or 0", key, v)
	}
	*dst = b
	return nil
}

// delimiters are the named delimiters, written {{ name }} like validators,
// for characters that are hard or impossible to put in an environment
// variable.
var delimiters = []struct{ Name, Char, Summary string }{
	{"tab", "\t", "Tab. The default output delimiter; works with `cut`, `awk -F'\\t'` and `read`"},
	{"null", "\x00", "NUL byte. No argument can contain it, so every value reads back exactly"},
}

// delimiter reads a delimiter setting: a {{ name }} from delimiters, or any
// string without a newline. Unset or empty gives fallback. Whether the
// delimiter suits the caller's tools is the caller's choice.
func delimiter(key, fallback string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	if m := templateRE.FindStringSubmatch(v); m != nil {
		var names []string
		for _, d := range delimiters {
			if d.Name == m[1] {
				return d.Char, nil
			}
			names = append(names, "{{ "+d.Name+" }}")
		}
		return "", fmt.Errorf("%s: unknown delimiter %q (named delimiters: %s)", key, v, strings.Join(names, ", "))
	}
	if strings.ContainsRune(v, '\n') {
		return "", fmt.Errorf("%s %q must not contain a newline", key, v)
	}
	return v, nil
}

func (c config) varName(long string) string {
	if c.upcase {
		long = strings.ToUpper(long)
	}
	return c.prefix + long
}
