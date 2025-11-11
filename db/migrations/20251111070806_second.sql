-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE property_types (
    id SERIAL PRIMARY KEY,
    property_name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_property_types_name ON property_types(property_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE property_types;
-- +goose StatementEnd
