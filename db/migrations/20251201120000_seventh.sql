-- +goose Up
-- +goose StatementBegin

-- Ensure property_address is not an empty string (whitespace-only also considered empty)
ALTER TABLE properties
  ADD CONSTRAINT chk_properties_property_address_not_empty
  CHECK (char_length(trim(property_address)) > 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE properties
  DROP CONSTRAINT IF EXISTS chk_properties_property_address_not_empty;

-- +goose StatementEnd
