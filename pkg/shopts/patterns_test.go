package shopts

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the README validator table from the registry")

// TestBuiltinExamples runs every registry entry's Valid and Invalid examples.
func TestBuiltinExamples(t *testing.T) {
	for _, v := range builtins {
		t.Run(v.Name, func(t *testing.T) {
			if v.Summary == "" || v.Failure == "" || len(v.Valid) == 0 || len(v.Invalid) == 0 {
				t.Fatal("registry entry needs Summary, Failure, Valid and Invalid")
			}
			for _, s := range v.Valid {
				if !v.Check(s) {
					t.Errorf("should accept %q", s)
				}
			}
			for _, s := range v.Invalid {
				if v.Check(s) {
					t.Errorf("should reject %q", s)
				}
			}
		})
	}
}

func TestTemplateSpacing(t *testing.T) {
	for _, p := range []string{"{{SemVer}}", "{{ SemVer }}", "{{SemVer }}", "{{ SemVer}}"} {
		v, err := resolvePattern(p, nil)
		if err != nil || v.Name != "SemVer" {
			t.Errorf("%q: got %v, %v", p, v, err)
		}
	}
}

func validatorTable() string {
	var b strings.Builder
	b.WriteString("| Template | Accepts | Example |\n|---|---|---|\n")
	for _, v := range builtins {
		fmt.Fprintf(&b, "| `%s` | %s | `%s` |\n", v.Name, v.Summary, v.Valid[0])
	}
	return b.String()
}

func delimiterTable() string {
	var b strings.Builder
	b.WriteString("| Name | Character |\n|---|---|\n")
	for _, d := range delimiters {
		fmt.Fprintf(&b, "| `{{ %s }}` | %s |\n", d.Name, d.Summary)
	}
	return b.String()
}

// helpExample is the README's help sample, generated from helpText so it
// cannot drift from the real output.
func helpExample() string {
	s, err := parseSchema(`
		short=e, long=env, type=enum, enum="dev,prod", required=true, help=Target;
		short=v, long=verbose, type=flag, help=Verbose output;
	`)
	if err != nil {
		panic(err)
	}
	return "```\n" + helpText(s, "deploy.sh") + "```\n"
}

// TestREADMETables keeps the generated README tables in step with the code.
// Regenerate them with: make readme
func TestREADMETables(t *testing.T) {
	const path = "../../README.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	for _, table := range []struct{ name, source, want string }{
		{"validators", "pkg/shopts/patterns.go", validatorTable()},
		{"delimiters", "pkg/shopts/run.go", delimiterTable()},
		{"help-example", "pkg/shopts/help.go", helpExample()},
	} {
		begin := "<!-- " + table.name + ":begin (generated from " + table.source + "; run make readme) -->"
		end := "<!-- " + table.name + ":end -->"
		start, stop := strings.Index(readme, begin), strings.Index(readme, end)
		if start < 0 || stop < start {
			t.Fatalf("README is missing the %q ... %q markers", begin, end)
		}
		start += len(begin) + 1
		if readme[start:stop] == table.want {
			continue
		}
		if !*update {
			t.Fatalf("README %s table is out of date; run: make readme", table.name)
		}
		readme = readme[:start] + table.want + readme[stop:]
	}
	if *update {
		if err := os.WriteFile(path, []byte(readme), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
