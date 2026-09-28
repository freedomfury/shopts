package shopts

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var types = []string{"string", "int", "float", "bool", "enum", "flag", "list"}

// fieldSpec says how to read one schema field and which types it applies to.
type fieldSpec struct {
	types []string // nil: every type
	apply func(e *entry, v string) error
}

func (f fieldSpec) appliesTo(typ string) bool {
	return f.types == nil || slices.Contains(f.types, typ)
}

// fieldTable holds every option field. The type field is applied first, so
// the others can depend on it. define entries are handled by buildDefine.
var fieldTable = map[string]fieldSpec{
	"long":        {nil, setLong},
	"short":       {nil, setShort},
	"type":        {nil, setType},
	"required":    {nil, setRequired},
	"default":     {nil, setDefault},
	"help":        {nil, func(e *entry, v string) error { e.help = v; return nil }},
	"description": {nil, func(e *entry, v string) error { e.description = v; return nil }},
	"enum":        {[]string{"enum"}, setEnum},
	"pattern":     {[]string{"string", "list"}, func(e *entry, v string) error { e.pattern = v; return nil }},
	"failure":     {[]string{"string", "list"}, func(e *entry, v string) error { e.failure = v; return nil }},
	"minLength":   {[]string{"string"}, count(func(e *entry) **int { return &e.minLength }, 0)},
	"maxLength":   {[]string{"string"}, count(func(e *entry) **int { return &e.maxLength }, 1)},
	"min":         {[]string{"int", "float"}, limit(func(e *entry) **bound { return &e.min })},
	"max":         {[]string{"int", "float"}, limit(func(e *entry) **bound { return &e.max })},
	"minItems":    {[]string{"list"}, count(func(e *entry) **int { return &e.minItems }, 0)},
	"maxItems":    {[]string{"list"}, count(func(e *entry) **int { return &e.maxItems }, 0)},
	"positional":  {[]string{"string", "list"}, setPositional},
}

func setLong(e *entry, v string) error {
	if !isName(v) {
		return fmt.Errorf("%q must start with a letter and contain only letters, digits and _", v)
	}
	if v == "help" || v == "version" {
		return fmt.Errorf("%q is reserved for -H/--help and -V/--version", v)
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

func setDefault(e *entry, v string) error {
	if e.typ == "flag" {
		return errors.New("a flag is true when given and false when not, so it cannot have a default; " +
			"for on-by-default behavior, name the flag for turning it off, e.g. long=no_cache (typed --no-cache)")
	}
	e.def = &v
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

func setPositional(e *entry, v string) error {
	if v == "rest" {
		e.positional = restPositional
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || strconv.Itoa(n) != v {
		return fmt.Errorf("must be 1, 2, ... or rest, got %q", v)
	}
	e.positional = n
	return nil
}

// count reads a whole-number field of at least lo.
func count(field func(*entry) **int, lo int) func(*entry, string) error {
	return func(e *entry, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil || n < lo {
			return fmt.Errorf("must be a whole number of at least %d, got %q", lo, v)
		}
		*field(e) = &n
		return nil
	}
}

// bound is a min or max limit, read as the entry's type.
type bound struct {
	raw string
	i   int64
	f   float64
}

func (b *bound) greater(o *bound) bool { return b.i > o.i || b.f > o.f }

func limit(field func(*entry) **bound) func(*entry, string) error {
	return func(e *entry, v string) error {
		b := &bound{raw: v}
		var err error
		if e.typ == "int" {
			b.i, err = strconv.ParseInt(v, 10, 64)
		} else {
			b.f, err = parseFloat(v)
		}
		if err != nil {
			return fmt.Errorf("must be a valid %s, got %q", e.typ, v)
		}
		*field(e) = b
		return nil
	}
}

func isName(s string) bool {
	if s == "" {
		return false
	}
	if c := s[0]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
		return false
	}
	for _, r := range s {
		if !isKeyChar(r) {
			return false
		}
	}
	return true
}

// suggestField returns a "did you mean" hint for a misspelled field name.
func suggestField(key string) string {
	names := []string{"define"}
	for name := range fieldTable {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if d := editDistance(strings.ToLower(key), strings.ToLower(name)); d <= 2 && d < len(key)/2 {
			return fmt.Sprintf(" (did you mean %q?)", name)
		}
	}
	return ""
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
