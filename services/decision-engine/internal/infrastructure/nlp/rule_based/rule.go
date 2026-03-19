package rule_based

import "strings"


type RuleType string

const (
	RuleKeyword RuleType = "keyword"
	RulePhrase  RuleType = "phrase"
)

type Rule struct {
	Type   RuleType
	Weight float64
	Values []string
}

func (r *Rule) Match(text string) ([]string, float64) {
	var matched []string
	var score float64
	for _, v := range r.Values {
		if strings.Contains(text, v) {
			matched = append(matched, v)
			score += r.Weight
			}
		}
	
	return matched, score
}
