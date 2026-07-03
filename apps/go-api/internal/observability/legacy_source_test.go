package observability

import "testing"

// TestRecordLegacySourceUsed vérifie que le helper produit le bon nom de compteur
// expvar (legacy_source_used_<source>) et incrémente correctement — le compteur
// est le signal machine du gate D2 (ADR 0023 Phase 5).
func TestRecordLegacySourceUsed(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	RecordLegacySourceUsed(LegacySourceDuckDBOAuth)
	RecordLegacySourceUsed(LegacySourceDuckDBOAuth)
	RecordLegacySourceUsed(LegacySourceEnvOAuth)

	cases := map[string]int64{
		"legacy_source_used_" + LegacySourceDuckDBOAuth: 2,
		"legacy_source_used_" + LegacySourceEnvOAuth:    1,
		"legacy_source_used_" + LegacySourceDuckDBMSAL:  0,
		"legacy_source_used_" + LegacySourceMonoUser:    0,
	}
	for name, want := range cases {
		if got := LoadCounter(name); got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
}
