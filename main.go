// Command gowx prints the National Weather Service forecast for a US ZIP code.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
)

func main() {
	zipFlag := flag.String("zip", "", "US ZIP code")
	hourly := flag.Bool("hourly", false, "print the hourly forecast instead of the daily summary")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: gowx [-hourly] [-zip ZIP] [ZIP]\n\n")
		fmt.Fprintf(out, "Show the National Weather Service forecast for a US ZIP code.\n")
		fmt.Fprintf(out, "With no ZIP, gowx asks for one.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*zipFlag, flag.Args(), *hourly); err != nil {
		fmt.Fprintf(os.Stderr, "gowx: %v\n", err)
		os.Exit(1)
	}
}

func run(zipFlag string, args []string, hourly bool) error {
	zip := strings.TrimSpace(zipFlag)
	if zip != "" && len(args) > 0 {
		return errors.New("pass the ZIP with -zip or as an argument, not both")
	}
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if zip == "" && len(args) == 1 {
		zip = args[0]
	}

	var err error
	if zip == "" {
		fmt.Fprint(os.Stderr, "Enter a US ZIP code: ")
		zip, err = readZIP(os.Stdin)
		if err != nil {
			return err
		}
	}
	zip, err = normalizeZIP(zip)
	if err != nil {
		return err
	}

	// NotifyContext cancels ctx on Ctrl-C. Requests use ctx, so the
	// in-flight call stops instead of waiting out the client timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	svc := newService()
	lat, lon, display, err := svc.geocode(ctx, zip)
	if err != nil {
		return err
	}
	meta, err := svc.points(ctx, lat, lon)
	if err != nil {
		return err
	}
	name := meta.name
	if name == "" {
		name = display
	}
	if name == "" {
		name = "ZIP " + zip
	}

	forecastURL := meta.forecast
	if hourly {
		if meta.hourly == "" {
			return fmt.Errorf("points %s,%s: response missing hourly forecast URL", lat, lon)
		}
		forecastURL = meta.hourly
	}
	periods, err := svc.forecast(ctx, forecastURL)
	if err != nil {
		return err
	}
	alerts, err := svc.alerts(ctx, lat, lon)
	if err != nil {
		return err
	}

	writeReport(os.Stdout, report{
		name:    name,
		lat:     lat,
		lon:     lon,
		periods: periods,
		alerts:  alerts,
		hourly:  hourly,
	})
	return nil
}

// normalizeZIP accepts 50583 and 50583-1234. Nominatim's postalcode
// query wants the 5-digit code, so a ZIP+4 suffix is dropped.
func normalizeZIP(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) == 10 && s[5] == '-' {
		if !digits(s[:5]) || !digits(s[6:]) {
			return "", fmt.Errorf("ZIP must be 5 digits (or ZIP+4), got %q", s)
		}
		return s[:5], nil
	}
	if len(s) != 5 || !digits(s) {
		return "", fmt.Errorf("ZIP must be 5 digits (or ZIP+4), got %q", s)
	}
	return s, nil
}

func digits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

func readZIP(r io.Reader) (string, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", fmt.Errorf("read ZIP: %w", err)
		}
		return "", errors.New("no ZIP entered")
	}
	return strings.TrimSpace(sc.Text()), nil
}
