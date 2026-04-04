package shopts

import (
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// builtinRE detects a whole-value {{ Name }} template in a pattern field.
// Accepts any spacing: {{Name}}, {{ Name }}, {{ Name}}, {{Name }}.
var builtinRE = regexp.MustCompile(`^\{\{\s*([A-Za-z][A-Za-z0-9]*)\s*\}\}$`)

// Pre-compiled regexes for regex-backed validators (anchored, whole-value).
var (
	reEmail     = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
	reURL       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+\-.]*://\S+$`)
	reScheme    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+\-.]*$`)
	reDomain    = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?)*$`)
	reSubdomain = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9\-]{0,61}[A-Za-z0-9])?$`)
	reURLPath   = regexp.MustCompile(`^/[^\s]*$`)
	reQuery     = regexp.MustCompile(`^\?[^\s#]*$`)
	reFragment  = regexp.MustCompile(`^#[^\s]*$`)
	reAbsPath   = regexp.MustCompile(`^/[^\x00]*$`)
	reRelPath   = regexp.MustCompile(`^\.\.?/[^\x00]*$`)
	reGitRef    = regexp.MustCompile(`^(HEAD|refs/[A-Za-z0-9._/\-]+|[A-Za-z0-9][A-Za-z0-9._/\-]*)$`)
	reGitSHA    = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	reEnvVar    = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	// dot-separated alphanumeric+hyphen identifiers, used for semver pre-release and build
	reSemVerMeta = regexp.MustCompile(`^[0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*$`)
)

// validateSemVer checks MAJOR.MINOR.PATCH[-pre][+build] per semver.org 2.0.0.
func validateSemVer(v string) bool {
	// Strip build metadata (everything after the first +).
	if i := strings.IndexByte(v, '+'); i >= 0 {
		if !reSemVerMeta.MatchString(v[i+1:]) {
			return false
		}
		v = v[:i]
	}
	// Strip pre-release (everything after the first -).
	if i := strings.IndexByte(v, '-'); i >= 0 {
		if !reSemVerMeta.MatchString(v[i+1:]) {
			return false
		}
		v = v[:i]
	}
	// Remaining must be exactly MAJOR.MINOR.PATCH.
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		// No leading zeros (except bare "0").
		if len(p) > 1 && p[0] == '0' {
			return false
		}
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// builtinValidators maps template names to validator functions.
// Stdlib-backed: IPv4Address, IPv6Address, CIDRBlock, PortNumber, SemVer.
// Regex-backed: everything else.
var builtinValidators = map[string]func(string) bool{
	"EmailAddress": func(v string) bool { return reEmail.MatchString(v) },
	"URL":          func(v string) bool { return reURL.MatchString(v) },
	"URLScheme":    func(v string) bool { return reScheme.MatchString(v) },
	"DomainName":   func(v string) bool { return reDomain.MatchString(v) },
	"Subdomain":    func(v string) bool { return reSubdomain.MatchString(v) },
	"URLPath":      func(v string) bool { return reURLPath.MatchString(v) },
	"QueryString":  func(v string) bool { return reQuery.MatchString(v) },
	"Fragment":     func(v string) bool { return reFragment.MatchString(v) },
	"IPv4Address": func(v string) bool {
		ip := net.ParseIP(v)
		return ip != nil && ip.To4() != nil
	},
	"IPv6Address": func(v string) bool {
		ip := net.ParseIP(v)
		return ip != nil && ip.To4() == nil
	},
	"CIDRBlock": func(v string) bool {
		_, _, err := net.ParseCIDR(v)
		return err == nil
	},
	"AbsolutePath": func(v string) bool { return reAbsPath.MatchString(v) },
	"RelativePath": func(v string) bool { return reRelPath.MatchString(v) },
	"GitRef":       func(v string) bool { return reGitRef.MatchString(v) },
	"GitSHA":       func(v string) bool { return reGitSHA.MatchString(v) },
	"SemVer":       validateSemVer,
	"PortNumber": func(v string) bool {
		n, err := strconv.Atoi(v)
		return err == nil && n >= 1 && n <= 65535
	},
	"EnvVar": func(v string) bool { return reEnvVar.MatchString(v) },
}

// lookupBuiltinValidator checks whether raw is a {{ Name }} template.
//
// Returns (fn, "") when raw is not a template at all — caller treats it as a
// raw regex. Returns (fn, name) when raw is a template: fn is non-nil on
// success, nil when the name is unrecognised (caller should return a schema
// error referencing the name).
func lookupBuiltinValidator(raw string) (func(string) bool, string) {
	m := builtinRE.FindStringSubmatch(raw)
	if m == nil {
		return nil, ""
	}
	name := m[1]
	return builtinValidators[name], name
}

// builtinValidatorNames returns the sorted list of all recognised template
// names, for use in error messages.
func builtinValidatorNames() []string {
	names := make([]string, 0, len(builtinValidators))
	for k := range builtinValidators {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
