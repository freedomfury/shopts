package shopts

import (
	"fmt"
	"strings"
)

// scanned is argv sorted into raw values, before defaults and validation.
type scanned struct {
	help, version bool
	values        map[*entry][]string // every occurrence given, by entry
	errs          []string
}

// scan reads argv against the schema. It checks only the shape of the
// arguments (known options, values present, no repeats); values are
// validated later.
func scan(s *schema, args []string) scanned {
	sc := scanned{values: map[*entry][]string{}}

	byLong := map[string]*entry{}
	byShort := map[string]*entry{}
	for _, e := range s.entries {
		if e.positional != 0 {
			continue
		}
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

	var bare []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			bare = append(bare, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			bare = append(bare, arg)
			continue
		}

		name, value, inline := strings.Cut(arg, "=")
		var e *entry
		if strings.HasPrefix(name, "--") {
			e = byLong[name[2:]]
			if e == nil {
				// Suggest the other spelling: --dry_run for --dry-run and back.
				alt := strings.NewReplacer("_", "-", "-", "_").Replace(name[2:])
				if byLong[alt] != nil {
					sc.errs = append(sc.errs, fmt.Sprintf("unknown option %s (did you mean --%s?)", name, alt))
					continue
				}
			}
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
			continue
		case e.typ == "flag":
			value = "true"
		case !inline && i+1 < len(args):
			i++
			value = args[i]
		case !inline:
			sc.errs = append(sc.errs, fmt.Sprintf("%s requires a value", name))
			continue
		}

		if e.typ != "list" && len(sc.values[e]) > 0 {
			sc.errs = append(sc.errs, fmt.Sprintf("%s is given more than once", displayName(e)))
			continue
		}
		sc.values[e] = append(sc.values[e], value)
	}

	for i, arg := range bare {
		switch {
		case i < len(s.positionals):
			sc.values[s.positionals[i]] = []string{arg}
		case s.rest != nil:
			sc.values[s.rest] = append(sc.values[s.rest], arg)
		default:
			sc.errs = append(sc.errs, fmt.Sprintf("unexpected argument %q", arg))
		}
	}
	return sc
}

// displayName is how errors and help refer to an entry.
func displayName(e *entry) string {
	switch e.positional {
	case 0:
		return "--" + e.name
	case restPositional:
		return "<" + e.name + "...>"
	default:
		return "<" + e.name + ">"
	}
}
