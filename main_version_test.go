package main

import "testing"

func TestShortRevision(t *testing.T) {
	if got := shortRevision("1234567890abcdef"); got != "1234567890ab" {
		t.Fatalf("shortRevision truncation mismatch: got %q", got)
	}
	if got := shortRevision("1234"); got != "1234" {
		t.Fatalf("shortRevision short value mismatch: got %q", got)
	}
}

func TestCompileDateStampUsesUTCYearDayMonth(t *testing.T) {
	got, ok := compileDateStamp("2026-04-24T02:26:27Z")
	if !ok {
		t.Fatalf("compileDateStamp returned not ok")
	}
	if got != "v26.24.04" {
		t.Fatalf("compileDateStamp mismatch: got %q", got)
	}
}

func TestCompileDateVersion(t *testing.T) {
	cases := []struct {
		name        string
		buildTime   string
		revision    string
		vcsModified string
		want        string
	}{
		{
			name:      "clean",
			buildTime: "2026-04-24T02:26:27Z",
			revision:  "78b3cd19baacffff",
			want:      "v26.24.04-78b3cd19baac",
		},
		{
			name:        "dirty",
			buildTime:   "2026-04-24T02:26:27Z",
			revision:    "78b3cd19baacffff",
			vcsModified: "true",
			want:        "v26.24.04-78b3cd19baac+dirty",
		},
		{
			name:      "unknown commit",
			buildTime: "2026-04-24T02:26:27Z",
			want:      "v26.24.04-unknown",
		},
		{
			name:     "bad date",
			revision: "78b3cd19baacffff",
			want:     "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compileDateVersion(tc.buildTime, tc.revision, tc.vcsModified); got != tc.want {
				t.Fatalf("compileDateVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsVCSModified(t *testing.T) {
	for _, value := range []string{"true", "TRUE", "1", "yes", "dirty"} {
		if !isVCSModified(value) {
			t.Fatalf("isVCSModified(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", "false", "0", "clean"} {
		if isVCSModified(value) {
			t.Fatalf("isVCSModified(%q) = true, want false", value)
		}
	}
}

func TestShouldPrintVersion(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "long", args: []string{"--version"}, want: true},
		{name: "short", args: []string{"-version"}, want: true},
		{name: "word", args: []string{"version"}, want: true},
		{name: "mixed", args: []string{"run", "--version"}, want: true},
		{name: "none", args: []string{"run"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldPrintVersion(tc.args); got != tc.want {
				t.Fatalf("shouldPrintVersion(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
