package shopts

import (
	"strings"
	"testing"
)

// TestLookupBuiltinValidator checks the detection and dispatch logic.
func TestLookupBuiltinValidator(t *testing.T) {
	t.Run("not a template", func(t *testing.T) {
		fn, name := lookupBuiltinValidator(`^\d+$`)
		if fn != nil || name != "" {
			t.Fatalf("expected (nil, \"\"), got fn=<non-nil> name=%q", name)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		fn, name := lookupBuiltinValidator("")
		if fn != nil || name != "" {
			t.Fatalf("expected (nil, \"\"), got fn=<non-nil> name=%q", name)
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		fn, name := lookupBuiltinValidator("{{ Bogus }}")
		if name != "Bogus" {
			t.Fatalf("expected name=%q got %q", "Bogus", name)
		}
		if fn != nil {
			t.Fatal("expected nil fn for unknown name")
		}
	})

	t.Run("spacing variants", func(t *testing.T) {
		variants := []string{
			"{{EmailAddress}}",
			"{{ EmailAddress }}",
			"{{EmailAddress }}",
			"{{ EmailAddress}}",
		}
		for _, v := range variants {
			fn, name := lookupBuiltinValidator(v)
			if name != "EmailAddress" {
				t.Errorf("input %q: expected name=%q got %q", v, "EmailAddress", name)
			}
			if fn == nil {
				t.Errorf("input %q: expected non-nil fn", v)
			}
		}
	})

	t.Run("all known names resolve", func(t *testing.T) {
		for _, name := range builtinValidatorNames() {
			fn, got := lookupBuiltinValidator("{{ " + name + " }}")
			if got != name {
				t.Errorf("name %q: got back %q", name, got)
			}
			if fn == nil {
				t.Errorf("name %q: fn is nil", name)
			}
		}
	})
}

// TestParseSchema_BuiltinValidator checks schema-level integration.
func TestParseSchema_BuiltinValidator(t *testing.T) {
	t.Run("each builtin parses ok", func(t *testing.T) {
		for _, name := range builtinValidatorNames() {
			schema := "long=x, type=string, pattern={{ " + name + " }};"
			entries, err := parseSchema(schema)
			if err != nil {
				t.Errorf("validator %q: unexpected schema error: %v", name, err)
				continue
			}
			e := entries[0]
			if e.BuiltinValidator == nil {
				t.Errorf("validator %q: BuiltinValidator is nil", name)
			}
			if e.CompiledPattern != nil {
				t.Errorf("validator %q: CompiledPattern should be nil", name)
			}
		}
	})

	t.Run("unknown template is a schema error", func(t *testing.T) {
		_, err := parseSchema("long=x, type=string, pattern={{ Bogus }};")
		if err == nil {
			t.Fatal("expected schema error for unknown template")
		}
		if !strings.Contains(err.Error(), "unknown built-in validator") {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(err.Error(), "Bogus") {
			t.Errorf("error should mention the bad name: %v", err)
		}
	})

	t.Run("raw regex still compiles normally", func(t *testing.T) {
		entries, err := parseSchema(`long=x, type=string, pattern=^\d{3}-\d{4}$;`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		e := entries[0]
		if e.CompiledPattern == nil {
			t.Fatal("CompiledPattern should be set for raw regex")
		}
		if e.BuiltinValidator != nil {
			t.Fatal("BuiltinValidator should be nil for raw regex")
		}
	})

	t.Run("failure= accepted alongside builtin", func(t *testing.T) {
		entries, err := parseSchema("long=x, type=string, pattern={{ EmailAddress }}, failure=not a valid email;")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entries[0].Failure != "not a valid email" {
			t.Errorf("Failure field not set correctly: %q", entries[0].Failure)
		}
	})

	t.Run("valid default passes builtin validation", func(t *testing.T) {
		_, err := parseSchema("long=x, type=string, pattern={{ SemVer }}, default=1.0.0;")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid default fails builtin validation", func(t *testing.T) {
		_, err := parseSchema("long=x, type=string, pattern={{ SemVer }}, default=not-a-version;")
		if err == nil {
			t.Fatal("expected error for invalid default")
		}
		if !strings.Contains(err.Error(), "invalid default") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestValidateValue_BuiltinValidator checks each validator accepts valid input
// and rejects invalid input.
func TestValidateValue_BuiltinValidator(t *testing.T) {
	cases := []struct {
		name    string
		valid   []string
		invalid []string
	}{
		{
			name:    "EmailAddress",
			valid:   []string{"user@example.com", "a+b@x.co", "first.last@sub.domain.org"},
			invalid: []string{"notanemail", "@nodomain", "user@", "user@domain"},
		},
		{
			name:    "URL",
			valid:   []string{"https://example.com", "http://example.com/path?q=1#frag", "ftp://files.example.com/file.txt"},
			invalid: []string{"example.com", "//example.com", "not-a-url", ""},
		},
		{
			name:    "URLScheme",
			valid:   []string{"https", "http", "ftp", "git+ssh", "s3"},
			invalid: []string{"123bad", "-nope", "has space", ""},
		},
		{
			name:    "DomainName",
			valid:   []string{"example.com", "sub.example.com", "github.com", "localhost"},
			invalid: []string{"has space.com", "-badstart.com", "a..b", ""},
		},
		{
			name:    "Subdomain",
			valid:   []string{"docs", "api", "my-service", "a"},
			invalid: []string{"has.dot", "-badstart", "has space", ""},
		},
		{
			name:    "URLPath",
			valid:   []string{"/", "/usr/local/bin", "/path/to/resource"},
			invalid: []string{"no-leading-slash", "relative/path", ""},
		},
		{
			name:    "QueryString",
			valid:   []string{"?", "?key=val", "?a=1&b=2"},
			invalid: []string{"key=val", "no-question-mark", ""},
		},
		{
			name:    "Fragment",
			valid:   []string{"#", "#section-2", "#top"},
			invalid: []string{"section-2", "no-hash", ""},
		},
		{
			name:    "IPv4Address",
			valid:   []string{"192.168.1.1", "0.0.0.0", "255.255.255.255", "10.0.0.1"},
			invalid: []string{"999.0.0.1", "192.168.1", "192.168.1.1.1", "not-an-ip", ""},
		},
		{
			name:    "IPv6Address",
			valid:   []string{"2001:db8::1", "::1", "fe80::1", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
			invalid: []string{"gggg::1", "192.168.1.1", "not-ipv6", ""},
		},
		{
			name:    "CIDRBlock",
			valid:   []string{"10.0.0.0/24", "192.168.0.0/16", "0.0.0.0/0", "10.1.2.3/32"},
			invalid: []string{"10.0.0.0/33", "10.0.0.0", "not-a-cidr", ""},
		},
		{
			name:    "AbsolutePath",
			valid:   []string{"/", "/usr/local/bin", "/etc/config.yaml"},
			invalid: []string{"usr/local", "relative/path", "./foo", ""},
		},
		{
			name:    "RelativePath",
			valid:   []string{"./config/file.yaml", "../parent/file", "./"},
			invalid: []string{"config/file", "/absolute/path", "no-dot-slash", ""},
		},
		{
			name:    "GitRef",
			valid:   []string{"main", "HEAD", "refs/heads/main", "refs/tags/v1.0.0", "feature/my-branch"},
			invalid: []string{"", " spaces"},
		},
		{
			name:    "GitSHA",
			valid:   []string{"a1b2c3d", "abc1234def5678901234567890123456789012345"[:40], "deadbeef"},
			invalid: []string{"ABCDEF0", "short", "gggggg0", ""},
		},
		{
			name:    "SemVer",
			valid:   []string{"1.0.0", "2.1.0-beta.1", "0.0.1+build.123", "2.1.0-beta.1+sha.abc"},
			invalid: []string{"v1.0.0", "1.0", "1.0.0.0", "01.0.0", ""},
		},
		{
			name:    "PortNumber",
			valid:   []string{"1", "80", "8080", "443", "65535"},
			invalid: []string{"0", "65536", "-1", "abc", ""},
		},
		{
			name:    "EnvVar",
			valid:   []string{"AWS_SECRET_ACCESS_KEY", "PATH", "_PRIVATE", "FOO123"},
			invalid: []string{"lower_case", "123START", "has-hyphen", ""},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			schema := "long=x, type=string, pattern={{ " + tc.name + " }};"
			entries, err := parseSchema(schema)
			if err != nil {
				t.Fatalf("schema error: %v", err)
			}
			entry := entries[0]

			for _, v := range tc.valid {
				if err := validateValue(entry, v); err != nil {
					t.Errorf("valid input %q rejected: %v", v, err)
				}
			}
			for _, v := range tc.invalid {
				if err := validateValue(entry, v); err == nil {
					t.Errorf("invalid input %q was accepted", v)
				}
			}
		})
	}
}

// TestValidateValue_BuiltinValidator_FailureOverride checks that failure=
// replaces the default error message.
func TestValidateValue_BuiltinValidator_FailureOverride(t *testing.T) {
	entries, err := parseSchema("long=x, type=string, pattern={{ SemVer }}, failure=must be a valid semver;")
	if err != nil {
		t.Fatalf("schema error: %v", err)
	}
	err = validateValue(entries[0], "not-semver")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "must be a valid semver" {
		t.Errorf("expected custom failure message, got: %q", err.Error())
	}
}
