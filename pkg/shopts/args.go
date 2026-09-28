package shopts

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// scanned is argv sorted into raw values, before defaults and validation.
type scanned struct {
	help, version bool
	values        map[*entry][]string // every occurrence given, by entry
	failed        map[*entry]bool     // options given in a broken form, already reported
	errs          []string
}

// scan reads argv against the schema. It checks only the shape of the
// arguments (known options, values present, no repeats); values are
// validated later.
func scan(s *schema, args []string) scanned {
	sc := scanned{values: map[*entry][]string{}, failed: map[*entry]bool{}}

	byLong := map[string]*entry{}
	byShort := map[string]*entry{}
	for _, e := range s.entries {
		byLong[e.name] = e
		if e.short != "" {
			byShort[e.short] = e
		}
	}

	// -H/--help and -V/--version win from any position before "--". A "--"
	// right after an option that takes a value is that value, not the end.
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "--":
			i = len(args)
		case "-H", "--help":
			sc.help = true
			return sc
		case "-V", "--version":
			sc.version = true
			return sc
		default:
			e := byShort[strings.TrimPrefix(arg, "-")]
			if strings.HasPrefix(arg, "--") {
				e = byLong[arg[2:]]
			}
			if e != nil && e.typ != "flag" && i+1 < len(args) && args[i+1] == "--" {
				i++
			}
		}
	}

	// Every argument must be an option or an option's value. Bare words, and
	// anything after "--", are rejected: accepting them would silently hide
	// mistakes such as a forgotten dash or unquoted spaces.
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			for _, rest := range args[i+1:] {
				sc.errs = append(sc.errs, fmt.Sprintf("unrecognized bare word %q after --", rest))
			}
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			sc.errs = append(sc.errs, fmt.Sprintf("unrecognized bare word %q", arg))
			continue
		}

		name, value, inline := strings.Cut(arg, "=")
		var e *entry
		if !isASCII(name) {
			// Option names are ASCII; anything else cannot match one.
		} else if strings.HasPrefix(name, "--") {
			e = byLong[name[2:]]
		} else if len(name) > 2 {
			sc.errs = append(sc.errs, fmt.Sprintf("short options cannot be combined: %s", name))
			continue
		} else {
			e = byShort[name[1:]]
		}
		if e == nil {
			sc.errs = append(sc.errs, fmt.Sprintf("unknown option %s", name))
			continue
		}

		switch {
		case e.typ == "flag" && inline:
			sc.errs = append(sc.errs, fmt.Sprintf("%s does not take a value", name))
			sc.failed[e] = true
			continue
		case e.typ == "flag":
			value = "true"
		case !inline && i+1 < len(args):
			i++
			value = args[i]
		case !inline:
			sc.errs = append(sc.errs, fmt.Sprintf("%s requires a value", name))
			sc.failed[e] = true
			continue
		}

		if e.typ != "list" && len(sc.values[e]) > 0 {
			sc.errs = append(sc.errs, fmt.Sprintf("%s is given more than once", displayName(e)))
			continue
		}
		sc.values[e] = append(sc.values[e], value)
	}

	return sc
}

// displayName is how errors refer to an option.
func displayName(e *entry) string {
	return "--" + e.name
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
