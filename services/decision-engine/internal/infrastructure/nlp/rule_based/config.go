package rule_based

type Intent struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type Threshold struct {
	MinConfidence  float64 `yaml:"min_confidence"`
	AmbiguityDelta float64 `yaml:"ambiguity_delta"`
}

type Config struct {
	Intents   []Intent  `yaml:"intents"`
	Threshold Threshold `yaml:"threshold"`
}