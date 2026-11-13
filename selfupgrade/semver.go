package selfupgrade

import (
	"regexp"
	"strconv"
	"strings"
)

// semverRe is node-semver's strict FULL pattern: an optional leading v or =,
// major.minor.patch without leading zeros, optional prerelease and build.
var semverRe = regexp.MustCompile(`^[v=]?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][a-zA-Z0-9-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][a-zA-Z0-9-]*))*))?` +
	`(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

// semver is a parsed semantic version.
type semver struct {
	major, minor, patch int
	pre                 []string
}

// parseSemver parses a version as node-semver's valid(clean(v)) does:
// surrounding spaces and a leading v or = are ignored, build metadata is
// dropped.
func parseSemver(s string) (semver, bool) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	v := semver{}
	var err error
	if v.major, err = strconv.Atoi(m[1]); err != nil {
		return semver{}, false
	}
	if v.minor, err = strconv.Atoi(m[2]); err != nil {
		return semver{}, false
	}
	if v.patch, err = strconv.Atoi(m[3]); err != nil {
		return semver{}, false
	}
	if m[4] != "" {
		v.pre = strings.Split(m[4], ".")
	}
	return v, true
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// compareSemvers orders versions as semver 2.0 does: a version without a
// prerelease is higher than the same version with one.
func compareSemvers(a, b semver) int {
	if c := compareInt(a.major, b.major); c != 0 {
		return c
	}
	if c := compareInt(a.minor, b.minor); c != 0 {
		return c
	}
	if c := compareInt(a.patch, b.patch); c != 0 {
		return c
	}
	switch {
	case len(a.pre) == 0 && len(b.pre) == 0:
		return 0
	case len(a.pre) == 0:
		return 1
	case len(b.pre) == 0:
		return -1
	}
	for i := 0; ; i++ {
		switch {
		case i >= len(a.pre) && i >= len(b.pre):
			return 0
		case i >= len(a.pre):
			return -1
		case i >= len(b.pre):
			return 1
		}
		x, y := a.pre[i], b.pre[i]
		if x == y {
			continue
		}
		xn, yn := isNumeric(x), isNumeric(y)
		switch {
		case xn && yn:
			xi, _ := strconv.Atoi(x)
			yi, _ := strconv.Atoi(y)
			return compareInt(xi, yi)
		case xn:
			return -1
		case yn:
			return 1
		case x < y:
			return -1
		default:
			return 1
		}
	}
}

// semverCompare orders two semantic versions as semver 2.0 does (-1, 0 or
// 1), with a leading v or = and build metadata ignored. ok is false when
// either is not a valid version.
func semverCompare(a, b string) (cmp int, ok bool) {
	va, okA := parseSemver(a)
	vb, okB := parseSemver(b)
	if !okA || !okB {
		return 0, false
	}
	return compareSemvers(va, vb), true
}

// semverValid reports whether s is a valid semantic version (a leading v
// or = and build metadata allowed).
func semverValid(s string) bool {
	_, ok := parseSemver(s)
	return ok
}
