package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestService(nominatim, nws string) *service {
	return &service{
		client:    &http.Client{Timeout: 2 * time.Second},
		nominatim: nominatim,
		nws:       nws,
		userAgent: "gowx-test",
		attempts:  4,
		baseWait:  time.Millisecond,
	}
}

func TestGetJSONRetries500Then200(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("User-Agent") != "gowx-test" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if hits == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, srv.URL)
	var dest struct {
		OK bool `json:"ok"`
	}
	if err := svc.getJSON(context.Background(), srv.URL, false, &dest); err != nil {
		t.Fatal(err)
	}
	if !dest.OK {
		t.Fatal("response was not decoded")
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2 (one 500, then 200)", hits)
	}
}

func TestGetJSONDoesNotRetry400(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "bad zip", http.StatusBadRequest)
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, srv.URL)
	var dest struct{}
	err := svc.getJSON(context.Background(), srv.URL, false, &dest)
	if err == nil {
		t.Fatal("expected an error")
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want 1 (4xx is not retried)", hits)
	}
}

func TestGetJSONHonorsCancelDuringBackoff(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	// The first attempt returns quickly. The retry wait is long, so if
	// getJSON ignores ctx it will still be sleeping when the test gives up.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	svc := newTestService(srv.URL, srv.URL)
	svc.baseWait = 30 * time.Second

	errc := make(chan error, 1)
	go func() {
		var dest struct{}
		errc <- svc.getJSON(ctx, srv.URL, false, &dest)
	}()

	select {
	case err := <-errc:
		// %w is what makes this true. %v would drop the context error.
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want context deadline", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("getJSON did not return when ctx ended")
	}
	if hits != 1 {
		t.Fatalf("hits = %d, want 1", hits)
	}
}
