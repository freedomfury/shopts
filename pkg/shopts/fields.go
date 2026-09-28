package shopts

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var types = []string{"string", "int", "float", "bool", "enum", "flag", "list"}

// fieldSpec says how to read one schema field and which types it applies to.
type fieldSpec struct {
	types []string // nil: every type
	apply func(e *entry, v string) error
	why   string // why the field does not apply to the other types, if not obvious
}

func (f fieldSpec) appliesTo(typ string) bool {
	return f.types == nil || slices.Contains(f.types, typ)
}

// fieldTable holds every option field. The type field is applied first, so
// the others can depend on it. define entries are handled by buildDefine.
var fieldTable = map[string]fieldSpec{
	"long":     {apply: setLong},
	"short":    {apply: setShort},
	"type":     {apply: setType},
	"required": {apply: setRequired},
	"default": {
		types: []string{"string", "int", "float", "bool", "enum", "list"},
		apply: func(e *entry, v string) error { e.def = &v; return nil },
		why: "a flag is true when given and false when not, so it cannot have a default; " +
			"for on-by-default behavior, name the flag for turning it off, e.g. long=no_cache (typed --no-cache)",
	},
	"help":        {apply: setHelp},
	"description": {apply: func(e *entry, v string) error { e.description = v; return nil }},
	"enum":        {types: []string{"enum"}, apply: setEnum},
	"pattern":     {types: []string{"string", "list"}, apply: func(e *entry, v string) error { e.pattern = v; return nil }},
	"failure":     {types: []string{"string", "list"}, apply: setFailure},
	"minLength":   {types: []string{"string"}, apply: func(e *entry, v string) (err error) { e.minLength, err = atLeast(v, 0); return }},
	"maxLength":   {types: []string{"string"}, apply: func(e *entry, v string) (err error) { e.maxLength, err = atLeast(v, 1); return }},
	"min":         {types: []string{"int", "float"}, apply: func(e *entry, v string) (err error) { e.min, err = parseBound(e.typ, v); return }},
	"max":         {types: []string{"int", "float"}, apply: func(e *entry, v string) (err error) { e.max, err = parseBound(e.typ, v); return }},
	"minItems":    {types: []string{"list"}, apply: func(e *entry, v string) (err error) { e.minItems, err = atLeast(v, 0); return }},
	"maxItems":    {types: []string{"list"}, apply: func(e *entry, v string) (err error) { e.maxItems, err = atLeast(v, 0); return }},
}

func setLong(e *entry, v string) error {
	if !isName(v) {
		return fmt.Errorf("%q must start with a letter and contain only letters, digits and _", v)
	}
	if v == "help" || v == "version" {
		return fmt.Errorf("%q is reserved for -H/--help and -V/--version", v)
	}
	// GO_SHOPTS_ is shopts's own namespace; with an empty prefix, such a
	// name would emit a variable in it.
	if strings.HasPrefix(strings.ToLower(v), "go_shopts") {
		return fmt.Errorf("%q is reserved: names starting with go_shopts belong to shopts", v)
	}
	e.long = v
	return nil
}

func setShort(e *entry, v string) error {
	if len(v) != 1 || !isKeyChar(rune(v[0])) || v == "_" {
		return fmt.Errorf("%q must be a single letter or digit", v)
	}
	if v == "H" || v == "V" {
		return fmt.Errorf("%q is reserved for -H/--help and -V/--version", v)
	}
	e.short = v
	return nil
}

func setType(e *entry, v string) error {
	if !slices.Contains(types, v) {
		return fmt.Errorf("%q is not a type (want one of %s)", v, strings.Join(types, ", "))
	}
	e.typ = v
	return nil
}

func setRequired(e *entry, v string) error {
	switch v {
	case "true":
		e.required = true
	case "false":
		e.required = false
	default:
		return fmt.Errorf("must be true or false, got %q", v)
	}
	return nil
}

func setEnum(e *entry, v string) error {
	items, err := splitItems(v)
	if err != nil {
		return err
	}
	for i, item := range items {
		if item == "" {
			return errors.New("enum values must not be empty")
		}
		if slices.Contains(items[:i], item) {
			return fmt.Errorf("enum value %q is listed twice", item)
		}
	}
	e.enum = items
	return nil
}

func setHelp(e *entry, v string) error {
	e.help = v
	return oneLine(v)
}

func setFailure(e *entry, v string) error {
	e.failure = v
	return oneLine(v)
}

// oneLine rejects a newline in help= and failure=, which are shown on one
// line; description= is the field for more lines.
func oneLine(v string) error {
	if strings.ContainsRune(v, '\n') {
		return errors.New("must be a single line (use description for more lines)")
	}
	return nil
}

// atLeast reads a whole number of at least lo, written as plain digits
// (no sign, no leading zeros).
func atLeast(v string, lo int) (*int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || n < lo || strconv.Itoa(n) != v {
		return nil, fmt.Errorf("must be a whole number of at least %d, got %q", lo, v)
	}
	return &n, nil
}

// bound is a min or max limit, read as the entry's type.
type bound struct {
	raw string
	i   int64
	f   float64
}

func (b *bound) greater(o *bound) bool { return b.i > o.i || b.f > o.f }

// parseBound reads a min or max limit as a value of type typ.
func parseBound(typ, v string) (*bound, error) {
	b := &bound{raw: v}
	var err error
	if typ == "int" {
		b.i, err = strconv.ParseInt(v, 10, 64)
	} else {
		b.f, err = parseFloat(v)
	}
	if err != nil {
		return nil, fmt.Errorf("must be a valid %s, got %q", typ, v)
	}
	return b, nil
}

// isIdent reports whether s is a shell identifier: letters, digits and _,
// not starting with a digit.
func isIdent(s string) bool {
	if s == "" || (s[0] >= '0' && s[0] <= '9') {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool { return !isKeyChar(r) }) < 0
}

// isName reports whether s is a valid long name: an identifier that starts
// with a letter.
func isName(s string) bool {
	return isIdent(s) && s[0] != '_'
}
