package shopts

import (
	"io"
	"testing"
)

const benchSchema = `
long=stringval, short=s, required=true, type=string, help="A required string value";
long=intval, short=i, type=int, default=42, min=0, help="Optional integer value";
long=floatval, short=f, type=float, default=3.14, help="Optional float value";
long=boolval, short=b, type=bool, default=false, help="Optional boolean value";
long=enumval, short=e, type=enum, enum="red,green,blue", default=green, help="Enum value";
long=listval, short=l, type=list, minItems=1, help="Optional list value";
long=flagval, short=F, type=flag, help="Optional flag";
long=defval, short=d, type=string, default=defaultval, help="Has a default";
long=email, short=E, type=string, pattern="[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}", failure="invalid email";
long=release, short=r, type=string, pattern={{ SemVer }};
`

var benchArgs = []string{"-s", "hello", "-i", "99", "-f", "2.71", "-b", "true", "-e", "blue",
	"-l", "a", "-l", "b", "-l", "c", "-F", "-d", "customdef", "-E", "user@example.com", "-r", "1.2.3"}

func BenchmarkParseSchema(b *testing.B) {
	for b.Loop() {
		if _, err := parseSchema(benchSchema); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScan(b *testing.B) {
	s, err := parseSchema(benchSchema)
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if sc := scan(s, benchArgs); len(sc.errs) > 0 {
			b.Fatal(sc.errs)
		}
	}
}

func BenchmarkRun(b *testing.B) {
	argv := append([]string{"shopts", benchSchema}, benchArgs...)
	for b.Loop() {
		if code := Run(argv, io.Discard, io.Discard, "bench"); code != ExitOK {
			b.Fatalf("exit %d", code)
		}
	}
}
