package shopts

import (
	"fmt"
	"strings"
)

// helpText is the usage generated from the schema. It describes the
// caller's script only: its name, arguments and options.
func helpText(s *schema, name string) string {
	type row struct {
		label string
		e     *entry
	}
	var args, opts []row
	for _, e := range s.positionals {
		args = append(args, row{argLabel(e), e})
	}
	if s.rest != nil {
		args = append(args, row{argLabel(s.rest), s.rest})
	}
	for _, e := range s.entries {
		if e.positional == 0 {
			opts = append(opts, row{optLabel(e), e})
		}
	}
	helpRow := row{"-H, --help", nil}
	opts = append(opts, helpRow)

	width := 0
	for _, r := range append(append([]row{}, args...), opts...) {
		width = max(width, len(r.label))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s\n", usageLine(s, name))
	section := func(title string, rows []row) {
		if len(rows) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n%s:\n", title)
		for _, r := range rows {
			if r.e == nil {
				fmt.Fprintf(&b, "  %-*s   %s\n", width, r.label, "Show this help")
				continue
			}
			fmt.Fprintf(&b, "  %-*s   %s\n", width, r.label, summary(r.e))
			for _, line := range strings.Split(r.e.description, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					fmt.Fprintf(&b, "  %-*s   %s\n", width, "", line)
				}
			}
		}
	}
	section("Arguments", args)
	section("Options", opts)
	return b.String()
}

// usageLine is the one-line synopsis, e.g. "deploy.sh [OPTIONS] <target> [files...]".
func usageLine(s *schema, name string) string {
	var parts []string
	if name != "" {
		parts = append(parts, name)
	}
	parts = append(parts, "[OPTIONS]")
	for _, e := range s.positionals {
		parts = append(parts, argLabel(e))
	}
	if s.rest != nil {
		parts = append(parts, argLabel(s.rest))
	}
	return strings.Join(parts, " ")
}

func argLabel(e *entry) string {
	name := e.name
	if e.positional == restPositional {
		name += "..."
	}
	if e.required {
		return "<" + name + ">"
	}
	return "[" + name + "]"
}

func optLabel(e *entry) string {
	long := "--" + e.name
	label := "    " + long
	if e.short != "" {
		label = "-" + e.short + ", " + long
	}
	if e.typ != "flag" {
		label += " <value>"
	}
	return label
}

// summary is the one-line description of an entry: its help text, then
// its type and constraints.
func summary(e *entry) string {
	var parts []string
	if e.help != "" {
		parts = append(parts, e.help)
	}
	parts = append(parts, e.typ)
	if e.required {
		parts = append(parts, "required")
	}
	if e.def != nil {
		parts = append(parts, "default: "+*e.def)
	}
	if len(e.enum) > 0 {
		parts = append(parts, "allowed: "+strings.Join(e.enum, ", "))
	}
	if e.min != nil {
		parts = append(parts, "min: "+e.min.raw)
	}
	if e.max != nil {
		parts = append(parts, "max: "+e.max.raw)
	}
	if e.minLength != nil {
		parts = append(parts, fmt.Sprintf("minimum length: %d", *e.minLength))
	}
	if e.maxLength != nil {
		parts = append(parts, fmt.Sprintf("maximum length: %d", *e.maxLength))
	}
	if e.minItems != nil {
		parts = append(parts, fmt.Sprintf("minimum items: %d", *e.minItems))
	}
	if e.maxItems != nil {
		parts = append(parts, fmt.Sprintf("maximum items: %d", *e.maxItems))
	}
	if e.match != nil {
		switch {
		case e.failure != "":
			parts = append(parts, "format: "+e.failure)
		case e.match.Name != "":
			parts = append(parts, "format: "+e.match.Name)
		default:
			parts = append(parts, "must match: "+e.pattern)
		}
	}
	return strings.Join(parts, "; ")
}
