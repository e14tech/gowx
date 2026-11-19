package main

import (
	"strings"
	"testing"
)

func TestNormalizeZIP(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{in: "50583", want: "50583", ok: true},
		{in: "  50583 ", want: "50583", ok: true},
		{in: "50583-1234", want: "50583", ok: true},
		{in: "5058", ok: false},
		{in: "505830", ok: false},
		{in: "abcde", ok: false},
		{in: "", ok: false},
		{in: "50583-abcd", ok: false},
	}
	for _, tc := range tests {
		got, err := normalizeZIP(tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Errorf("normalizeZIP(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("normalizeZIP(%q) = %q, want an error", tc.in, got)
		}
	}
}

func TestReadZIP(t *testing.T) {
	got, err := readZIP(strings.NewReader("  50583 \n"))
	if err != nil || got != "50583" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := readZIP(strings.NewReader("")); err == nil {
		t.Fatal("expected an error on empty input")
	}
}
