package geocoder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	cfg "github.com/Oleja123/estate-agency/internal/infrastructure/config"
)

type GeoapifyService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewGeoapify(conf cfg.GeoServiceConfig, httpClient *http.Client) *GeoapifyService {
	base := conf.BaseURL
	if base == "" {
		base = "https://api.geoapify.com/v1/geocode/search"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &GeoapifyService{apiKey: conf.APIKey, baseURL: base, client: httpClient}
}

// geoapify response subset
type geoapifyResp struct {
	Features []struct {
		Properties struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"properties"`
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

func (g *GeoapifyService) Geocode(address string) (float64, float64, error) {
	if g == nil {
		return 0, 0, NewErrGeoConfig("сервис не инициализирован")
	}
	if g.apiKey == "" {
		return 0, 0, NewErrGeoConfig("пустой api_key")
	}

	// build request URL
	u, err := url.Parse(g.baseURL)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("text", address)
	q.Set("apiKey", g.apiKey)
	q.Set("limit", "1")
	u.RawQuery = q.Encode()

	resp, err := g.client.Get(u.String())
	if err != nil {
		return 0, 0, NewErrGeoRequest(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, NewErrGeoRequest(fmt.Sprintf("сервер вернул статус %d", resp.StatusCode))
	}

	var gr geoapifyResp
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return 0, 0, NewErrGeoDecode(err.Error())
	}

	if len(gr.Features) == 0 {
		return 0, 0, NewErrGeoNoResults(address)
	}

	f := gr.Features[0]
	// prefer geometry coordinates if present
	if len(f.Geometry.Coordinates) >= 2 {
		lon := f.Geometry.Coordinates[0]
		lat := f.Geometry.Coordinates[1]
		return lat, lon, nil
	}

	// fallback to properties
	if f.Properties.Lat != 0 || f.Properties.Lon != 0 {
		return f.Properties.Lat, f.Properties.Lon, nil
	}

	return 0, 0, NewErrGeoDecode("результат не содержит координат")
}
