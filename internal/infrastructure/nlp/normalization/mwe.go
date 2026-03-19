package normalization

const negationParticle = "не"

func ApplyMWE(tokens []string) []string {
	if len(tokens) == 0 {
		return tokens
	}

	result := make([]string, 0, len(tokens))
	i := 0

	for i < len(tokens) {
		if tokens[i] == negationParticle && i+1 < len(tokens) {
			result = append(result, negationParticle+"_"+tokens[i+1])
			i += 2
		} else {
			result = append(result, tokens[i])
			i++
		}
	}

	return result
}