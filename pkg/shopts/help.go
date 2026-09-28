package shopts

import (
	"fmt"
	"strings"
)

// helpText is the usage generated from the schema. It describes the
// caller's script only: its name and options.
func helpText(s *schema, name string) string {
	type row struct{ label, summary, description string }
	var rows []row
	for _, e := range s.entries {
		rows = append(rows, row{optLabel(e), summary(e), e.description})
	}
	rows = append(rows, row{"-H, --help", "Show this help", ""})

	width := 0
	for _, r := range rows {
		width = max(width, len(r.label))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s\n\nOptions:\n", usageLine(name))
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-*s   %s\n", width, r.label, r.summary)
		for _, line := range strings.Split(r.description, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				fmt.Fprintf(&b, "  %-*s   %s\n", width, "", line)
			}
		}
	}
	return b.String()
}

// usageLine is the one-line synopsis, e.g. "deploy.sh [OPTIONS]".
func usageLine(name string) string {
	if name == "" {
		return "[OPTIONS]"
	}
	return name + " [OPTIONS]"
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
