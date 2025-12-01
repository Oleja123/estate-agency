package geocoder

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	cfg "github.com/Oleja123/estate-agency/internal/infrastructure/config"
)

func TestGeoapifyGeocodeSuccess(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"features":[{"geometry":{"coordinates":[37.123,55.456]}}]}`)
	}))
	defer ts.Close()

	svc := NewGeoapify(cfg.GeoServiceConfig{APIKey: "key", BaseURL: ts.URL}, &http.Client{})
	lat, lon, err := svc.Geocode("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lat != 55.456 || lon != 37.123 {
		t.Fatalf("unexpected coords: %v,%v", lat, lon)
	}
}

func TestGeoapifyGeocodeNoResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"features":[]}`)
	}))
	defer ts.Close()

	svc := NewGeoapify(cfg.GeoServiceConfig{APIKey: "key", BaseURL: ts.URL}, &http.Client{})
	_, _, err := svc.Geocode("noresults")
	if err == nil {
		t.Fatal("expected error")
	}
	var e ErrGeoNoResults
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrGeoNoResults, got %v", err)
	}
}

func TestGeoapifyGeocodeBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `error`)
	}))
	defer ts.Close()

	svc := NewGeoapify(cfg.GeoServiceConfig{APIKey: "key", BaseURL: ts.URL}, &http.Client{})
	_, _, err := svc.Geocode("test")
	if err == nil {
		t.Fatal("expected error")
	}
	var e ErrGeoRequest
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrGeoRequest, got %v", err)
	}
}

func TestGeoapifyGeocodeInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{invalid json`)
	}))
	defer ts.Close()

	svc := NewGeoapify(cfg.GeoServiceConfig{APIKey: "key", BaseURL: ts.URL}, &http.Client{})
	_, _, err := svc.Geocode("test")
	if err == nil {
		t.Fatal("expected error")
	}
	var e ErrGeoDecode
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrGeoDecode, got %v", err)
	}
}

func TestGeoapifyGeocodeMissingAPIKey(t *testing.T) {
	svc := NewGeoapify(cfg.GeoServiceConfig{APIKey: "", BaseURL: "http://example.com"}, &http.Client{})
	_, _, err := svc.Geocode("test")
	if err == nil {
		t.Fatal("expected error")
	}
	var e ErrGeoConfig
	if !errors.As(err, &e) {
		t.Fatalf("expected ErrGeoConfig, got %v", err)
	}
}
