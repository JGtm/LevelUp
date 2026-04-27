package handlers

import (
	"testing"
)

func TestIsValidPlaylistKind_ValidValues(t *testing.T) {
	t.Parallel()
	for _, valid := range []string{"ranked", "social", "btb", "firefight"} {
		if !IsValidPlaylistKind(valid) {
			t.Errorf("%q should be valid", valid)
		}
	}
}

func TestIsValidPlaylistKind_EmptyAccepted(t *testing.T) {
	t.Parallel()
	// "" = pas de filtre, doit etre accepte (sinon les handlers devraient
	// rejeter chaque requete sans playlist_kind).
	if !IsValidPlaylistKind("") {
		t.Error("empty string should be valid (no filter)")
	}
	if !IsValidPlaylistKind("   ") {
		t.Error("whitespace-only should be valid (no filter)")
	}
}

func TestIsValidPlaylistKind_CaseInsensitive(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"Ranked", "RANKED", "rAnKeD", " ranked ", "\tranked\n"} {
		if !IsValidPlaylistKind(variant) {
			t.Errorf("%q should be valid (case-insensitive trimmed)", variant)
		}
	}
}

func TestIsValidPlaylistKind_RejectsUnknown(t *testing.T) {
	t.Parallel()
	for _, invalid := range []string{
		"unknown",
		"all",
		"slayer", // sous-mode, pas une categorie
		"ctf",
	} {
		if IsValidPlaylistKind(invalid) {
			t.Errorf("%q should NOT be valid", invalid)
		}
	}
}

func TestIsValidPlaylistKind_RejectsInjectionAttempts(t *testing.T) {
	t.Parallel()
	// Tests d'injection : SQL, ReDoS-like patterns, regex chars, escape
	// sequences, control chars. La whitelist ferme la porte avant que ces
	// strings n'atteignent le SQL ou un regex compiler.
	for _, malicious := range []string{
		"'; DROP TABLE shared.match_participants; --",
		"ranked OR 1=1",
		"(a+)+",            // ReDoS classique
		"^(?=(a+)+$).*",    // ReDoS catastrophic backtracking
		".*",               // wildcard regex
		"ranked\x00social", // null byte
		"ranked\nsocial",   // newline injection
		"<script>alert(1)</script>",
		"../../etc/passwd",
		"%27%20OR%201%3D1", // URL-encoded SQL injection
		"ranked ;--",
	} {
		if IsValidPlaylistKind(malicious) {
			t.Errorf("malicious input %q should be rejected", malicious)
		}
	}
}

func TestNormalisePlaylistKind_ValidReturnsCanonical(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Ranked":      "ranked",
		"RANKED":      "ranked",
		" ranked ":    "ranked",
		"social":      "social",
		"BTB":         "btb",
		"  Firefight": "firefight",
	}
	for in, want := range cases {
		if got := NormalisePlaylistKind(in); got != want {
			t.Errorf("Normalise(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalisePlaylistKind_InvalidReturnsEmpty(t *testing.T) {
	t.Parallel()
	for _, invalid := range []string{
		"", "   ", // empty -> empty (= no filter)
		"unknown",
		"slayer",
		"'; DROP TABLE",
	} {
		if got := NormalisePlaylistKind(invalid); got != "" {
			t.Errorf("Normalise(%q) = %q, want empty", invalid, got)
		}
	}
}

func TestAllowedPlaylistKinds_StableOrder(t *testing.T) {
	t.Parallel()
	got := AllowedPlaylistKinds()
	if len(got) != 4 {
		t.Fatalf("want 4 alias, got %d: %v", len(got), got)
	}
	want := []string{"btb", "firefight", "ranked", "social"} // alphabetical
	for i, w := range want {
		if got[i] != w {
			t.Errorf("position %d: want %q, got %q", i, w, got[i])
		}
	}
}

// TestPlaylistKindWhitelist_SyncWithRepo verifie indirectement que la whitelist
// du handler correspond a celle du repository (duckdb.playlistKindClause).
// Si une nouvelle valeur est ajoutee cote handler sans etre cote repo, le
// repo retournera ErrUnknownPlaylistKind a runtime. Ce test echoue tot pour
// detecter la divergence.
//
// Note : on ne peut pas importer duckdb.playlistKindClause depuis ici sans
// cycle d'import (duckdb depend de domain qui depend de... bref, pas trivial).
// Le test reste donc une garde de cardinalite : la liste fermee doit avoir
// exactement 4 entrees connues. Si on en ajoute, mettre a jour aussi
// duckdb.playlistKindClause manuellement.
func TestPlaylistKindWhitelist_FixedCardinality(t *testing.T) {
	t.Parallel()
	if len(allowedPlaylistKinds) != 4 {
		t.Errorf("whitelist size changed (got %d) -- mettre a jour"+
			" duckdb.playlistKindClause en consequence",
			len(allowedPlaylistKinds))
	}
}
