-- +goose Up
CREATE TABLE IF NOT EXISTS semantic_catalog_settings (
    key VARCHAR(80) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

INSERT INTO semantic_catalog_settings (key, value)
VALUES ('embedding_dimension', '384')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = now();

-- +goose Down
DROP TABLE IF EXISTS semantic_catalog_settings;
