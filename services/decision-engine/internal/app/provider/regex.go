package provider

import "regexp"

func matchesAny(value string, patterns ...string) bool {
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(value) {
			return true
		}
	}
	return false
}
