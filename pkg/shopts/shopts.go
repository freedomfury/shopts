package shopts

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const DefaultPrefix = "GO_SHOPTS_"

var bashVarRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func humanType(value string) string {
	switch value {
	case "string":
		return "string"
	case "enum":
		return "enum"
	case "flag":
		return "flag (boolean switch)"
	case "int":
		return "int"
	case "float":
		return "float"
	case "bool":
		return "bool (explicit true/false)"
	case "list":
		return "list (repeatable, joined by delimiter)"
	default:
		return value
	}
}

func Run(argv []string, w io.Writer) error {
	if len(argv) < 2 {
		return errors.New("usage: shopts SCHEMA [ARGS...]")
	}

	currentPrefix := DefaultPrefix
	if p, ok := os.LookupEnv("GO_SHOPTS_PREFIX"); ok {
		currentPrefix = p
	}
	if currentPrefix != "" && !bashVarRE.MatchString(currentPrefix) {
		return fmt.Errorf("GO_SHOPTS_PREFIX %q is not a valid shell variable prefix", currentPrefix)
	}

	useUpcase := getenvBool("GO_SHOPTS_UPCASE")
	schemaText := argv[1]
	args := argv[2:]

	schema, err := parseSchema(schemaText)
	if err != nil {
		return err
	}

	if wantsHelp(args) {
		return printUsage(w, schema)
	}

	values, err := parseArgs(args, schema)
	if err != nil {
		return err
	}

	validationErrors := validateParsedValues(schema, values)
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	for _, entry := range schema {
		val, emit := resolvedValue(entry, values)
		if !emit {
			continue
		}
		outName, err := shVarName(entry.Long, currentPrefix, useUpcase)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\x00%s\n", outName, val); err != nil {
			return err
		}
	}

	return nil
}

// schemaEntry holds one parsed option definition.
type schemaEntry struct {
	Short           string
	Long            string
	Required        bool
	Type            string
	Help            string
	Description     string
	Enum            []string
	Default         string
	MinLength       *int
	MaxLength       *int
	Pattern         string
	Failure         string
	MinItems        *int
	MaxItems        *int
	CompiledPattern *regexp.Regexp
}

// parseSchema parses the key=value; schema format.
// Each option occupies one non-empty line. Every line must end with a semicolon.
//
// Values may be provided either unquoted (e.g. long=user;type=string;) or
// double-quoted as Go string literals when they contain delimiters or special
// characters (e.g. help="Contains; semicolons and = equals"). Quoting is
// optional and the parser will unquote values that start with a '"'. Use
// quoting for fields like `help` or `enum` when their contents may include
// commas, semicolons, or equals characters.
//
// Example unquoted line:
//
//	long=user;short=u;required=true;type=string;minLength=3;help=Username;
//
// Example quoted line:
//
//	long=mode;short=m;type=enum;enum="dev,prod,test";help="Mode; selects env";
func parseSchema(schemaText string) ([]schemaEntry, error) {
	// Allow callers to pass indented heredocs / multiline strings. Remove
	// common leading indentation so schema lines don't need manual trimming.
	schemaText = dedent(schemaText)

	if strings.TrimSpace(schemaText) == "" {
		return nil, errors.New("schema cannot be empty")
	}

	var entries []schemaEntry
	lineNum := 0
	for _, line := range strings.Split(schemaText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineNum++

		if !strings.HasSuffix(line, ";") {
			return nil, fmt.Errorf("schema line %d: line must end with a semicolon: %q", lineNum, line)
		}

		entry := schemaEntry{}
		fields, err := splitFields(line)
		if err != nil {
			return nil, fmt.Errorf("schema line %d: %w", lineNum, err)
		}
		for _, field := range fields {
			idx := strings.IndexByte(field, '=')
			if idx < 0 {
				return nil, fmt.Errorf("schema line %d: field %q missing '='", lineNum, field)
			}
			key := strings.TrimSpace(field[:idx])
			rawVal := strings.TrimSpace(field[idx+1:])
			// Accept both quoted and unquoted values. If quoted, unquote; else use as-is.
			var val string
			if strings.HasPrefix(rawVal, "\"") {
				valUnq, err := strconv.Unquote(rawVal)
				if err != nil {
					return nil, fmt.Errorf("schema line %d: invalid quoted value for %q: %w", lineNum, key, err)
				}
				val = strings.TrimSpace(valUnq)
			} else {
				val = strings.TrimSpace(rawVal)
			}
			switch key {
			case "short":
				entry.Short = val
			case "long":
				entry.Long = val
			case "required":
				entry.Required = val == "true" || val == "1" || val == "yes"
			case "type":
				entry.Type = val
			case "help":
				entry.Help = val
			case "description":
				entry.Description = val
			case "enum":
				if val != "" {
					// parse enum items from the quoted value; allow escaped commas (\,)
					entry.Enum = splitEnum(val)
				}
			case "default":
				entry.Default = val
			case "minLength":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("schema line %d: minLength must be an integer, got %q", lineNum, val)
				}
				entry.MinLength = &n
			case "maxLength":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("schema line %d: maxLength must be an integer, got %q", lineNum, val)
				}
				entry.MaxLength = &n
			case "pattern":
				entry.Pattern = val
			case "failure":
				entry.Failure = val
			case "minItems":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("schema line %d: minItems must be an integer, got %q", lineNum, val)
				}
				entry.MinItems = &n
			case "maxItems":
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("schema line %d: maxItems must be an integer, got %q", lineNum, val)
				}
				entry.MaxItems = &n
			default:
				return nil, fmt.Errorf("schema line %d: unknown field %q", lineNum, key)
			}
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, errors.New("schema must contain at least one option")
	}

	seenShort := map[string]struct{}{}
	seenLong := map[string]struct{}{}
	for i := range entries {
		e := &entries[i]
		if e.Long == "" {
			return nil, fmt.Errorf("schema entry %d: missing required field 'long'", i+1)
		}
		if !isValidName(e.Long) {
			return nil, fmt.Errorf("option %q: long name contains invalid characters", e.Long)
		}
		if e.Type == "" {
			return nil, fmt.Errorf("option %q: missing required field 'type'", e.Long)
		}
		switch e.Type {
		case "string", "enum", "flag", "int", "float", "bool", "list":
		default:
			return nil, fmt.Errorf("option %q: invalid type %q", e.Long, e.Type)
		}
		if e.Short != "" {
			if len(e.Short) != 1 || !isAlphanumeric(e.Short[0]) {
				return nil, fmt.Errorf("option %q: short flag %q must be a single alphanumeric character", e.Long, e.Short)
			}
			if _, dup := seenShort[e.Short]; dup {
				return nil, fmt.Errorf("duplicate short flag %q", e.Short)
			}
			seenShort[e.Short] = struct{}{}
		}
		if _, dup := seenLong[e.Long]; dup {
			return nil, fmt.Errorf("duplicate long name %q", e.Long)
		}
		seenLong[e.Long] = struct{}{}
		if e.Required && e.Default != "" {
			return nil, fmt.Errorf("option %q: cannot be both required and have a default", e.Long)
		}
		if e.Type == "flag" {
			if e.Default != "" && e.Default != "true" && e.Default != "false" {
				return nil, fmt.Errorf("option %q: flag default must be true or false", e.Long)
			}
			if len(e.Enum) > 0 || e.Pattern != "" || e.MinLength != nil || e.MaxLength != nil {
				return nil, fmt.Errorf("option %q: flag cannot declare string validation fields", e.Long)
			}
		}
		if (e.Type == "int" || e.Type == "float" || e.Type == "bool") && (e.MinLength != nil || e.MaxLength != nil || e.Pattern != "") {
			return nil, fmt.Errorf("option %q: %s type cannot use minLength, maxLength, or pattern", e.Long, e.Type)
		}
		if e.Type == "enum" && len(e.Enum) == 0 {
			return nil, fmt.Errorf("option %q: enum type must declare enum values", e.Long)
		}
		if e.Type != "enum" && len(e.Enum) > 0 {
			return nil, fmt.Errorf("option %q: enum values declared but type is %q", e.Long, e.Type)
		}
		if e.MinLength != nil && *e.MinLength < 0 {
			return nil, fmt.Errorf("option %q: minLength must be >= 0", e.Long)
		}
		if e.MaxLength != nil && *e.MaxLength < 0 {
			return nil, fmt.Errorf("option %q: maxLength must be >= 0", e.Long)
		}
		if e.MinLength != nil && e.MaxLength != nil && *e.MinLength > *e.MaxLength {
			return nil, fmt.Errorf("option %q: minLength greater than maxLength", e.Long)
		}
		if e.Type != "list" && (e.MinItems != nil || e.MaxItems != nil) {
			return nil, fmt.Errorf("option %q: minItems/maxItems can only be used with type list", e.Long)
		}
		if e.MinItems != nil && *e.MinItems < 0 {
			return nil, fmt.Errorf("option %q: minItems must be >= 0", e.Long)
		}
		if e.MaxItems != nil && *e.MaxItems < 0 {
			return nil, fmt.Errorf("option %q: maxItems must be >= 0", e.Long)
		}
		if e.MinItems != nil && e.MaxItems != nil && *e.MinItems > *e.MaxItems {
			return nil, fmt.Errorf("option %q: minItems greater than maxItems", e.Long)
		}
		if e.Failure != "" && e.Pattern == "" {
			return nil, fmt.Errorf("option %q: failure message declared but no pattern specified", e.Long)
		}
		if e.Pattern != "" {
			re, err := regexp.Compile(e.Pattern)
			if err != nil {
				return nil, fmt.Errorf("option %q: invalid pattern: %w", e.Long, err)
			}
			e.CompiledPattern = re
		}
		if e.Default != "" {
			if err := validateValue(*e, e.Default); err != nil {
				return nil, fmt.Errorf("option %q: invalid default: %w", e.Long, err)
			}
		}
	}

	return entries, nil
}

func isValidName(s string) bool {
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, " \t\n\r\x00=;") {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// dedent removes common leading indentation from all non-empty lines.
// Useful for callers that pass indented heredocs so schema lines can be
// written with natural indentation in shell scripts.
func dedent(s string) string {
	// Space-only dedent: count leading spaces on non-empty lines
	lines := strings.Split(s, "\n")
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		i := 0
		for i < len(l) && l[i] == ' ' {
			i++
		}
		if min == -1 || i < min {
			min = i
		}
	}
	if min > 0 {
		for idx, l := range lines {
			if strings.TrimSpace(l) == "" {
				lines[idx] = ""
				continue
			}
			if len(l) >= min {
				lines[idx] = l[min:]
			} else {
				lines[idx] = strings.TrimLeft(l, " ")
			}
		}
	}
	return strings.Join(lines, "\n")
}

// splitFields splits a schema line into key=value fields separated by
// unquoted semicolons. Values are expected to be quoted (start with ").
func splitFields(line string) ([]string, error) {
	var out []string
	inQuotes := false
	esc := false
	start := 0
	// iterate by byte to avoid rune allocations; quotes and escapes are ASCII
	for i := 0; i < len(line); i++ {
		b := line[i]
		if esc {
			esc = false
			continue
		}
		if inQuotes && b == '\\' {
			esc = true
			continue
		}
		if b == '"' {
			inQuotes = !inQuotes
			continue
		}
		if b == ';' && !inQuotes {
			field := strings.TrimSpace(line[start:i])
			if field != "" {
				out = append(out, field)
			}
			start = i + 1
		}
	}
	if inQuotes {
		return nil, errors.New("unterminated quoted value")
	}
	if start < len(line) {
		field := strings.TrimSpace(line[start:])
		if field != "" {
			out = append(out, field)
		}
	}
	return out, nil
}

// splitEnum splits an enum value on unescaped commas and unescapes \, sequences.
func splitEnum(s string) []string {
	var out []string
	var cur []byte
	esc := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if esc {
			// accept any escaped char literally (e.g., ", or ,)
			cur = append(cur, b)
			esc = false
			continue
		}
		if b == '\\' {
			esc = true
			continue
		}
		if b == ',' {
			out = append(out, strings.TrimSpace(string(cur)))
			cur = cur[:0]
			continue
		}
		cur = append(cur, b)
	}
	out = append(out, strings.TrimSpace(string(cur)))
	return out
}

func parseArgs(args []string, schema []schemaEntry) (map[string]string, error) {
	shortMapping := map[string]schemaEntry{}
	longMapping := map[string]schemaEntry{}
	for _, e := range schema {
		if e.Short != "" {
			shortMapping[e.Short] = e
		}
		longMapping[e.Long] = e
	}

	listValues := map[string][]string{}
	result := map[string]string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return nil, fmt.Errorf("unsupported positional argument after --: %q", args[i+1])
			}
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return nil, fmt.Errorf("unsupported argument: %q", arg)
		}

		entry, key, val, consumedNext, err := parseOption(arg, args, i, shortMapping, longMapping)
		if err != nil {
			return nil, err
		}
		if consumedNext {
			i++
		}

		switch entry.Type {
		case "flag":
			result[key] = "true"
		case "list":
			listValues[key] = append(listValues[key], val)
		default:
			result[key] = val
		}
	}

	delim := os.Getenv("GO_SHOPTS_LIST_DELIM")
	if delim == "" {
		delim = ","
	}
	for key, vals := range listValues {
		entry := longMapping[key]
		if entry.MinItems != nil && len(vals) < *entry.MinItems {
			return nil, fmt.Errorf("option %q requires at least %d items, got %d", displayName(entry), *entry.MinItems, len(vals))
		}
		if entry.MaxItems != nil && len(vals) > *entry.MaxItems {
			return nil, fmt.Errorf("option %q allows at most %d items, got %d", displayName(entry), *entry.MaxItems, len(vals))
		}
		for _, item := range vals {
			if err := validateValue(entry, item); err != nil {
				return nil, fmt.Errorf("option %q invalid: %s", displayName(entry), err.Error())
			}
		}
		result[key] = strings.Join(vals, delim)
	}

	return result, nil
}

func parseOption(arg string, args []string, index int, shortMapping, longMapping map[string]schemaEntry) (schemaEntry, string, string, bool, error) {
	var entry schemaEntry
	var key, inlineValue string

	if strings.HasPrefix(arg, "--") {
		name := strings.TrimPrefix(arg, "--")
		if name == "" {
			return entry, "", "", false, errors.New("invalid option: --")
		}
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			inlineValue = name[idx+1:]
			name = name[:idx]
		}
		mapped, ok := longMapping[name]
		if !ok {
			return entry, "", "", false, fmt.Errorf("unknown option: --%s", name)
		}
		entry = mapped
		key = entry.Long
	} else {
		name := strings.TrimPrefix(arg, "-")
		if name == "" {
			return entry, "", "", false, errors.New("invalid option: -")
		}
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			inlineValue = name[idx+1:]
			name = name[:idx]
		}
		if len(name) != 1 {
			return entry, "", "", false, fmt.Errorf("unsupported short option bundle: -%s", name)
		}
		mapped, ok := shortMapping[name]
		if !ok {
			return entry, "", "", false, fmt.Errorf("unknown option: -%s", name)
		}
		entry = mapped
		key = entry.Long
	}

	if entry.Type == "flag" {
		if inlineValue != "" {
			return entry, "", "", false, fmt.Errorf("option %q does not take a value", displayName(entry))
		}
		return entry, key, "true", false, nil
	}

	if inlineValue != "" {
		return entry, key, inlineValue, false, nil
	}
	if index+1 >= len(args) {
		return entry, "", "", false, fmt.Errorf("option %q requires a value", displayName(entry))
	}
	return entry, key, args[index+1], true, nil
}

func validateParsedValues(schema []schemaEntry, values map[string]string) []string {
	var problems []string
	for _, entry := range schema {
		val, ok := values[entry.Long]
		if !ok || (entry.Required && val == "") {
			if entry.Required {
				problems = append(problems, fmt.Sprintf("missing required option %q", displayName(entry)))
			}
			continue
		}
		if strings.ContainsRune(val, '\n') {
			problems = append(problems, fmt.Sprintf("option %q value contains newline", displayName(entry)))
			continue
		}
		if strings.ContainsRune(val, '\x00') {
			problems = append(problems, fmt.Sprintf("option %q value contains NUL", displayName(entry)))
			continue
		}
		if entry.Type != "list" {
			if err := validateValue(entry, val); err != nil {
				problems = append(problems, fmt.Sprintf("option %q invalid: %s", displayName(entry), err.Error()))
			}
		}
	}
	return problems
}

func validateValue(entry schemaEntry, value string) error {
	switch entry.Type {
	case "flag":
		if value != "true" && value != "false" {
			return errors.New("flag value must be true or false")
		}
		return nil
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("int value required: %v", err)
		}
	case "float":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("float value required: %v", err)
		}
	case "bool":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("bool value required: %v", err)
		}
	}

	if entry.MinLength != nil && len(value) < *entry.MinLength {
		return fmt.Errorf("must be at least %d characters long", *entry.MinLength)
	}
	if entry.MaxLength != nil && len(value) > *entry.MaxLength {
		return fmt.Errorf("must be no more than %d characters long", *entry.MaxLength)
	}
	if len(entry.Enum) > 0 {
		for _, allowed := range entry.Enum {
			if value == allowed {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(entry.Enum, ", "))
	}
	if entry.CompiledPattern != nil {
		if !entry.CompiledPattern.MatchString(value) {
			if entry.Failure != "" {
				return fmt.Errorf("%s", entry.Failure)
			}
			return errors.New("value did not match the expected format")
		}
	}
	return nil
}

func resolvedValue(entry schemaEntry, values map[string]string) (string, bool) {
	if val, ok := values[entry.Long]; ok {
		return val, true
	}
	if entry.Default != "" {
		return entry.Default, true
	}
	if entry.Type == "flag" {
		return "false", true
	}
	return "", false
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

// errWriter wraps an io.Writer and captures the first write error so callers
// can check once at the end rather than after every individual write.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) println(args ...any) {
	if ew.err == nil {
		_, ew.err = fmt.Fprintln(ew.w, args...)
	}
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err == nil {
		_, ew.err = fmt.Fprintf(ew.w, format, args...)
	}
}

func printUsage(w io.Writer, schema []schemaEntry) error {
	ew := &errWriter{w: w}
	ew.println("Usage: shopts SCHEMA [OPTIONS]")
	ew.println()
	ew.println("Options:")
	for _, entry := range schema {
		ew.printf("  %-24s %s\n", usageLabel(entry), usageSummary(entry))
		if entry.Description != "" {
			for _, line := range strings.Split(entry.Description, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				ew.printf("  %-24s %s\n", "", line)
			}
		}
	}
	ew.println("  -h, --help               Show schema-derived usage and exit")
	ew.println("  -V, --version            Print version and exit")
	ew.println()
	ew.println("Environment variables:")
	ew.println("  GO_SHOPTS_UPCASE=1       Output variable names in uppercase")
	ew.println("  GO_SHOPTS_LIST_DELIM=,   Delimiter for list-type options (default: ',')")
	ew.println("  GO_SHOPTS_PREFIX=X_      Override output variable prefix (default: 'GO_SHOPTS_')")
	ew.println()
	ew.println("Type notes:")
	ew.println("  int, float, bool: parsed and validated as native Go types")
	ew.println("  list: option may be repeated, values joined by delimiter")
	return ew.err
}

func usageLabel(entry schemaEntry) string {
	var parts []string
	if entry.Short != "" {
		parts = append(parts, "-"+entry.Short)
	}
	parts = append(parts, "--"+entry.Long)
	label := strings.Join(parts, ", ")
	if entry.Type != "flag" {
		label += " <value>"
	}
	return label
}

func usageSummary(entry schemaEntry) string {
	var parts []string
	if entry.Help != "" {
		parts = append(parts, entry.Help)
	}
	parts = append(parts, humanType(entry.Type))
	if entry.Required {
		parts = append(parts, "required")
	}
	if entry.Default != "" {
		parts = append(parts, "default: "+entry.Default)
	}
	if len(entry.Enum) > 0 {
		parts = append(parts, "allowed: "+strings.Join(entry.Enum, ", "))
	}
	if entry.MinLength != nil {
		parts = append(parts, fmt.Sprintf("minimum length: %d", *entry.MinLength))
	}
	if entry.MaxLength != nil {
		parts = append(parts, fmt.Sprintf("maximum length: %d", *entry.MaxLength))
	}
	if entry.MinItems != nil {
		parts = append(parts, fmt.Sprintf("minimum items: %d", *entry.MinItems))
	}
	if entry.MaxItems != nil {
		parts = append(parts, fmt.Sprintf("maximum items: %d", *entry.MaxItems))
	}
	if entry.Pattern != "" {
		if entry.Failure != "" {
			parts = append(parts, "format: "+entry.Failure)
		} else {
			parts = append(parts, "must match: "+entry.Pattern)
		}
	}
	return strings.Join(parts, "; ")
}

func displayName(entry schemaEntry) string {
	if entry.Long != "" {
		return "--" + entry.Long
	}
	return "-" + entry.Short
}

func getenvBool(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes"
}

func shVarName(long, prefix string, upcase bool) (string, error) {
	if long == "" {
		return "", errors.New("empty long name")
	}
	if upcase {
		long = strings.ToUpper(long)
	}
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, long)
	if !bashVarRE.MatchString(sanitized) {
		return "", fmt.Errorf("invalid variable name after sanitization: %q", sanitized)
	}
	return prefix + sanitized, nil
}
