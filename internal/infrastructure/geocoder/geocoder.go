package geocoder

type GeoService interface {
	Geocode(address string) (float64, float64, error)
}

type NoopService struct{}

func NewNoop() *NoopService { return &NoopService{} }

func (n *NoopService) Geocode(address string) (float64, float64, error) {

	return 0, 0, nil
}
