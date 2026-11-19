package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteDailyStartsInDaytime(t *testing.T) {
	var buf bytes.Buffer
	writeReport(&buf, report{
		name: "Sac City, IA",
		lat:  "42.4171",
		lon:  "-94.9960",
		periods: []period{
			{
				Name: "Today", IsDaytime: true, Temperature: 64, TemperatureUnit: "F",
				WindSpeed: "2 to 10 mph", WindDirection: "SW", ShortForecast: "Sunny",
				DetailedForecast: "Sunny. High near 64.",
			},
			{
				Name: "Tonight", IsDaytime: false, Temperature: 44, TemperatureUnit: "F",
				ShortForecast: "Mostly Cloudy",
			},
			{
				Name: "Friday", IsDaytime: true, Temperature: 67, TemperatureUnit: "F",
				ShortForecast: "Mostly Sunny",
			},
			{
				Name: "Friday Night", IsDaytime: false, Temperature: 52, TemperatureUnit: "F",
			},
		},
	})
	out := buf.String()
	today, list := splitSection(t, out, "NEXT 7 DAYS")
	if !strings.Contains(today, "Sac City, IA (42.4171, -94.9960)") {
		t.Fatalf("header missing:\n%s", today)
	}
	if !strings.Contains(today, "Today: high 64°F, wind 2 to 10 mph SW") {
		t.Fatalf("today line missing:\n%s", today)
	}
	if !strings.Contains(today, "Sunny. High near 64.") {
		t.Fatalf("detailed forecast missing:\n%s", today)
	}
	if !strings.Contains(list, "Today            high 64°F, low 44°F, Sunny") {
		t.Fatalf("today compact line missing:\n%s", list)
	}
	if !strings.Contains(list, "Friday           high 67°F, low 52°F, Mostly Sunny") {
		t.Fatalf("friday line missing:\n%s", list)
	}
	if !strings.Contains(out, "ALERTS\nnone") {
		t.Fatalf("alerts:\n%s", out)
	}
}

func TestWriteDailyStartsAtNight(t *testing.T) {
	var buf bytes.Buffer
	writeReport(&buf, report{
		name: "Sac City, IA",
		lat:  "42.4171",
		lon:  "-94.9960",
		periods: []period{
			{
				Name: "Tonight", IsDaytime: false, Temperature: 44, TemperatureUnit: "F",
				WindSpeed: "2 to 8 mph", WindDirection: "SE", ShortForecast: "Mostly Cloudy",
				DetailedForecast: "Mostly cloudy. Low around 44.",
			},
			{
				Name: "Friday", IsDaytime: true, Temperature: 67, TemperatureUnit: "F",
				ShortForecast: "Mostly Sunny",
			},
			{
				Name: "Friday Night", IsDaytime: false, Temperature: 52, TemperatureUnit: "F",
			},
		},
		alerts: []alert{{
			Event: "Frost Advisory", Severity: "Minor",
			Headline: "Frost Advisory until\nFriday morning",
		}},
	})
	out := buf.String()
	today, list := splitSection(t, out, "NEXT 7 DAYS")
	if !strings.Contains(today, "Tonight: low 44°F, wind 2 to 8 mph SE") {
		t.Fatalf("tonight line missing:\n%s", today)
	}
	if !strings.Contains(today, "Mostly cloudy. Low around 44.") {
		t.Fatalf("tonight description missing:\n%s", today)
	}
	if strings.Contains(list, "Tonight") {
		t.Fatalf("tonight should not be a 7-day row:\n%s", list)
	}
	if !strings.Contains(list, "Friday           high 67°F, low 52°F, Mostly Sunny") {
		t.Fatalf("friday line missing:\n%s", list)
	}
	if !strings.Contains(out, "- Frost Advisory (Minor)") || !strings.Contains(out, "Frost Advisory until Friday morning") {
		t.Fatalf("alert missing:\n%s", out)
	}
}

func TestWriteHourly(t *testing.T) {
	var buf bytes.Buffer
	writeReport(&buf, report{
		name:   "Sac City, IA",
		lat:    "42.4171",
		lon:    "-94.9960",
		hourly: true,
		periods: []period{{
			Name: "2pm", Temperature: 63, TemperatureUnit: "F",
			WindSpeed: "5 mph", WindDirection: "SW", ShortForecast: "Sunny",
		}},
	})
	out := buf.String()
	if strings.Contains(out, "NEXT 7 DAYS") {
		t.Fatalf("hourly view should not include the 7-day list:\n%s", out)
	}
	if !strings.Contains(out, "HOURLY") || !strings.Contains(out, "2pm: 63°F, wind 5 mph SW, Sunny") {
		t.Fatalf("hourly line missing:\n%s", out)
	}
	if !strings.Contains(out, "ALERTS\nnone") {
		t.Fatalf("alerts:\n%s", out)
	}
}

func TestWriteHourlyUsesStartTimeWhenNameEmpty(t *testing.T) {
	var buf bytes.Buffer
	writeReport(&buf, report{
		name:   "Sac City, IA",
		lat:    "42.4171",
		lon:    "-94.9960",
		hourly: true,
		periods: []period{{
			StartTime:       "2026-11-19T11:00:00-06:00",
			Temperature:     54,
			TemperatureUnit: "F",
			WindSpeed:       "10 mph",
			WindDirection:   "W",
			ShortForecast:   "Sunny",
		}},
	})
	out := buf.String()
	if !strings.Contains(out, "Thu 11 AM: 54°F, wind 10 mph W, Sunny") {
		t.Fatalf("start time not used as the label:\n%s", out)
	}
	if !strings.Contains(out, "ALERTS\nnone") {
		t.Fatalf("alerts:\n%s", out)
	}
}

func splitSection(t *testing.T, out, header string) (before, after string) {
	t.Helper()
	parts := strings.SplitN(out, header, 2)
	if len(parts) != 2 {
		t.Fatalf("output has no %s section:\n%s", header, out)
	}
	return parts[0], parts[1]
}
