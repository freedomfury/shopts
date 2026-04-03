package shopts

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

const sampleSchema = `
short=u;long=username;required=true;type=string;help=Username;minLength=3;
short=p;long=pass;required=true;type=string;help=Password;minLength=6;
short=v;long=verbose;required=false;type=flag;help=Verbose;
`

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()
	f()
	_ = w.Close()
	os.Stdout = old
	return <-outC
}

func TestRun_Help(t *testing.T) {
	out := captureStdout(func() {
		if err := Run([]string{"go-shopts", sampleSchema, "--help"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Usage: go-shopts") {
		t.Fatalf("expected help output, got: %q", out)
	}
}

func TestRun_ValidationAndOutput(t *testing.T) {
	got := captureStdout(func() {
		err := Run([]string{"go-shopts", sampleSchema, "-u", "alice", "-p", "s3cret", "-v"})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(got, "GO_GETOPT_username") {
		t.Fatalf("expected key output, got: %q", got)
	}
	if !strings.Contains(got, "alice") {
		t.Fatalf("expected value output, got: %q", got)
	}
}

func TestParseSchema_Invalid(t *testing.T) {
	_, err := parseSchema("short=x;type=invalid;long=foo;")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}
