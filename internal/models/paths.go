package models

import (
	"regexp"
	"strings"
)

var winPathRegex = regexp.MustCompile(`^(?:[A-Za-z]:|\\\\)`)

// CanonicalProject folds separator variants of the same folder into one project identity.
// Windows-shaped paths (drive-letter or UNC prefix) are unified to forward slashes;
// trailing slashes are trimmed. Non-path values, sentinels, and POSIX backslashes pass through.
func CanonicalProject(path string) string {
	if path == "" {
		return path
	}
	res := path
	if winPathRegex.MatchString(res) {
		res = strings.ReplaceAll(res, "\\", "/")
	}
	trimmed := strings.TrimRight(res, "/")
	if trimmed == "" {
		return res
	}
	return trimmed
}
