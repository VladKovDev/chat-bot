package normalization

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed data/stopwords_ru.txt
var stopwordsRaw string

var (
	stopwordsOnce sync.Once
	stopwordsSet  map[string]struct{}
)

// loadStopwords parses the embedded stopwords file into a set.
// Called lazily on first use via sync.Once.
func loadStopwords() {
	stopwordsOnce.Do(func() {
		lines := strings.Split(stopwordsRaw, "\n")
		set := make(map[string]struct{}, len(lines))

		for _, line := range lines {
			word := strings.TrimSpace(line)
			if word == "" || strings.HasPrefix(word, "#") {
				continue
			}
			set[word] = struct{}{}
		}

		stopwordsSet = set
	})
}

// RemoveStopwords filters tokens that appear in the stopwords set.
func RemoveStopwords(tokens []string) []string {
	loadStopwords()

	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, isStop := stopwordsSet[t]; !isStop {
			result = append(result, t)
		}
	}

	return result
}
