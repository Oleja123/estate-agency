-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE properties (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    property_description TEXT,
    type_id INTEGER NOT NULL REFERENCES property_types(id),
    transaction_type VARCHAR(10) NOT NULL CHECK (transaction_type IN ('sale', 'rent')),
    price DECIMAL(15,2) NOT NULL CHECK (price >= 0),
    area DECIMAL(10,2) NOT NULL CHECK (area > 0),
    property_address VARCHAR(500) NOT NULL,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    city VARCHAR(100) NOT NULL,
    property_status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (property_status IN ('active', 'sold', 'rented', 'inactive')),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_properties_type_id ON properties(type_id);
CREATE INDEX idx_properties_transaction_type ON properties(transaction_type);
CREATE INDEX idx_properties_status ON properties(property_status);
CREATE INDEX idx_properties_city ON properties(city);
CREATE INDEX idx_properties_price ON properties(price);
CREATE INDEX idx_properties_created_by ON properties(created_by);
CREATE INDEX idx_properties_created_at ON properties(created_at);
CREATE INDEX idx_properties_location ON properties(latitude, longitude);

CREATE INDEX idx_properties_city_status ON properties(city, property_status);
CREATE INDEX idx_properties_type_transaction ON properties(type_id, transaction_type);
CREATE INDEX idx_properties_price_range ON properties(price) WHERE property_status = 'active';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE properties;
-- +goose StatementEnd
