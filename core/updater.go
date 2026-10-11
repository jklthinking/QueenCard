package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ReleaseInfo struct {
	TagName    string
	Name       string
	Body       string
	Prerelease bool
	CreatedAt  string
}

const SelfUpdateDisabledMessage = "QueenCard: self-update is disabled in this build; rebuild from source"

// CheckForUpdate deliberately performs no network requests.
func CheckForUpdate(currentVersion string, preferGitee bool) (*ReleaseInfo, error) {
	return nil, fmt.Errorf("%s", SelfUpdateDisabledMessage)
}
func SelfUpdate(tag string, preferGitee bool) error {
	return fmt.Errorf("%s", SelfUpdateDisabledMessage)
}
func replaceBinary(newBinary []byte) error { return fmt.Errorf("%s", SelfUpdateDisabledMessage) }

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

type semver struct {
	major, minor, patch int
	pre                 string
	preNum              int
}

func parseSemver(v string) semver {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return semver{}
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	pre := m[4]
	preNum := 0
	if idx := strings.LastIndex(pre, "."); idx >= 0 {
		preNum, _ = strconv.Atoi(pre[idx+1:])
	}
	return semver{major: major, minor: minor, patch: patch, pre: pre, preNum: preNum}
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// semverCompare returns >0 if a > b, <0 if a < b, 0 if equal.
func semverCompare(a, b string) int {
	sa := parseSemver(a)
	sb := parseSemver(b)

	if d := sa.major - sb.major; d != 0 {
		return d
	}
	if d := sa.minor - sb.minor; d != 0 {
		return d
	}
	if d := sa.patch - sb.patch; d != 0 {
		return d
	}
	// No pre-release > has pre-release (1.0.0 > 1.0.0-beta.1)
	if sa.pre == "" && sb.pre != "" {
		return 1
	}
	if sa.pre != "" && sb.pre == "" {
		return -1
	}
	// Both have pre-release: compare lexicographically, then by number
	if sa.pre != sb.pre {
		if d := strings.Compare(sa.pre, sb.pre); d != 0 {
			// "beta" prefix comparison; if same prefix, compare numbers
			aPre := strings.TrimRight(sa.pre, "0123456789.")
			bPre := strings.TrimRight(sb.pre, "0123456789.")
			if aPre == bPre {
				return sa.preNum - sb.preNum
			}
			return d
		}
	}
	return 0
}
