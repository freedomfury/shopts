package shopts

import (
	"fmt"
	"net/netip"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"sync"
)

// validator is a named check usable as pattern={{ Name }}. Built-ins live in
// the builtins registry; schemas add their own with define entries; an
// inline regex is an unnamed one.
type validator struct {
	Name    string
	Summary string // what it accepts, for the README table
	Check   func(string) bool
	Failure string   // message when Check fails
	Valid   []string // examples Check must accept; run as tests
	Invalid []string // examples Check must reject; run as tests

	source string // the regex, for generic failure messages
}

// builtins is the registry of built-in validators. Adding a validator means
// adding one entry; its examples become tests and its README row is
// generated from it.
var builtins = []*validator{
	{
		Name:    "EmailAddress",
		Summary: "Email address",
		Check:   matches(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`),
		Failure: "must be an email address like user@example.com",
		Valid:   []string{"user@example.com", "first.last+tag@mail.example.org"},
		Invalid: []string{"user@", "@example.com", "user@example", "a b@example.com"},
	},
	{
		Name:    "URL",
		Summary: "Full URL with a scheme",
		Check:   matches(`[A-Za-z][A-Za-z0-9+\-.]*://\S+`),
		Failure: "must be a URL with a scheme, like https://example.com",
		Valid:   []string{"https://example.com/path?q=1", "git+ssh://git@host/repo"},
		Invalid: []string{"example.com", "https://", "http://has space"},
	},
	{
		Name:    "URLScheme",
		Summary: "URL scheme",
		Check:   matches(`[A-Za-z][A-Za-z0-9+\-.]*`),
		Failure: "must be a URL scheme like https",
		Valid:   []string{"https", "git+ssh"},
		Invalid: []string{"1http", "ht tp", "https://"},
	},
	{
		Name:    "DomainName",
		Summary: "Dot-separated DNS labels",
		Check:   matches(`[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?)*`),
		Failure: "must be a domain name like example.com",
		Valid:   []string{"github.com", "a.b-c.io", "localhost"},
		Invalid: []string{"-bad.com", "exa_mple.com", "example..com"},
	},
	{
		Name:    "Subdomain",
		Summary: "Single DNS label",
		Check:   matches(`[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?`),
		Failure: "must be a single DNS label like api",
		Valid:   []string{"docs", "api-v2"},
		Invalid: []string{"a.b", "-x", "x-"},
	},
	{
		Name:    "URLPath",
		Summary: "URL path starting with `/`",
		Check:   matches(`/\S*`),
		Failure: "must be a URL path starting with /",
		Valid:   []string{"/users/123", "/"},
		Invalid: []string{"users", "/a b"},
	},
	{
		Name:    "QueryString",
		Summary: "Query string starting with `?`",
		Check:   matches(`\?[^\s#]*`),
		Failure: "must be a query string starting with ?",
		Valid:   []string{"?key=val", "?"},
		Invalid: []string{"key=val", "?a#b"},
	},
	{
		Name:    "Fragment",
		Summary: "URL fragment starting with `#`",
		Check:   matches(`#\S*`),
		Failure: "must be a URL fragment starting with #",
		Valid:   []string{"#section-2"},
		Invalid: []string{"section", "#a b"},
	},
	{
		Name:    "IPv4Address",
		Summary: "IPv4 address",
		Check: func(v string) bool {
			a, err := netip.ParseAddr(v)
			return err == nil && a.Is4()
		},
		Failure: "must be a valid IPv4 address",
		Valid:   []string{"192.168.1.1", "10.0.0.0"},
		Invalid: []string{"999.1.1.1", "1.2.3", "01.2.3.4", "::1"},
	},
	{
		Name:    "IPv6Address",
		Summary: "IPv6 address, including IPv4-mapped",
		Check: func(v string) bool {
			a, err := netip.ParseAddr(v)
			return err == nil && a.Is6() && a.Zone() == ""
		},
		Failure: "must be a valid IPv6 address",
		Valid:   []string{"2001:db8::1", "::1", "::ffff:192.0.2.1"},
		Invalid: []string{"192.168.1.1", "2001:db8::g", "fe80::1%eth0"},
	},
	{
		Name:    "CIDRBlock",
		Summary: "IP address with a prefix length",
		Check: func(v string) bool {
			_, err := netip.ParsePrefix(v)
			return err == nil
		},
		Failure: "must be a CIDR block like 10.0.0.0/24",
		Valid:   []string{"10.0.0.0/24", "2001:db8::/32"},
		Invalid: []string{"10.0.0.0", "10.0.0.0/33", "10.0.0.0/024"},
	},
	{
		Name:    "AbsolutePath",
		Summary: "Path starting with `/`",
		Check:   matches(`/[^\x00]*`),
		Failure: "must be an absolute path starting with /",
		Valid:   []string{"/usr/local/bin", "/"},
		Invalid: []string{"usr/bin", "./x", ""},
	},
	{
		Name:    "RelativePath",
		Summary: "Path not starting with `/` or `-`",
		Check:   matches(`[^/\-\x00][^\x00]*`),
		Failure: "must be a relative path (not starting with / or -)",
		Valid:   []string{"config/file.yaml", "./config/file.yaml", "../up", ".", "./-rf"},
		Invalid: []string{"/etc/hosts", "", "-rf", "--no-preserve-root"},
	},
	{
		Name:    "GitRef",
		Summary: "Branch, tag, `HEAD` or `refs/` path, by git's naming rules",
		Check:   isGitRef,
		Failure: "must be a valid git ref name",
		Valid:   []string{"main", "feature/login", "v1.2.3", "HEAD", "refs/heads/main"},
		Invalid: []string{"feature..x", "-leading", "ends/", "has space", "x.lock", ".hidden",
			"a/.b", "a@{b", "@", "a//b", "ref^", "ref~1", "ref:x", `a\b`, "ends.", "what?", "a*", "a[b"},
	},
	{
		Name:    "GitSHA",
		Summary: "7–40 lowercase hex characters",
		Check:   matches(`[0-9a-f]{7,40}`),
		Failure: "must be a git SHA of 7 to 40 lowercase hex characters",
		Valid:   []string{"abc1234", "0123456789abcdef0123456789abcdef01234567"},
		Invalid: []string{"abc12", "ABC1234", "g123456"},
	},
	{
		Name:    "SemVer",
		Summary: "`MAJOR.MINOR.PATCH[-pre][+build]`",
		// The regular expression suggested by semver.org 2.0.0.
		Check: matches(`(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)` +
			`(?:-(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*)?` +
			`(?:\+[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)?`),
		Failure: "must be a semantic version like 1.2.3",
		Valid:   []string{"1.0.0", "2.1.3-rc.1+build.5", "1.0.0-0.3.7", "1.0.0-x.7.z.92", "1.0.0+001"},
		Invalid: []string{"1.0", "01.0.0", "1.0.0-", "v1.0.0", "1.0.0-01", "1.0.0-alpha..1", "1.0.0+"},
	},
	{
		Name:    "PortNumber",
		Summary: "Integer 1–65535, digits only",
		Check: func(v string) bool {
			if v == "" || v[0] < '1' || v[0] > '9' || strings.Trim(v, "0123456789") != "" {
				return false
			}
			n, err := strconv.Atoi(v)
			return err == nil && n <= 65535
		},
		Failure: "must be a port number from 1 to 65535",
		Valid:   []string{"8080", "1", "65535"},
		Invalid: []string{"0", "65536", "+80", "080", "http"},
	},
	{
		Name:    "EnvVar",
		Summary: "`SCREAMING_SNAKE_CASE` name",
		Check:   matches(`[A-Z_][A-Z0-9_]*`),
		Failure: "must be an uppercase variable name like MY_VAR",
		Valid:   []string{"PATH", "MY_VAR_2", "_X"},
		Invalid: []string{"my_var", "1VAR", "MY-VAR"},
	},
}

var builtinByName = func() map[string]*validator {
	m := make(map[string]*validator, len(builtins))
	for _, v := range builtins {
		m[v.Name] = v
	}
	return m
}()

var (
	// templateRE matches a whole-value {{ Name }} reference, any spacing.
	// Any name is captured, so a malformed one is an unknown validator
	// rather than a literal regex.
	templateRE      = lazyRegexp(`^\{\{\s*([^{}]*?)\s*\}\}$`)
	validatorNameRE = lazyRegexp(`^[A-Za-z][A-Za-z0-9]*$`)
)

// resolvePattern turns a pattern field into a validator: a built-in or
// defined {{ Name }}, or an inline regex.
func resolvePattern(p string, defines map[string]*validator) (*validator, error) {
	m := templateRE().FindStringSubmatch(p)
	if m == nil {
		return regexValidator(p)
	}
	if v, ok := builtinByName[m[1]]; ok {
		return v, nil
	}
	if v, ok := defines[m[1]]; ok {
		return v, nil
	}
	names := make([]string, len(builtins))
	for i, v := range builtins {
		names[i] = v.Name
	}
	return nil, fmt.Errorf("unknown validator %q (built-ins: %s; or declare it with a define entry)",
		m[1], strings.Join(names, ", "))
}

// regexValidator compiles an inline regex. Every pattern matches the whole
// value, so the regex is anchored at both ends. It is parsed on its own
// first, so its text cannot close or swallow the anchoring group.
func regexValidator(p string) (*validator, error) {
	parsed, err := syntax.Parse(p, syntax.Perl)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %v", err)
	}
	re := regexp.MustCompile(`^(?:` + parsed.String() + `)$`)
	return &validator{Check: re.MatchString, source: p}, nil
}

// matches returns a whole-value regex check.
func matches(p string) func(string) bool {
	re := lazyRegexp(`^(?:` + p + `)$`)
	return func(v string) bool { return re().MatchString(v) }
}

// lazyRegexp compiles expr on first use rather than at program start:
// shopts runs once per script call, and most schemas use few regexes or none.
func lazyRegexp(expr string) func() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(expr) })
}

// isGitRef applies git's ref naming rules (git check-ref-format
// --allow-onelevel), and also rejects a leading '-'.
func isGitRef(v string) bool {
	if v == "" || v == "@" || v[0] == '-' || v[0] == '/' ||
		strings.HasSuffix(v, "/") || strings.HasSuffix(v, ".") ||
		strings.Contains(v, "..") || strings.Contains(v, "//") || strings.Contains(v, "@{") {
		return false
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	for _, part := range strings.Split(v, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}
