package main

import (
	"fmt"
	"github.com/chenhg5/cc-connect/core"
	"os"
	"strings"
)

func disabledUpdateMessage() string {
	locale := ""
	for _, key := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale = os.Getenv(key); locale != "" {
			break
		}
	}
	locale = strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	lang := core.LangEnglish
	switch {
	case strings.HasPrefix(locale, "zh-tw"), strings.HasPrefix(locale, "zh-hk"):
		lang = core.LangTraditionalChinese
	case strings.HasPrefix(locale, "zh"):
		lang = core.LangChinese
	case strings.HasPrefix(locale, "ja"):
		lang = core.LangJapanese
	case strings.HasPrefix(locale, "es"):
		lang = core.LangSpanish
	}
	return core.NewI18n(lang).T(core.MsgSelfUpdateDisabled)
}

func checkUpdateAsync()                {}
func getUpdateHintIfAvailable() string { return "\n" + disabledUpdateMessage() + "\n" }
func checkUpdate()                     { fmt.Println(disabledUpdateMessage()) }
func runUpdate()                       { fmt.Fprintln(os.Stderr, disabledUpdateMessage()); os.Exit(1) }
func replaceExecutable(target, src string) error {
	return fmt.Errorf("%s", core.SelfUpdateDisabledMessage)
}

func isNewer(latest, current string) bool {
	if latest == "" || current == "" {
		return false
	}
	if strings.HasPrefix(current, "dev") {
		return true
	}

	l := strings.TrimPrefix(latest, "v")
	c := strings.TrimPrefix(current, "v")

	lBase, lPre, _ := strings.Cut(l, "-")
	cBase, cPre, _ := strings.Cut(c, "-")

	lParts := strings.Split(lBase, ".")
	cParts := strings.Split(cBase, ".")

	for i := 0; i < len(lParts) || i < len(cParts); i++ {
		var lv, cv int
		if i < len(lParts) {
			_, _ = fmt.Sscanf(lParts[i], "%d", &lv)
		}
		if i < len(cParts) {
			_, _ = fmt.Sscanf(cParts[i], "%d", &cv)
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}

	// Same base version — compare pre-release suffix
	// No pre-release beats a pre-release (1.2.0 > 1.2.0-beta.1)
	if cPre != "" && lPre == "" {
		return true
	}
	if cPre == "" && lPre != "" {
		return false
	}
	// Both have pre-release: split on "." and compare each segment
	// numerically where possible so beta.10 > beta.2.
	if lPre != "" && cPre != "" {
		return comparePreRelease(lPre, cPre) > 0
	}

	return false
}

// comparePreRelease compares two pre-release strings segment by segment.
// Numeric segments are compared as integers; non-numeric segments are
// compared lexicographically. Returns >0 if a is greater, <0 if b is
// greater, 0 if equal.
func comparePreRelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}
	for i := 0; i < max; i++ {
		var ap, bp string
		if i < len(aParts) {
			ap = aParts[i]
		}
		if i < len(bParts) {
			bp = bParts[i]
		}

		var an, bn int
		aN, _ := fmt.Sscanf(ap, "%d", &an)
		bN, _ := fmt.Sscanf(bp, "%d", &bn)
		aIsNum := aN == 1 && fmt.Sprintf("%d", an) == ap
		bIsNum := bN == 1 && fmt.Sprintf("%d", bn) == bp

		if aIsNum && bIsNum {
			if an != bn {
				return an - bn
			}
			continue
		}
		// Non-numeric: lexicographic
		if ap < bp {
			return -1
		}
		if ap > bp {
			return 1
		}
	}
	return 0
}
