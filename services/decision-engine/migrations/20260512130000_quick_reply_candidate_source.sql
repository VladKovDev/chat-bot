-- +goose Up
ALTER TABLE decision_candidates
    DROP CONSTRAINT IF EXISTS decision_candidates_source_check;

ALTER TABLE decision_candidates
    ADD CONSTRAINT decision_candidates_source_check
    CHECK (source IN ('intent_example', 'knowledge_chunk', 'exact_command', 'fallback', 'lexical_fuzzy', 'quick_reply_intent'));

-- +goose Down
ALTER TABLE decision_candidates
    DROP CONSTRAINT IF EXISTS decision_candidates_source_check;

ALTER TABLE decision_candidates
    ADD CONSTRAINT decision_candidates_source_check
    CHECK (source IN ('intent_example', 'knowledge_chunk', 'exact_command', 'fallback', 'lexical_fuzzy'));
