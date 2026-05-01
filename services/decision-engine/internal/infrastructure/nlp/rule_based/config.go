package rule_based

type Intent struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Threshold struct {
	MinScore       float64 `json:"min_score"`
	AmbiguityDelta float64 `json:"ambiguity_delta"`
}

type Config struct {
	Intents   []Intent  `json:"intents"`
	Threshold Threshold `json:"threshold"`
}