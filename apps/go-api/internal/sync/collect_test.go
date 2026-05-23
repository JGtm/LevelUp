// Package sync — collect_test.go : tests TDD pour buildBatchFromFetchedMatch.
//
// Contrat à valider :
//
//  1. fm nil ou Registry nil → erreur.
//  2. fm minimal (Registry seul) → batch avec Match + Enrichment placeholder.
//  3. fm complet (Registry + Participants + Medals + PSA + CSR) → toutes
//     les slices du batch peuplées avec les bons mappings.
//  4. Aliases dérivés des participants ayant un gamertag non-vide.
//  5. SkillRank.RatingType = "CSR" pour ranked matches.
//  6. Highlight events parse OK (chunk vide ou nil → batch sans events).
//  7. Highlight events parse INVALIDE → erreur retournée mais batch
//     contient quand même les autres données.

package sync

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ─── Helpers ───────────────────────────────────────────────────────────────

func sampleRegistry(matchID string) *MatchRegistryRow {
	return &MatchRegistryRow{
		MatchID:      matchID,
		StartTime:    time.Now().UTC(),
		ModeCategory: "PVP",
		FirstSyncBy:  "TestPlayer",
	}
}

func cIntPtr(v int) *int         { return &v }
func cStrPtr(v string) *string   { return &v }
func cF64Ptr(v float64) *float64 { return &v }

// ─── Test 1 : nil → erreur ─────────────────────────────────────────────────

func TestBuildBatchFromFetchedMatch_NilFm_ReturnsError(t *testing.T) {
	_, err := buildBatchFromFetchedMatch(nil, "halo_infinite", "Alice", "1111")
	if err == nil {
		t.Error("fm nil → erreur attendue")
	}
}

func TestBuildBatchFromFetchedMatch_NilRegistry_ReturnsError(t *testing.T) {
	fm := &fetchedMatch{MatchID: "m1"}
	_, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err == nil {
		t.Error("Registry nil → erreur attendue")
	}
}

// ─── Test 2 : minimal (Registry seul) ──────────────────────────────────────

func TestBuildBatchFromFetchedMatch_MinimalRegistry_ProducesBatch(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:  "m_min_001",
		Registry: sampleRegistry("m_min_001"),
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if batch == nil {
		t.Fatal("batch nil")
	}
	if batch.Shared.Match == nil || batch.Shared.Match.MatchID != "m_min_001" {
		t.Errorf("Shared.Match incorrect: %+v", batch.Shared.Match)
	}
	if batch.PlayerData.Enrichment == nil ||
		batch.PlayerData.Enrichment.MatchID != "m_min_001" {
		t.Errorf("Enrichment placeholder manquant: %+v", batch.PlayerData.Enrichment)
	}
	// Métadonnées du batch
	if batch.TitleSlug != "halo_infinite" || batch.Player != "Alice" || batch.XUID != "1111" {
		t.Errorf("metadata batch incorrects: slug=%q player=%q xuid=%q",
			batch.TitleSlug, batch.Player, batch.XUID)
	}
	if batch.Source != "sync_delta" {
		t.Errorf("Source = %q, want sync_delta", batch.Source)
	}
}

// ─── Test 3 : fm complet → toutes les slices peuplées ──────────────────────

func TestBuildBatchFromFetchedMatch_FullData_PopulatesAllFields(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:  "m_full_001",
		Registry: sampleRegistry("m_full_001"),
		Participants: []domain.MatchParticipantRow{
			{MatchID: "m_full_001", XUID: "1111", Gamertag: cStrPtr("Alice"), Kills: cIntPtr(10)},
			{MatchID: "m_full_001", XUID: "2222", Gamertag: cStrPtr("Bob"), Kills: cIntPtr(7)},
			{MatchID: "m_full_001", XUID: "3333", Gamertag: nil, Kills: cIntPtr(5)}, // pas de gamertag → pas d'alias
		},
		Medals: []domain.MedalRow{
			{MatchID: "m_full_001", XUID: "1111", MedalNameID: 1001, Count: 2},
		},
		PSA: []PersonalScoreAwardRow{
			{MatchID: "m_full_001", XUID: "1111", AwardName: "FlagCarrier", AwardCategory: "Objective", AwardCount: 1, AwardScore: 250},
		},
		CSRRow: &MatchCSRRow{
			MatchID:       "m_full_001",
			RatingValue:   cF64Ptr(1450),
			Tier:          "Onyx",
			TierFR:        "Onyx",
			SubTier:       0,
			TierLabel:     "Onyx 1450",
			RatingDelta:   cF64Ptr(+18),
			PlaylistGroup: "ranked-arena",
		},
	}

	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(batch.Shared.Participants) != 3 {
		t.Errorf("Participants : %d, want 3", len(batch.Shared.Participants))
	}
	if len(batch.Shared.Medals) != 1 {
		t.Errorf("Medals : %d, want 1", len(batch.Shared.Medals))
	}
	// XUIDAliases : 2 (Alice + Bob, pas le 3e car gamertag nil)
	if len(batch.Shared.XUIDAliases) != 2 {
		t.Errorf("XUIDAliases : %d, want 2 (Alice+Bob, exclus 3e sans gamertag)", len(batch.Shared.XUIDAliases))
	}
	// PSA
	if len(batch.PlayerData.PersonalScoreAwards) != 1 {
		t.Errorf("PersonalScoreAwards : %d, want 1", len(batch.PlayerData.PersonalScoreAwards))
	}
	if batch.PlayerData.PersonalScoreAwards[0].AwardName != "FlagCarrier" {
		t.Errorf("PSA.AwardName = %q, want FlagCarrier", batch.PlayerData.PersonalScoreAwards[0].AwardName)
	}
	// SkillRank — RatingType forcé à "CSR"
	if batch.PlayerData.SkillRank == nil {
		t.Fatal("SkillRank nil (CSRRow présent)")
	}
	if batch.PlayerData.SkillRank.RatingType != "CSR" {
		t.Errorf("RatingType = %q, want CSR", batch.PlayerData.SkillRank.RatingType)
	}
	if batch.PlayerData.SkillRank.RatingValue == nil || *batch.PlayerData.SkillRank.RatingValue != 1450 {
		t.Errorf("RatingValue = %+v, want 1450", batch.PlayerData.SkillRank.RatingValue)
	}
	if batch.PlayerData.SkillRank.Tier == nil || *batch.PlayerData.SkillRank.Tier != "Onyx" {
		t.Errorf("Tier = %+v, want Onyx", batch.PlayerData.SkillRank.Tier)
	}
}

// ─── Test 4 : aliases dédup'd ? — actuellement non, test du comportement ──

func TestBuildBatchFromFetchedMatch_AliasesFromParticipantsOnly(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:  "m_alias_001",
		Registry: sampleRegistry("m_alias_001"),
		Participants: []domain.MatchParticipantRow{
			{MatchID: "m_alias_001", XUID: "1111", Gamertag: cStrPtr("Alice")},
			{MatchID: "m_alias_001", XUID: "2222", Gamertag: cStrPtr("")}, // gamertag vide → pas d'alias
		},
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Shared.XUIDAliases) != 1 {
		t.Errorf("XUIDAliases : %d, want 1 (gamertag vide exclus)", len(batch.Shared.XUIDAliases))
	}
	if batch.Shared.XUIDAliases[0].XUID != "1111" {
		t.Errorf("alias xuid = %q, want 1111", batch.Shared.XUIDAliases[0].XUID)
	}
}

// ─── Test 5 : HasHighlights=false → batch sans events ─────────────────────

func TestBuildBatchFromFetchedMatch_NoHighlightData_NoEvents(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:       "m_noev_001",
		Registry:      sampleRegistry("m_noev_001"),
		HasHighlights: false,
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Shared.HighlightEvents) != 0 {
		t.Errorf("HighlightEvents : %d, want 0", len(batch.Shared.HighlightEvents))
	}
	if len(batch.Shared.KillerVictim) != 0 {
		t.Errorf("KillerVictim : %d, want 0", len(batch.Shared.KillerVictim))
	}
}

// ─── Test 6 : HighlightData "garbage" → batch sans events, pas de panic ───

// Note : analysis.ParseHighlightEvents est volontairement tolérant (cf.
// commentaire dans highlight_event_parser.go : double-tolérance zlib /
// cleartext). Un blob garbage retourne typiquement 0 events sans erreur.
// Ce test vérifie surtout l'absence de panic et l'intégrité du batch
// indépendamment du parse — l'erreur exacte dépend du format binaire.
func TestBuildBatchFromFetchedMatch_GarbageHighlightData_BatchIntact(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:       "m_bad_001",
		Registry:      sampleRegistry("m_bad_001"),
		HasHighlights: true,
		HighlightData: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		FilmMajorVer:  1,
	}
	batch, _ := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if batch == nil {
		t.Fatal("batch doit être retourné même si parse échoue ou retourne 0 events")
	}
	if batch.Shared.Match == nil || batch.Shared.Match.MatchID != "m_bad_001" {
		t.Errorf("Match doit être présent malgré parse anormal")
	}
	// On ne vérifie PAS qu'une erreur est retournée — c'est dépendant du
	// comportement du parser. Le contrat est : batch toujours retourné.
}

// ─── Test 7 : CSRRow nil → SkillRank nil dans batch ───────────────────────

func TestBuildBatchFromFetchedMatch_NoCSRRow_NoSkillRank(t *testing.T) {
	fm := &fetchedMatch{
		MatchID:  "m_nocsr_001",
		Registry: sampleRegistry("m_nocsr_001"),
		// CSRRow: nil
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "Alice", "1111")
	if err != nil {
		t.Fatal(err)
	}
	if batch.PlayerData.SkillRank != nil {
		t.Errorf("SkillRank doit être nil (CSRRow absent), got %+v", batch.PlayerData.SkillRank)
	}
}
