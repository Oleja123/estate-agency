package geocoder

// GeoService provides geocoding capabilities (address -> latitude, longitude).
// Keep this package small: production implementations may call external APIs; tests use the Noop implementation.
type GeoService interface {
	Geocode(address string) (float64, float64, error)
}

// NoopService is a simple GeoService that returns zero coordinates.
type NoopService struct{}

func NewNoop() *NoopService { return &NoopService{} }

func (n *NoopService) Geocode(address string) (float64, float64, error) {
	// Noop returns 0,0 for any address. Real implementations should call external geocoders.
	return 0, 0, nil
}
