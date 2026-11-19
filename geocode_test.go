package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeocode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("postalcode") != "50583" || q.Get("country") != "United States" || q.Get("format") != "json" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		// The old Sprintf URL embedded %20S, which is a format verb, not a space.
		if strings.Contains(r.URL.RawQuery, "%20S") {
			t.Errorf("query still has a %%20S verb: %s", r.URL.RawQuery)
		}
		if r.Header.Get("User-Agent") != "gowx-test" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Accept") == "application/geo+json" {
			t.Error("Nominatim got the NWS Accept header")
		}
		fmt.Fprint(w, `[{"lat":"42.4170794","lon":"-94.9959818","display_name":"Sac City, Iowa"},{"lat":"0","lon":"0","display_name":"ignore me"}]`)
	}))
	defer srv.Close()

	lat, lon, name, err := newTestService(srv.URL, srv.URL).geocode(context.Background(), "50583")
	if err != nil {
		t.Fatal(err)
	}
	if lat != "42.4171" || lon != "-94.9960" {
		t.Fatalf("got %s, %s; want 42.4171, -94.9960", lat, lon)
	}
	if name != "Sac City, Iowa" {
		t.Fatalf("name = %q", name)
	}
}

func TestGeocodeNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	_, _, _, err := newTestService(srv.URL, srv.URL).geocode(context.Background(), "00000")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no location") {
		t.Fatalf("err = %v", err)
	}
}
