package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type report struct {
	name    string
	lat     string
	lon     string
	periods []period
	alerts  []alert
	hourly  bool
}

func writeReport(w io.Writer, rep report) {
	fmt.Fprintf(w, "%s (%s, %s)\n\n", rep.name, rep.lat, rep.lon)
	if rep.hourly {
		writeHourly(w, rep.periods)
	} else {
		writeDaily(w, rep.periods)
	}
	fmt.Fprintln(w)
	writeAlerts(w, rep.alerts)
}

// writeDaily prints the current period in full, then up to 7 compact days.
// The first period is "today": a daytime high, or tonight's low when the
// forecast starts after dark. The compact lines still pair each daytime
// high with the following night's low, so today's overnight low is visible.
func writeDaily(w io.Writer, periods []period) {
	if len(periods) == 0 {
		fmt.Fprintln(w, "No forecast periods returned.")
		return
	}
	cur := periods[0]
	kind := "high"
	if !cur.IsDaytime {
		kind = "low"
	}
	fmt.Fprintln(w, "TODAY")
	fmt.Fprintf(w, "%s: %s %d°%s, wind %s\n", cur.Name, kind, cur.Temperature, cur.TemperatureUnit, windText(cur))
	if cur.DetailedForecast != "" {
		fmt.Fprintln(w, cur.DetailedForecast)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "NEXT 7 DAYS")
	days := pairDays(periods)
	if len(days) == 0 {
		fmt.Fprintln(w, "(no daytime periods)")
		return
	}
	for _, d := range days {
		low := "—"
		if d.hasLow {
			low = fmt.Sprintf("%d°%s", d.low, d.lowUnit)
		}
		fmt.Fprintf(w, "%-16s high %d°%s, low %s, %s\n", d.name, d.high, d.unit, low, d.short)
	}
}

func writeHourly(w io.Writer, periods []period) {
	fmt.Fprintln(w, "HOURLY")
	if len(periods) == 0 {
		fmt.Fprintln(w, "No forecast periods returned.")
		return
	}
	for _, p := range periods {
		fmt.Fprintf(w, "%s: %d°%s, wind %s, %s\n", periodLabel(p), p.Temperature, p.TemperatureUnit, windText(p), p.ShortForecast)
	}
}

// periodLabel is the name NWS gave the period ("Today", "Tonight").
// Hourly rows often have an empty name, so fall back to the start time.
func periodLabel(p period) string {
	if strings.TrimSpace(p.Name) != "" {
		return p.Name
	}
	t, err := time.Parse(time.RFC3339, p.StartTime)
	if err != nil {
		if p.StartTime != "" {
			return p.StartTime
		}
		return "Forecast"
	}
	return t.Format("Mon 3 PM")
}

func writeAlerts(w io.Writer, alerts []alert) {
	fmt.Fprintln(w, "ALERTS")
	if len(alerts) == 0 {
		fmt.Fprintln(w, "none")
		return
	}
	for _, a := range alerts {
		if a.Severity != "" {
			fmt.Fprintf(w, "- %s (%s)\n", a.Event, a.Severity)
		} else {
			fmt.Fprintf(w, "- %s\n", a.Event)
		}
		if h := strings.Join(strings.Fields(a.Headline), " "); h != "" {
			fmt.Fprintf(w, "  %s\n", h)
		}
	}
}

func windText(p period) string {
	w := strings.TrimSpace(p.WindSpeed + " " + p.WindDirection)
	if w == "" {
		return "n/a"
	}
	return w
}
