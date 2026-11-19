package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// nominatimPlace is one element of the search JSON array.
// The json tag is the key in the response. encoding/json copies "lat"
// into Lat even though the names differ, and it skips keys we do not list.
// Nominatim sends coordinates as strings, not numbers, so these fields are strings.
type nominatimPlace struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

// geocode returns lat and lon for a US ZIP, rounded to 4 decimals, plus a
// display name. The error goes back to the caller instead of exiting here,
// so main can print it and tests can check it. %w wraps the cause; errors.Is
// can still match it.
func (s *service) geocode(ctx context.Context, zip string) (lat, lon, name string, err error) {
	// Build the query with net/url. fmt.Sprintf is the wrong tool: a
	// hand-written "United%20States" contains %20S, and Sprintf treats
	// that as a format verb (go vet reports an unknown verb 'S').
	// Values.Encode escapes the space for us.
	u, err := url.Parse(strings.TrimRight(s.nominatim, "/") + "/search")
	if err != nil {
		return "", "", "", fmt.Errorf("nominatim url: %w", err)
	}
	q := u.Query()
	q.Set("postalcode", zip)
	q.Set("country", "United States")
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	// The body is a JSON array of place objects, not one object and not
	// a two-element [lat, lon] array. A slice is the matching Go type.
	var places []nominatimPlace
	if err := s.getJSON(ctx, u.String(), false, &places); err != nil {
		return "", "", "", fmt.Errorf("geocode %s: %w", zip, err)
	}
	if len(places) == 0 {
		return "", "", "", fmt.Errorf("no location for ZIP %s", zip)
	}
	// Nominatim orders hits with the best match first.
	lat, err = round4(places[0].Lat)
	if err != nil {
		return "", "", "", fmt.Errorf("geocode %s: %w", zip, err)
	}
	lon, err = round4(places[0].Lon)
	if err != nil {
		return "", "", "", fmt.Errorf("geocode %s: %w", zip, err)
	}
	return lat, lon, strings.TrimSpace(places[0].DisplayName), nil
}

// round4 formats a coordinate with 4 digits after the decimal.
// NWS asks for latitude and longitude rounded to 4 decimals.
func round4(raw string) (string, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "", fmt.Errorf("coordinate %q: %w", raw, err)
	}
	return strconv.FormatFloat(v, 'f', 4, 64), nil
}
