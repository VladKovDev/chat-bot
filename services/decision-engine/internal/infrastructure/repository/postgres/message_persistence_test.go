package postgres

import "testing"

func TestAllowedCandidateSource(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"intent_example",
		"knowledge_chunk",
		"exact_command",
		"fallback",
		"lexical_fuzzy",
	}
	for _, source := range allowed {
		if !allowedCandidateSource(source) {
			t.Fatalf("source %q should be allowed", source)
		}
	}

	if allowedCandidateSource("unsupported_source") {
		t.Fatal("unsupported_source should not be allowed")
	}
}
