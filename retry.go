package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// service is the one HTTP setup shared by every call.
// Tests point nominatim and nws at httptest servers so they never use the network.
type service struct {
	client    *http.Client
	nominatim string
	nws       string
	userAgent string
	attempts  int
	baseWait  time.Duration
}

func newService() *service {
	return &service{
		// A timeout of 0 waits forever. Each attempt gets its own 15s budget.
		client:    &http.Client{Timeout: 15 * time.Second},
		nominatim: "https://nominatim.openstreetmap.org",
		nws:       "https://api.weather.gov",
		// Nominatim and NWS both reject an empty or generic User-Agent.
		// The value names the program and where to find it.
		userAgent: "gowx (github.com/e14tech/gowx)",
		// api.weather.gov often returns 500 on the first try.
		attempts: 4,
		baseWait: 300 * time.Millisecond,
	}
}

// getJSON GETs rawURL and decodes the JSON body into dest.
// Set nws for api.weather.gov calls so the request sends the Accept
// header that service asks for. Transient failures are retried.
func (s *service) getJSON(ctx context.Context, rawURL string, nws bool, dest any) error {
	attempts := s.attempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			// %w keeps context.Canceled in the chain, so errors.Is can see it.
			return fmt.Errorf("GET %s: %w", rawURL, err)
		}
		if attempt > 0 {
			// Exponential backoff: baseWait, then double that, then double again.
			wait := s.baseWait
			for i := 1; i < attempt; i++ {
				wait *= 2
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("GET %s: %w", rawURL, ctx.Err())
			case <-timer.C:
			}
		}

		// NewRequestWithContext attaches ctx to this attempt. When main's
		// signal handler cancels ctx, Do returns instead of running on.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", s.userAgent)
		if nws {
			req.Header.Set("Accept", "application/geo+json")
		}

		resp, err := s.client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("GET %s: %w", rawURL, ctx.Err())
			}
			lastErr = err
			continue
		}

		// io.ReadAll is the replacement for ioutil.ReadAll (deprecated
		// since Go 1.16). Close now, not with defer: defer runs when
		// getJSON returns, so a retry loop would hold every body open.
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet(body))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GET %s: HTTP %d: %s", rawURL, resp.StatusCode, snippet(body))
		}
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("decode %s: %w", rawURL, err)
		}
		return nil
	}
	return fmt.Errorf("GET %s after %d attempts: %w", rawURL, attempts, lastErr)
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
