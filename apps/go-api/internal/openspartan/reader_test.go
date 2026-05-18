package openspartan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const fixtureOwnerXUID = "2533274823110022"

func TestIsOpenSpartanDB_ValidFixture(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	ok, err := IsOpenSpartanDB(path)
	if err != nil {
		t.Fatalf("IsOpenSpartanDB: %v", err)
	}
	if !ok {
		t.Fatal("expected fixture to be detected as OpenSpartan, got false")
	}
}

func TestIsOpenSpartanDB_NotOpenSpartan(t *testing.T) {
	dir := t.TempDir()
	path := buildEmptyDB(t, dir, "empty.db")
	ok, err := IsOpenSpartanDB(path)
	if err != nil {
		t.Fatalf("IsOpenSpartanDB: %v", err)
	}
	if ok {
		t.Fatal("expected empty database to be rejected, got true")
	}
}

func TestOpen_ReturnsErrNotOpenSpartanDB_OnUnrecognizedSchema(t *testing.T) {
	dir := t.TempDir()
	path := buildEmptyDB(t, dir, "empty.db")
	_, err := Open(path)
	if !errors.Is(err, ErrNotOpenSpartanDB) {
		t.Fatalf("expected ErrNotOpenSpartanDB, got %v", err)
	}
}

func TestOpen_NonexistentFileFailsButNotAsSignatureMismatch(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(filepath.Join(dir, "does-not-exist.db"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if errors.Is(err, ErrNotOpenSpartanDB) {
		t.Fatal("missing file should not surface as ErrNotOpenSpartanDB")
	}
}

func TestReader_MatchCount(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if got := matchCountDirect(t, r); got != 3 {
		t.Fatalf("MatchCount: want 3, got %d", got)
	}
}

func TestReader_CloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close should be a no-op, got %v", err)
	}
	if _, err := r.MatchCount(context.Background()); !errors.Is(err, ErrReaderClosed) {
		t.Fatalf("after Close, MatchCount should return ErrReaderClosed, got %v", err)
	}
}

func TestDetectOwner_HighConfidenceWhenFilenameAndFrequencyAgree(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	xuid, conf, err := r.DetectOwner(context.Background(), "")
	if err != nil {
		t.Fatalf("DetectOwner: %v", err)
	}
	if xuid != fixtureOwnerXUID {
		t.Errorf("xuid: want %s, got %s", fixtureOwnerXUID, xuid)
	}
	if conf != ConfidenceHigh {
		t.Errorf("confidence: want High, got %s", conf)
	}
}

func TestDetectOwner_MediumConfidenceWhenFilenameDoesNotMatch(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, "renamed-without-xuid.db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	xuid, conf, err := r.DetectOwner(context.Background(), "")
	if err != nil {
		t.Fatalf("DetectOwner: %v", err)
	}
	if xuid != fixtureOwnerXUID {
		t.Errorf("xuid: want %s, got %s", fixtureOwnerXUID, xuid)
	}
	if conf != ConfidenceMedium {
		t.Errorf("confidence: want Medium, got %s", conf)
	}
}

func TestDetectOwner_RespectsExplicitFilenameHint(t *testing.T) {
	dir := t.TempDir()
	// Database file itself is not named after the XUID, but caller passes a
	// filename hint that *is*. The hint should take precedence over the path.
	path := buildFixtureDB(t, dir, "anonymous.db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	xuid, conf, err := r.DetectOwner(context.Background(), fixtureOwnerXUID+".db")
	if err != nil {
		t.Fatalf("DetectOwner: %v", err)
	}
	if xuid != fixtureOwnerXUID {
		t.Errorf("xuid: want %s, got %s", fixtureOwnerXUID, xuid)
	}
	if conf != ConfidenceHigh {
		t.Errorf("confidence: want High (hint + frequency agree), got %s", conf)
	}
}

func TestMatches_YieldsAllInChronologicalOrder(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	var collected []*ParsedMatch
	for pm, err := range r.Matches(context.Background()) {
		if err != nil {
			t.Fatalf("Matches yielded error: %v", err)
		}
		collected = append(collected, pm)
	}
	if len(collected) != 3 {
		t.Fatalf("Matches: want 3 entries, got %d", len(collected))
	}
	wantIDs := []string{
		"11111111-aaaa-bbbb-cccc-000000000001",
		"22222222-aaaa-bbbb-cccc-000000000002",
		"33333333-aaaa-bbbb-cccc-000000000003",
	}
	for i, pm := range collected {
		if pm.MatchID != wantIDs[i] {
			t.Errorf("match[%d]: want %s, got %s", i, wantIDs[i], pm.MatchID)
		}
		if len(pm.RawMatchStats) == 0 {
			t.Errorf("match[%d]: RawMatchStats should be preserved", i)
		}
		if len(pm.PlayerStats) == 0 {
			t.Errorf("match[%d]: PlayerStats should be parsed", i)
		}
		if len(pm.PlayerStats) > 0 && pm.PlayerStats[0].Result == nil {
			t.Errorf("match[%d]: PlayerStats[0].Result should be present", i)
		}
		if len(pm.PlayerStats) > 0 && pm.PlayerStats[0].Result != nil &&
			pm.PlayerStats[0].Result.TeamMmr != 1234.5 {
			t.Errorf("match[%d]: TeamMmr: want 1234.5, got %f", i, pm.PlayerStats[0].Result.TeamMmr)
		}
	}
}

func TestMatches_StopsWhenYieldReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	path := buildFixtureDB(t, dir, fixtureOwnerXUID+".db", fixtureOwnerXUID)
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	count := 0
	for _, err := range r.Matches(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		if count == 1 {
			break
		}
	}
	if count != 1 {
		t.Fatalf("expected iterator to stop at 1, got %d", count)
	}
}

func TestExtractXUIDFromFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2533274823110022.db", "2533274823110022"},
		{"/tmp/2533274823110022.db", "2533274823110022"},
		{"C:\\users\\me\\25332748231100229.db", "25332748231100229"},
		{"some-other-name.db", ""},
		{"2533274823110022", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := extractXUIDFromFilename(tc.in); got != tc.want {
			t.Errorf("extractXUIDFromFilename(%q): want %q, got %q", tc.in, tc.want, got)
		}
	}
}

func TestParseXUID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"xuid(2533274823110022)", "2533274823110022"},
		{"2533274823110022", "2533274823110022"},
		{"  2533274823110022  ", "2533274823110022"},
		{"bid(8589934592-100)", ""},
		{"xuid()", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := ParseXUID(tc.in); got != tc.want {
			t.Errorf("ParseXUID(%q): want %q, got %q", tc.in, tc.want, got)
		}
	}
}

func TestMain(m *testing.M) {
	// Ensure tests run in a stable temp directory regardless of CI env quirks.
	if v := os.Getenv("TMPDIR"); v == "" {
		_ = os.Setenv("TMPDIR", os.TempDir())
	}
	os.Exit(m.Run())
}
