-- +goose Up
-- +goose StatementBegin

-- Create table for property images
CREATE TABLE property_images (
    id SERIAL PRIMARY KEY,
    property_id INTEGER NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX idx_property_images_property_id ON property_images(property_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_property_images_property_id;
DROP TABLE IF EXISTS property_images;

-- +goose StatementEnd
