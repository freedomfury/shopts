package shopts

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

// pos is a location in the schema text, as the author wrote it.
type pos struct{ line, col int }

// errAt reports a mistake in the schema with where it is.
func errAt(p pos, format string, args ...any) error {
	return fmt.Errorf("schema line %d, col %d: %s", p.line, p.col, fmt.Sprintf(format, args...))
}

// rawField is one key=value pair as read by the lexer.
type rawField struct {
	key    string
	keyPos pos
	value  string
	valPos pos
}

// rawEntry is one ;-terminated entry as read by the lexer.
type rawEntry struct {
	pos    pos
	fields []rawField
}

func (r rawEntry) has(key string) bool {
	for _, f := range r.fields {
		if f.key == key {
			return true
		}
	}
	return false
}

// entry is one option.
type entry struct {
	pos         pos
	fieldPos    map[string]pos // where each field's value starts, for errors
	long        string
	name        string // long as typed on the command line; see spell
	short       string
	typ         string
	required    bool
	def         *string // default as written; nil when there is none
	help        string
	description string
	enum        []string
	pattern     string // as written
	failure     string
	match       *validator
	minLength   *int
	maxLength   *int
	min, max    *bound
	minItems    *int
	maxItems    *int
}

// at returns the position of field key's value, or the entry's position.
func (e *entry) at(key string) pos {
	if p, ok := e.fieldPos[key]; ok {
		return p
	}
	return e.pos
}

// schema is a parsed schema.
type schema struct {
	entries []*entry // options, in schema order
}

// ---------------------------------------------------------------------------
// Lexer
// ---------------------------------------------------------------------------

// lexer reads schema text into raw entries. Grammar:
//
//	schema := entry*
//	entry  := field ("," field)* ";"
//	field  := key "=" value
//	value  := quoted | bare     (bare runs to the next unquoted "," or ";")
type lexer struct {
	sc      scanner.Scanner
	removed []int // columns dedent removed from each line; added back to reported columns
	err     error
}

func newLexer(text string) *lexer {
	text = strings.ReplaceAll(text, "\r", "")
	l := &lexer{err: checkUTF8(text)}
	text, l.removed = dedent(text)
	l.sc.Init(strings.NewReader(text))
	l.sc.Error = func(_ *scanner.Scanner, msg string) {
		if l.err == nil {
			l.err = errAt(l.pos(), "%s", msg)
		}
	}
	return l
}

// pos returns the position of the next character, as the author wrote it.
func (l *lexer) pos() pos {
	l.sc.Peek()
	p := l.sc.Pos()
	col := p.Column
	if p.Line >= 1 && p.Line <= len(l.removed) {
		col += l.removed[p.Line-1]
	}
	return pos{line: p.Line, col: col}
}

// checkUTF8 reports the first byte that is not UTF-8, or a byte order mark.
// Schemas are plain UTF-8 text.
func checkUTF8(text string) error {
	if strings.HasPrefix(text, "\uFEFF") {
		return errAt(pos{1, 1}, "schema starts with a byte order mark (BOM); save it as UTF-8 without one")
	}
	for i, line := range strings.Split(text, "\n") {
		col := 1
		for len(line) > 0 {
			r, size := utf8.DecodeRuneInString(line)
			if r == utf8.RuneError && size == 1 {
				return errAt(pos{i + 1, col}, "schema is not valid UTF-8")
			}
			line = line[size:]
			col++
		}
	}
	return nil
}

func (l *lexer) skip(chars string) {
	for strings.ContainsRune(chars, l.sc.Peek()) {
		l.sc.Next()
	}
}

func (l *lexer) entries() ([]rawEntry, error) {
	if l.err != nil {
		return nil, l.err
	}
	var out []rawEntry
	for {
		l.skip(" \t\n")
		if l.sc.Peek() == scanner.EOF {
			return out, l.err
		}
		e, err := l.entry()
		if err != nil {
			return nil, err
		}
		if l.err != nil {
			return nil, l.err
		}
		out = append(out, e)
	}
}

func (l *lexer) entry() (rawEntry, error) {
	e := rawEntry{pos: l.pos()}
	for {
		f, err := l.field()
		if err != nil {
			return e, err
		}
		e.fields = append(e.fields, f)
		l.skip(" \t\n")
		switch ch := l.sc.Peek(); ch {
		case ',':
			l.sc.Next()
		case ';':
			l.sc.Next()
			return e, nil
		case scanner.EOF:
			return e, errAt(e.pos, "entry is missing its terminating ';'")
		default:
			return e, errAt(l.pos(), "expected ',' or ';' after the value of %q, found %q", f.key, ch)
		}
	}
}

func (l *lexer) field() (rawField, error) {
	l.skip(" \t\n")
	f := rawField{keyPos: l.pos()}
	var b strings.Builder
	for isKeyChar(l.sc.Peek()) {
		b.WriteRune(l.sc.Next())
	}
	f.key = b.String()
	if f.key == "" {
		return f, errAt(f.keyPos, "expected a field name, found %s", describe(l.sc.Peek()))
	}
	l.skip(" \t")
	if l.sc.Peek() != '=' {
		return f, errAt(l.pos(), "expected '=' after field name %q, found %s", f.key, describe(l.sc.Peek()))
	}
	l.sc.Next()
	l.skip(" \t")
	f.valPos = l.pos()
	if l.sc.Peek() == '\n' {
		// Only a quoted value may start on the next line.
		newline := f.valPos
		l.skip(" \t\n")
		f.valPos = l.pos()
		if ch := l.sc.Peek(); ch != '"' && ch != ',' && ch != ';' && ch != scanner.EOF {
			return f, errAt(newline, "value of %q is missing on this line; a bare value cannot start on the next line (quote it, or end the field with ',' or ';')", f.key)
		}
	}
	var err error
	f.value, err = l.value(f.key)
	return f, err
}

// value reads one value. This is the only place the quoting rule lives.
func (l *lexer) value(key string) (string, error) {
	if l.sc.Peek() == '"' {
		return l.quoted()
	}
	var b strings.Builder
	var newline, quote *pos // first newline; opening quote while inside quotes
	for {
		ch := l.sc.Peek()
		if ch == scanner.EOF && quote != nil {
			return "", errAt(*quote, "quote in the value of %q is never closed", key)
		}
		if quote == nil && (ch == ',' || ch == ';') || ch == scanner.EOF {
			break
		}
		if ch == '\n' && newline == nil {
			p := l.pos()
			newline = &p
		}
		if ch == '"' {
			if quote == nil {
				p := l.pos()
				quote = &p
			} else {
				quote = nil
			}
		}
		b.WriteRune(l.sc.Next())
		if ch == '\\' && quote != nil {
			if next := l.sc.Peek(); next == '"' || next == '\\' {
				b.WriteRune(l.sc.Next()) // kept as written; it does not end the quotes
			}
		}
	}
	v := strings.TrimSpace(b.String())
	if strings.ContainsRune(v, '\n') {
		return "", errAt(*newline, "unquoted value of %q runs onto the next line; end the field with ',' or ';', or quote the value", key)
	}
	return v, nil
}

// quoted reads a double-quoted value. Only \" and \\ are escapes; every
// other backslash is kept as written, so regexes need no double escaping.
func (l *lexer) quoted() (string, error) {
	open := l.pos()
	l.sc.Next()
	var b strings.Builder
	for {
		switch ch := l.sc.Next(); ch {
		case scanner.EOF:
			return "", errAt(open, "quoted value is never closed")
		case '"':
			return b.String(), nil
		case '\\':
			if next := l.sc.Peek(); next == '"' || next == '\\' {
				b.WriteRune(l.sc.Next())
				continue
			}
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}
}

func isKeyChar(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func describe(r rune) string {
	if r == scanner.EOF {
		return "end of schema"
	}
	return fmt.Sprintf("%q", r)
}

// dedent removes common leading indentation from all non-blank lines, and
// empties blank ones. It returns how many columns it removed from each line.
func dedent(s string) (string, []int) {
	lines := strings.Split(s, "\n")
	removed := make([]int, len(lines))
	indent := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if indent == -1 || n < indent {
			indent = n
		}
	}
	if indent <= 0 {
		return s, removed
	}
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			removed[i] = len(l)
			lines[i] = ""
		} else {
			removed[i] = indent
			lines[i] = l[indent:]
		}
	}
	return strings.Join(lines, "\n"), removed
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// parseSchema parses the schema text. dash is GO_SHOPTS_DASH: whether long
// names are typed with dashes; see spell.
func parseSchema(text string, dash bool) (*schema, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("schema is empty")
	}
	raws, err := newLexer(text).entries()
	if err != nil {
		return nil, err
	}

	// Defines first, so options can use a name defined further down.
	defines := map[string]*validator{}
	for _, r := range raws {
		if !r.has("define") {
			continue
		}
		v, err := buildDefine(r)
		if err != nil {
			return nil, err
		}
		if _, dup := defines[v.Name]; dup {
			return nil, errAt(r.pos, "pattern %q is defined twice", v.Name)
		}
		defines[v.Name] = v
	}

	s := &schema{}
	for _, r := range raws {
		if r.has("define") {
			continue
		}
		e, err := buildEntry(r, defines)
		if err != nil {
			return nil, err
		}
		s.entries = append(s.entries, e)
	}
	if err := s.check(); err != nil {
		return nil, err
	}
	s.spell(dash)
	return s, nil
}

// spell sets how each long name is typed on the command line. With dash
// (GO_SHOPTS_DASH, the default) underscores become dashes: long=dry_run is
// --dry-run. Without it names are typed as written.
func (s *schema) spell(dash bool) {
	for _, e := range s.entries {
		e.name = e.long
		if dash {
			e.name = strings.ReplaceAll(e.long, "_", "-")
		}
	}
}

func buildDefine(r rawEntry) (*validator, error) {
	var name, pattern, failure string
	var namePos, patternPos pos
	seen := map[string]bool{}
	for _, f := range r.fields {
		if seen[f.key] {
			return nil, errAt(f.keyPos, "field %q is given twice", f.key)
		}
		seen[f.key] = true
		switch f.key {
		case "define":
			name, namePos = f.value, f.valPos
		case "pattern":
			pattern, patternPos = f.value, f.valPos
		case "failure":
			if err := oneLine(f.value); err != nil {
				return nil, errAt(f.valPos, "failure: %v", err)
			}
			failure = f.value
		default:
			return nil, errAt(f.keyPos, "field %q is not allowed in a define entry (only define, pattern, failure)", f.key)
		}
	}
	if !validatorNameRE.MatchString(name) {
		return nil, errAt(namePos, "define: name %q must start with a letter and contain only letters and digits", name)
	}
	if _, ok := builtinByName[name]; ok {
		return nil, errAt(namePos, "define: %q is a built-in validator and cannot be redefined", name)
	}
	if pattern == "" {
		return nil, errAt(r.pos, "define %q has no pattern", name)
	}
	if templateRE.MatchString(pattern) {
		return nil, errAt(patternPos, "define %q: pattern must be a regex, not a {{ }} reference", name)
	}
	v, err := regexValidator(pattern)
	if err != nil {
		return nil, errAt(patternPos, "define %q: %v", name, err)
	}
	v.Name = name
	v.Failure = failure
	return v, nil
}

func buildEntry(r rawEntry, defines map[string]*validator) (*entry, error) {
	e := &entry{pos: r.pos, fieldPos: map[string]pos{}}

	// Type first: it decides which other fields apply.
	fields := make([]rawField, 0, len(r.fields))
	for _, f := range r.fields {
		if _, ok := fieldTable[f.key]; !ok {
			return nil, errAt(f.keyPos, "unknown field %q", f.key)
		}
		if _, dup := e.fieldPos[f.key]; dup {
			return nil, errAt(f.keyPos, "field %q is given twice", f.key)
		}
		e.fieldPos[f.key] = f.valPos
		if f.key == "type" {
			fields = append([]rawField{f}, fields...)
		} else {
			fields = append(fields, f)
		}
	}
	if _, ok := e.fieldPos["long"]; !ok {
		return nil, errAt(r.pos, "entry has no 'long' field")
	}
	if _, ok := e.fieldPos["type"]; !ok {
		return nil, errAt(r.pos, "entry has no 'type' field")
	}

	for _, f := range fields {
		spec := fieldTable[f.key]
		if !spec.appliesTo(e.typ) {
			if spec.why != "" {
				return nil, errAt(f.keyPos, "%s: %s", f.key, spec.why)
			}
			return nil, errAt(f.keyPos, "field %q does not apply to type %s (only %s)",
				f.key, e.typ, strings.Join(spec.types, ", "))
		}
		if err := spec.apply(e, f.value); err != nil {
			return nil, errAt(f.valPos, "%s: %v", f.key, err)
		}
	}

	if e.pattern != "" {
		v, err := resolvePattern(e.pattern, defines)
		if err != nil {
			return nil, errAt(e.at("pattern"), "pattern: %v", err)
		}
		e.match = v
	}
	if err := e.check(); err != nil {
		return nil, err
	}
	return e, nil
}

// check runs the rules that span fields, once the entry is complete.
func (e *entry) check() error {
	fail := func(key, format string, args ...any) error {
		return errAt(e.at(key), "option %q: %s", e.long, fmt.Sprintf(format, args...))
	}
	if e.required && e.typ == "flag" {
		return fail("required", "a flag is true when given and false when not, so it cannot be required")
	}
	if e.required && e.def != nil {
		return fail("default", "required and default cannot both be set")
	}
	if e.typ == "enum" && len(e.enum) == 0 {
		return fail("type", "type enum needs an enum field listing the allowed values")
	}
	if e.minLength != nil && e.maxLength != nil && *e.minLength > *e.maxLength {
		return fail("minLength", "minLength %d is greater than maxLength %d", *e.minLength, *e.maxLength)
	}
	if e.min != nil && e.max != nil && e.min.greater(e.max) {
		return fail("min", "min %s is greater than max %s", e.min.raw, e.max.raw)
	}
	if lo, hi := e.itemLimits(); e.typ == "list" && lo > hi {
		minNote, maxNote := "", ""
		if e.minItems == nil {
			minNote = " (required lists need at least 1 item)"
		}
		if e.maxItems == nil {
			maxNote = " (the default)"
		}
		key := "maxItems"
		if e.minItems != nil {
			key = "minItems"
		}
		return fail(key, "minItems %d%s is greater than maxItems %d%s", lo, minNote, hi, maxNote)
	}
	if e.failure != "" && e.pattern == "" {
		return fail("failure", "failure is set but there is no pattern")
	}
	if e.def != nil {
		if e.typ == "list" {
			items, err := splitItems(*e.def)
			if err != nil {
				return fail("default", "%v", err)
			}
			if err := e.checkItems(items); err != nil {
				return fail("default", "default %v", err)
			}
			for _, item := range items {
				if _, err := validate(e, item); err != nil {
					return fail("default", "default item %q: %v", item, err)
				}
			}
		} else if _, err := validate(e, *e.def); err != nil {
			return fail("default", "default %q: %v", *e.def, err)
		}
	}
	return nil
}

// check runs the rules that span entries.
func (s *schema) check() error {
	if len(s.entries) == 0 {
		return errors.New("schema has no options")
	}
	longs := map[string]string{} // lowercased long name -> long name as written
	shorts := map[string]bool{}
	for _, e := range s.entries {
		// Output names are uppercased by default, so names that differ only
		// in case would emit the same variable.
		if other, dup := longs[strings.ToLower(e.long)]; dup {
			if other == e.long {
				return errAt(e.at("long"), "long name %q is used twice", e.long)
			}
			return errAt(e.at("long"), "long names %q and %q differ only in case and would emit the same variable", other, e.long)
		}
		longs[strings.ToLower(e.long)] = e.long
		if e.short != "" {
			if shorts[e.short] {
				return errAt(e.at("short"), "short flag %q is used twice", e.short)
			}
			shorts[e.short] = true
		}
	}
	return nil
}

// checkItems applies the rules on a whole value rather than on each item:
// a list's item count, and that a required option is not empty.
func (e *entry) checkItems(items []string) error {
	if e.typ == "list" {
		if lo, hi := e.itemLimits(); len(items) < lo {
			return fmt.Errorf("needs at least %d items, got %d", lo, len(items))
		} else if len(items) > hi {
			return fmt.Errorf("allows at most %d items, got %d", hi, len(items))
		}
	}
	if e.required && strings.Join(items, "") == "" {
		return errors.New("requires a non-empty value")
	}
	return nil
}

// itemLimits returns the item count a list accepts: minItems (1 when
// required and unset) to maxItems (default 100).
func (e *entry) itemLimits() (int, int) {
	lo, hi := 0, defaultMaxItems
	if e.minItems != nil {
		lo = *e.minItems
	} else if e.required {
		lo = 1
	}
	if e.maxItems != nil {
		hi = *e.maxItems
	}
	return lo, hi
}

const defaultMaxItems = 100

// splitItems splits a comma-separated value (an enum or a list default) into
// items. A bare item runs to the next ',' and is trimmed; an item may be
// double-quoted, with the same rule as schema values, to hold a comma.
func splitItems(s string) ([]string, error) {
	l := &lexer{}
	l.sc.Init(strings.NewReader(s))
	l.sc.Error = func(*scanner.Scanner, string) {}
	var out []string
	for {
		l.skip(" \t\n")
		var item string
		if l.sc.Peek() == '"' {
			v, err := l.quoted()
			if err != nil {
				return nil, errors.New("quoted item is never closed")
			}
			item = v
			l.skip(" \t\n")
			if ch := l.sc.Peek(); ch != ',' && ch != scanner.EOF {
				return nil, fmt.Errorf("expected ',' after quoted item %q, found %q", item, ch)
			}
		} else {
			var b strings.Builder
			for ch := l.sc.Peek(); ch != ',' && ch != scanner.EOF; ch = l.sc.Peek() {
				b.WriteRune(l.sc.Next())
			}
			item = strings.TrimSpace(b.String())
		}
		out = append(out, item)
		if l.sc.Next() == scanner.EOF {
			return out, nil
		}
	}
}
