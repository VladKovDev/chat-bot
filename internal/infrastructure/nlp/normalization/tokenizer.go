package normalization

import (
	"regexp"
	"strings"
)
 

var tokenWordRe = regexp.MustCompile(`\p{L}+`)

func Tokenize(text string) []string {
	if text == "" {
		return []string{}
	}
 
	matches := tokenWordRe.FindAllString(strings.ToLower(text), -1)
	if matches == nil {
		return []string{}
	}
 
	return matches
}
 