package security

import (
	"strings"
)

// DangerousEnvVars lists variables capable of code injection, dynamic link hijacking,
// or shell corruption that must be stripped from any child process environment.
var DangerousEnvVars = map[string]struct{}{
	"LD_PRELOAD":            {},
	"LD_LIBRARY_PATH":       {},
	"LD_AUDIT":              {},
	"LD_ORIGIN_PATH":        {},
	"LD_DEBUG":              {},
	"DYLD_INSERT_LIBRARIES": {},
	"DYLD_LIBRARY_PATH":     {},
	"DYLD_FRAMEWORK_PATH":   {},
	"IFS":                   {},
	"BASH_ENV":              {},
	"ENV":                   {},
	"CDPATH":                {},
	"GLIBC_TUNABLES":        {},
}

// SanitizeEnvironment filters out dangerous variables from the provided environment slice.
// If PATH is missing from rawEnv, a safe default is injected.
func SanitizeEnvironment(rawEnv []string) []string {
	sanitized := make([]string, 0, len(rawEnv))
	hasPath := false

	for _, entry := range rawEnv {
		idx := strings.IndexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		key := entry[:idx]

		if _, blocked := DangerousEnvVars[key]; blocked {
			continue
		}

		if key == "PATH" {
			hasPath = true
		}

		sanitized = append(sanitized, entry)
	}

	if !hasPath {
		sanitized = append(sanitized, "PATH=/usr/local/bin:/usr/bin:/bin")
	}

	return sanitized
}
