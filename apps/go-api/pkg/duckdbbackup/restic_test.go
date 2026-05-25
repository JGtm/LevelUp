package duckdbbackup

import (
	"testing"
)

func TestParseSnapshotID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "valid summary last line",
			in: `{"message_type":"status","percent_done":0.5}
{"message_type":"summary","snapshot_id":"abc123def456","files_new":3}`,
			want: "abc123def456",
		},
		{
			name: "single summary line",
			in:   `{"message_type":"summary","snapshot_id":"deadbeef"}`,
			want: "deadbeef",
		},
		{
			name: "no summary line",
			in:   `{"message_type":"status","percent_done":1.0}`,
			want: "",
		},
		{
			name: "empty output",
			in:   "",
			want: "",
		},
		{
			name: "summary without snapshot_id field",
			in:   `{"message_type":"summary","files_new":0}`,
			want: "",
		},
		{
			name: "malformed json mixed with valid",
			in: `not json at all
{"message_type":"summary","snapshot_id":"cafebabe"}`,
			want: "cafebabe",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSnapshotID([]byte(tc.in))
			if got != tc.want {
				t.Errorf("parseSnapshotID: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNoPasswordFlag(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantFlag bool // true if --insecure-no-password expected
	}{
		{
			name:     "no password set → insecure flag",
			cfg:      Config{},
			wantFlag: true,
		},
		{
			name:     "ResticPassword set → no insecure flag",
			cfg:      Config{ResticPassword: "secret"},
			wantFlag: false,
		},
		{
			name:     "ResticPwdFile set → no insecure flag",
			cfg:      Config{ResticPwdFile: "/path/to/key"},
			wantFlag: false,
		},
		{
			name:     "both set → no insecure flag",
			cfg:      Config{ResticPassword: "s", ResticPwdFile: "/k"},
			wantFlag: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewResticClient(tc.cfg)
			flags := r.noPasswordFlag()
			hasFlag := len(flags) == 1 && flags[0] == "--insecure-no-password"
			if hasFlag != tc.wantFlag {
				t.Errorf("noPasswordFlag: got %v, wantFlag=%v", flags, tc.wantFlag)
			}
		})
	}
}
