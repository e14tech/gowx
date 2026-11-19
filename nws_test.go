package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPairDays(t *testing.T) {
	tests := []struct {
		name string
		in   []period
		want []daySummary
	}{
		{
			name: "day then night",
			in: []period{
				{Name: "Today", IsDaytime: true, Temperature: 64, TemperatureUnit: "F", ShortForecast: "Sunny"},
				{Name: "Tonight", IsDaytime: false, Temperature: 44, TemperatureUnit: "F", ShortForecast: "Mostly Cloudy"},
				{Name: "Friday", IsDaytime: true, Temperature: 67, TemperatureUnit: "F", ShortForecast: "Mostly Sunny"},
				{Name: "Friday Night", IsDaytime: false, Temperature: 52, TemperatureUnit: "F"},
			},
			want: []daySummary{
				{name: "Today", high: 64, low: 44, hasLow: true, unit: "F", lowUnit: "F", short: "Sunny"},
				{name: "Friday", high: 67, low: 52, hasLow: true, unit: "F", lowUnit: "F", short: "Mostly Sunny"},
			},
		},
		{
			name: "starts at night",
			in: []period{
				{Name: "Tonight", IsDaytime: false, Temperature: 44, TemperatureUnit: "F", ShortForecast: "Mostly Cloudy"},
				{Name: "Friday", IsDaytime: true, Temperature: 67, TemperatureUnit: "F", ShortForecast: "Mostly Sunny"},
				{Name: "Friday Night", IsDaytime: false, Temperature: 52, TemperatureUnit: "F"},
			},
			want: []daySummary{
				{name: "Friday", high: 67, low: 52, hasLow: true, unit: "F", lowUnit: "F", short: "Mostly Sunny"},
			},
		},
		{
			name: "day without a following night",
			in: []period{
				{Name: "Today", IsDaytime: true, Temperature: 50, TemperatureUnit: "F", ShortForecast: "Sunny"},
				{Name: "Friday", IsDaytime: true, Temperature: 40, TemperatureUnit: "F", ShortForecast: "Rain"},
				{Name: "Friday Night", IsDaytime: false, Temperature: 30, TemperatureUnit: "F"},
			},
			want: []daySummary{
				{name: "Today", high: 50, hasLow: false, unit: "F", short: "Sunny"},
				{name: "Friday", high: 40, low: 30, hasLow: true, unit: "F", lowUnit: "F", short: "Rain"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pairDays(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d days %#v, want %d", len(got), got, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("day %d:\n got %+v\nwant %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestPairDaysLimitsToSeven(t *testing.T) {
	periods := make([]period, 0, 20)
	for i := 0; i < 10; i++ {
		periods = append(periods,
			period{Name: fmt.Sprintf("Day%d", i), IsDaytime: true, Temperature: 40 + i, TemperatureUnit: "F", ShortForecast: "Sun"},
			period{Name: "Night", IsDaytime: false, Temperature: 20 + i, TemperatureUnit: "F"},
		)
	}
	days := pairDays(periods)
	if len(days) != 7 {
		t.Fatalf("got %d days, want 7", len(days))
	}
	if days[0].name != "Day0" || days[6].name != "Day6" || days[6].high != 46 || days[6].low != 26 {
		t.Fatalf("unexpected ends: %+v ... %+v", days[0], days[6])
	}
}

func TestNWSJSON(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/geo+json" {
			t.Errorf("%s Accept = %q", r.URL.Path, r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "gowx-test" {
			t.Errorf("%s User-Agent = %q", r.URL.Path, r.Header.Get("User-Agent"))
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/points/"):
			if r.URL.Path != "/points/42.4171,-94.9960" {
				t.Errorf("path = %s", r.URL.Path)
			}
			fmt.Fprintf(w, `{
				"properties": {
					"forecast": %q,
					"forecastHourly": %q,
					"relativeLocation": {"properties": {"city": "Sac City", "state": "IA"}}
				}
			}`, srv.URL+"/forecast", srv.URL+"/hourly")
		case r.URL.Path == "/forecast":
			fmt.Fprint(w, `{
				"properties": {
					"periods": [
						{
							"name": "Today",
							"temperature": 64,
							"temperatureUnit": "F",
							"windSpeed": "2 to 10 mph",
							"windDirection": "SW",
							"shortForecast": "Sunny",
							"detailedForecast": "Sunny. High near 64.",
							"isDaytime": true
						},
						{
							"name": "Tonight",
							"temperature": 44,
							"temperatureUnit": "F",
							"windSpeed": "2 to 8 mph",
							"windDirection": "SE",
							"shortForecast": "Mostly Cloudy",
							"detailedForecast": "Mostly cloudy. Low around 44.",
							"isDaytime": false
						}
					]
				}
			}`)
		case r.URL.Path == "/hourly":
			fmt.Fprint(w, `{
				"properties": {
					"periods": [
						{
							"name": "",
							"startTime": "2026-11-19T14:00:00-06:00",
							"temperature": 63,
							"temperatureUnit": "F",
							"windSpeed": "5 mph",
							"windDirection": "SW",
							"shortForecast": "Sunny",
							"isDaytime": true
						}
					]
				}
			}`)
		case r.URL.Path == "/alerts/active":
			if got := r.URL.Query().Get("point"); got != "42.4171,-94.9960" {
				t.Errorf("point = %q", got)
			}
			fmt.Fprint(w, `{
				"features": [
					{"properties": {"event": "Frost Advisory", "headline": "Frost Advisory until Friday morning", "severity": "Minor"}}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, srv.URL)
	ctx := context.Background()

	meta, err := svc.points(ctx, "42.4171", "-94.9960")
	if err != nil {
		t.Fatal(err)
	}
	if meta.name != "Sac City, IA" {
		t.Fatalf("name = %q", meta.name)
	}
	if meta.forecast == "" || meta.hourly == "" {
		t.Fatalf("missing forecast URLs: %+v", meta)
	}

	periods, err := svc.forecast(ctx, meta.forecast)
	if err != nil {
		t.Fatal(err)
	}
	if len(periods) != 2 || !periods[0].IsDaytime || periods[0].DetailedForecast != "Sunny. High near 64." {
		t.Fatalf("periods = %+v", periods)
	}
	if periods[1].Name != "Tonight" || periods[1].IsDaytime || periods[1].Temperature != 44 {
		t.Fatalf("tonight = %+v", periods[1])
	}

	hourly, err := svc.forecast(ctx, meta.hourly)
	if err != nil {
		t.Fatal(err)
	}
	if len(hourly) != 1 || hourly[0].StartTime != "2026-11-19T14:00:00-06:00" || hourly[0].ShortForecast != "Sunny" {
		t.Fatalf("hourly = %+v", hourly)
	}

	alerts, err := svc.alerts(ctx, "42.4171", "-94.9960")
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Event != "Frost Advisory" || alerts[0].Severity != "Minor" {
		t.Fatalf("alerts = %+v", alerts)
	}
}

func TestAlertsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[]}`)
	}))
	defer srv.Close()

	alerts, err := newTestService(srv.URL, srv.URL).alerts(context.Background(), "42.4171", "-94.9960")
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("got %#v", alerts)
	}
}
