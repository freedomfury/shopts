package shopts

import (
	"io"
	"os"
	"testing"
)

var benchSchema = `
long="stringval", short="s", required="true", type="string", help="A required string value";
long="intval", short="i", required="false", type="int", default="42", help="Optional integer value";
long="floatval", short="f", required="false", type="float", default="3.14", help="Optional float value";
long="boolval", short="b", required="false", type="bool", default="false", help="Optional boolean value";
long="enumval", short="e", required="false", type="enum", enum="red,green,blue", default="green", help="Enum value";
long="listval", short="l", required="false", type="list", minItems="1", help="Optional list value";
long="flagval", short="F", required="false", type="flag", help="Optional flag";
long="defval", short="d", required="false", type="string", default="defaultval", help="Has a default";
long="email", short="E", required="false", type="string", pattern="^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$", failure="invalid email", help="Email address";
`

var benchArgs = []string{"-s", "hello", "-i", "99", "-f", "2.71", "-b", "true", "-e", "blue", "-l", "a", "-l", "b", "-l", "c", "-F", "-d", "customdef", "-E", "user@example.com"}

func BenchmarkParseSchema(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := parseSchema(benchSchema); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseArgs(b *testing.B) {
	schema, err := parseSchema(benchSchema)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, errs := parseArgs(benchArgs, schema); len(errs) > 0 {
			b.Fatal(errs[0])
		}
	}
}

func BenchmarkValidateValue(b *testing.B) {
	schema, err := parseSchema(benchSchema)
	if err != nil {
		b.Fatal(err)
	}
	// pick the email entry for validation (exercises regex path)
	var emailEntry schemaEntry
	for _, e := range schema {
		if e.Long == "email" {
			emailEntry = e
			break
		}
	}
	if emailEntry.Long == "" {
		b.Fatal("email entry not found in schema")
	}
	value := "user@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validateValue(emailEntry, value); err != nil {
			b.Fatal(err)
		}
	}
}

var builtinSchema = `
long=email,   type=string, pattern={{ EmailAddress }};
long=version, type=string, pattern={{ SemVer }};
long=host,    type=string, pattern={{ IPv4Address }};
long=port,    type=string, pattern={{ PortNumber }};
long=sha,     type=string, pattern={{ GitSHA }};
long=envvar,  type=string, pattern={{ EnvVar }};
`

func BenchmarkParseSchema_Builtins(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := parseSchema(builtinSchema); err != nil {
			b.Fatal(err)
		}
	}
}

func benchBuiltin(b *testing.B, name, value string) {
	b.Helper()
	schema, err := parseSchema("long=x, type=string, pattern={{ " + name + " }};")
	if err != nil {
		b.Fatal(err)
	}
	entry := schema[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validateValue(entry, value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuiltin_EmailAddress(b *testing.B) {
	benchBuiltin(b, "EmailAddress", "user@example.com")
}

func BenchmarkBuiltin_SemVer(b *testing.B) {
	benchBuiltin(b, "SemVer", "1.2.3-beta.1+sha.abc123")
}

func BenchmarkBuiltin_IPv4Address(b *testing.B) {
	benchBuiltin(b, "IPv4Address", "192.168.1.1")
}

func BenchmarkBuiltin_PortNumber(b *testing.B) {
	benchBuiltin(b, "PortNumber", "8080")
}

func BenchmarkBuiltin_GitSHA(b *testing.B) {
	benchBuiltin(b, "GitSHA", "a1b2c3d4e5f6a1b2")
}

func BenchmarkBuiltin_EnvVar(b *testing.B) {
	benchBuiltin(b, "EnvVar", "AWS_SECRET_ACCESS_KEY")
}

func BenchmarkFullRun(b *testing.B) {
	// silence stdout to avoid noise and I/O interference
	origStdout := os.Stdout
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	defer func() {
		_ = devNull.Close()
		os.Stdout = origStdout
	}()

	for i := 0; i < b.N; i++ {
		os.Stdout = devNull
		schema, err := parseSchema(benchSchema)
		if err != nil {
			b.Fatal(err)
		}
		values, errs := parseArgs(benchArgs, schema)
		if len(errs) > 0 {
			b.Fatal(errs[0])
		}
		// Simulate output
		_ = values
	}
	// restore just in case
	os.Stdout = origStdout
	// ensure io is used so import isn't optimized away
	_ = io.Discard
}
