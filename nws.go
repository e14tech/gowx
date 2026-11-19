package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// period is one block of an NWS forecast (a daytime high or a nighttime low).
// The json tags are the field names in the response.
// Hourly periods usually leave Name empty and put the clock time in StartTime.
type period struct {
	Name             string `json:"name"`
	StartTime        string `json:"startTime"`
	Temperature      int    `json:"temperature"`
	TemperatureUnit  string `json:"temperatureUnit"`
	WindSpeed        string `json:"windSpeed"`
	WindDirection    string `json:"windDirection"`
	ShortForecast    string `json:"shortForecast"`
	DetailedForecast string `json:"detailedForecast"`
	IsDaytime        bool   `json:"isDaytime"`
}

type alert struct {
	Event    string `json:"event"`
	Headline string `json:"headline"`
	Severity string `json:"severity"`
}

// daySummary is one line of the compact list: the daytime high and the
// nighttime low that follows it.
type daySummary struct {
	name    string
	high    int
	low     int
	hasLow  bool
	unit    string
	lowUnit string
	short   string
}

type pointMeta struct {
	forecast string
	hourly   string
	name     string
}

// City and state sit two levels down, under relativeLocation.properties.
// We only list the fields we print; encoding/json ignores the rest.
type pointsResponse struct {
	Properties struct {
		Forecast         string `json:"forecast"`
		ForecastHourly   string `json:"forecastHourly"`
		RelativeLocation struct {
			Properties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type forecastResponse struct {
	Properties struct {
		Periods []period `json:"periods"`
	} `json:"properties"`
}

type alertsResponse struct {
	Features []struct {
		Properties alert `json:"properties"`
	} `json:"features"`
}

func (s *service) points(ctx context.Context, lat, lon string) (pointMeta, error) {
	// lat and lon are numbers we formatted (digits, a dot, maybe a minus),
	// so they are safe in the path. The comma is the NWS route:
	// /points/{lat},{lon}.
	raw := strings.TrimRight(s.nws, "/") + "/points/" + lat + "," + lon
	var resp pointsResponse
	if err := s.getJSON(ctx, raw, true, &resp); err != nil {
		return pointMeta{}, fmt.Errorf("points %s,%s: %w", lat, lon, err)
	}
	if resp.Properties.Forecast == "" {
		return pointMeta{}, fmt.Errorf("points %s,%s: response missing forecast URL", lat, lon)
	}
	city := resp.Properties.RelativeLocation.Properties.City
	state := resp.Properties.RelativeLocation.Properties.State
	name := strings.TrimSpace(city)
	if city != "" && state != "" {
		name = city + ", " + state
	} else if name == "" {
		name = strings.TrimSpace(state)
	}
	return pointMeta{
		forecast: resp.Properties.Forecast,
		hourly:   resp.Properties.ForecastHourly,
		name:     name,
	}, nil
}

func (s *service) forecast(ctx context.Context, forecastURL string) ([]period, error) {
	// /points already returned the full forecast URL. Request it as-is.
	var resp forecastResponse
	if err := s.getJSON(ctx, forecastURL, true, &resp); err != nil {
		return nil, fmt.Errorf("forecast: %w", err)
	}
	return resp.Properties.Periods, nil
}

func (s *service) alerts(ctx context.Context, lat, lon string) ([]alert, error) {
	u, err := url.Parse(strings.TrimRight(s.nws, "/") + "/alerts/active")
	if err != nil {
		return nil, fmt.Errorf("alerts url: %w", err)
	}
	q := u.Query()
	q.Set("point", lat+","+lon)
	u.RawQuery = q.Encode()

	var resp alertsResponse
	if err := s.getJSON(ctx, u.String(), true, &resp); err != nil {
		return nil, fmt.Errorf("alerts: %w", err)
	}
	out := make([]alert, 0, len(resp.Features))
	for _, f := range resp.Features {
		out = append(out, f.Properties)
	}
	return out, nil
}

// pairDays builds the compact 7-day list.
// A daytime period is the high. The nighttime period right after it is
// the low. If the forecast starts at night (a normal evening response),
// that first period is not a row in this list; the caller prints it on
// its own as tonight. The loop stops at 7 days.
func pairDays(periods []period) []daySummary {
	const maxDays = 7
	days := make([]daySummary, 0, maxDays)
	for i := 0; i < len(periods) && len(days) < maxDays; {
		p := periods[i]
		if !p.IsDaytime {
			i++
			continue
		}
		d := daySummary{
			name:  p.Name,
			high:  p.Temperature,
			unit:  p.TemperatureUnit,
			short: p.ShortForecast,
		}
		i++
		if i < len(periods) && !periods[i].IsDaytime {
			d.low = periods[i].Temperature
			d.lowUnit = periods[i].TemperatureUnit
			d.hasLow = true
			i++
		}
		days = append(days, d)
	}
	return days
}
