package nlp

import (
	"regexp"
	"strings"
)

// CaseFolding converts text to lowercase
func CaseFolding(text string) string {
	return strings.ToLower(text)
}

// Strip removes leading, trailing and multiple internal spaces
func Strip(text string) string {
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// Tokenize splits text into words (tokens)
func Tokenize(text string) []string {
	tokens := strings.Fields(text)

	// Filter out empty tokens
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" {
			result = append(result, token)
		}
	}

	return result
}

// NormalizeSymbols removes extra spaces, special chars, and normalizes text
func Normalize(text string) string {
	text = CaseFolding(text)

	// Remove punctuation and special characters (keep only letters, digits and spaces)
	// This preserves Cyrillic and Latin letters
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	text = reg.ReplaceAllString(text, "")

	text = Strip(text)

	return text
}
