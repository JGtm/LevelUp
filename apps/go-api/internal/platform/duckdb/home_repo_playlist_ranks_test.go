// Package duckdb — home_repo_playlist_ranks_test.go : tests unitaires des
// helpers d'assemblage Playlists récentes (parsePlacementRemaining et
// buildPlaylistRankItem). Cible : aucun accès DB.
package duckdb

import (
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"
)

// ─── parsePlacementRemaining ────────────────────────────────────────────

func TestParsePlacementRemaining_Pluriel(t *testing.T) {
	t.Parallel()
	if got := parsePlacementRemaining("Placement (4 restants)"); got != 4 {
		t.Errorf("parsePlacementRemaining(\"Placement (4 restants)\") = %d, want 4", got)
	}
}

func TestParsePlacementRemaining_Singulier(t *testing.T) {
	t.Parallel()
	if got := parsePlacementRemaining("Placement (1 restant)"); got != 1 {
		t.Errorf("parsePlacementRemaining(\"Placement (1 restant)\") = %d, want 1", got)
	}
}

func TestParsePlacementRemaining_UnparseableReturns10(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "Onyx 3", "Placement", "Bronze 1"} {
		if got := parsePlacementRemaining(in); got != 10 {
			t.Errorf("parsePlacementRemaining(%q) = %d, want 10", in, got)
		}
	}
}

func TestParsePlacementRemaining_ClampHighValues(t *testing.T) {
	t.Parallel()
	if got := parsePlacementRemaining("Placement (99 restants)"); got != 10 {
		t.Errorf("parsePlacementRemaining(\"Placement (99 restants)\") = %d, want 10 (clamp)", got)
	}
}

// ─── buildPlaylistRankItem ──────────────────────────────────────────────

func TestBuildPlaylistRankItem_PlacementFromSnapshot(t *testing.T) {
	t.Parallel()
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Ranked Arena",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	// MSR row from placement (sync écrit rating=0, tier="Placement").
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 0, tier: "Placement", tierLabel: "Placement (4 restants)"},
	}
	snap := map[string]int{"pl-ranked": 4}

	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, snap, 10, true)

	if item.RatingValue != nil {
		t.Errorf("RatingValue: want nil (placement), got %v", *item.RatingValue)
	}
	if item.TierLabel != nil {
		t.Errorf("TierLabel: want nil (placement), got %v", *item.TierLabel)
	}
	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 4 {
		t.Errorf("MeasurementMatchesRemaining: want 4, got %v", item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL == nil || *item.BadgeImageURL == "" {
		t.Fatalf("BadgeImageURL: want unranked_6.png, got nil/empty")
	}
	if got := *item.BadgeImageURL; !contains(got, "unranked_6") {
		t.Errorf("BadgeImageURL: want unranked_6.png, got %q", got)
	}
}

func TestBuildPlaylistRankItem_PlacementFromMSROnly(t *testing.T) {
	t.Parallel()
	// Cas dégradé : snapshot absent, on parse "Placement (N restants)".
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Ranked Arena",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 0, tier: "Placement", tierLabel: "Placement (7 restants)"},
	}
	snap := map[string]int{} // pas de snapshot

	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, snap, 10, true)

	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 7 {
		t.Errorf("MeasurementMatchesRemaining: want 7, got %v", item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL == nil || !contains(*item.BadgeImageURL, "unranked_3") {
		t.Errorf("BadgeImageURL: want unranked_3.png, got %v", item.BadgeImageURL)
	}
}

func TestBuildPlaylistRankItem_StableRank(t *testing.T) {
	t.Parallel()
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Ranked Arena",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 1450, tier: "Onyx", tierLabel: "Onyx 1450", subTier: 0},
	}
	snap := map[string]int{"pl-ranked": 0} // matured

	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, snap, 10, true)

	if item.RatingValue == nil || *item.RatingValue != 1450 {
		t.Errorf("RatingValue: want 1450, got %v", item.RatingValue)
	}
	if item.TierLabel == nil || *item.TierLabel != "Onyx 1450" {
		t.Errorf("TierLabel: want \"Onyx 1450\", got %v", item.TierLabel)
	}
	if item.MeasurementMatchesRemaining != nil {
		t.Errorf("MeasurementMatchesRemaining: want nil (matured), got %v", *item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL == nil {
		t.Error("BadgeImageURL: want non-nil Onyx badge")
	}
}

func TestBuildPlaylistRankItem_MaturedRankComputesTierBand(t *testing.T) {
	t.Parallel()
	// Rang matured non-Onyx : la bande ORDINALE (sub_tier/6) + le sous-palier
	// suivant doivent être renseignés (même calcul que le skill peak). Gold IV
	// → 4/6 ≈ 66.67 %, suivant = "Or V" (frPreferred=true).
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Ranked Arena",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 1450, tier: "Gold", tierLabel: "Or IV", subTier: 4},
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, map[string]int{"pl-ranked": 0}, 10, true)

	if item.TierProgressPct == nil {
		t.Fatalf("TierProgressPct: want non-nil (matured Gold IV), got nil")
	}
	if got := *item.TierProgressPct; got < 66.0 || got > 67.0 {
		t.Errorf("TierProgressPct: want ~66.67 (4/6), got %v", got)
	}
	if item.NextTierLabel == nil || *item.NextTierLabel != "Or V" {
		t.Errorf("NextTierLabel: want \"Or V\", got %v", item.NextTierLabel)
	}
}

func TestBuildPlaylistRankItem_PlacementHasNoTierBand(t *testing.T) {
	t.Parallel()
	// Placement : pas de bande ni de sous-palier suivant (parité skill peak).
	p := playlistPhaseBRow{
		playlistID: "pl-ranked", playlistName: "Ranked Arena", isRanked: true, lastMatchID: "m1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 0, tier: "Placement", tierLabel: "Placement (4 restants)"},
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, map[string]int{"pl-ranked": 4}, 10, true)

	if item.TierProgressPct != nil {
		t.Errorf("TierProgressPct: want nil (placement), got %v", *item.TierProgressPct)
	}
	if item.NextTierLabel != nil {
		t.Errorf("NextTierLabel: want nil (placement), got %v", *item.NextTierLabel)
	}
}

func TestBuildPlaylistRankItem_PlacementFromSnapshotNoMSR(t *testing.T) {
	t.Parallel()
	// Joueur en placement, pas encore de match avec MSR enregistré.
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Ranked Arena",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	msr := map[string]playlistMSRRow{} // pas de MSR
	snap := map[string]int{"pl-ranked": 9}

	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, snap, 10, true)

	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 9 {
		t.Errorf("MeasurementMatchesRemaining: want 9, got %v", item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL == nil || !contains(*item.BadgeImageURL, "unranked_1") {
		t.Errorf("BadgeImageURL: want unranked_1.png, got %v", item.BadgeImageURL)
	}
	if item.RatingValue != nil {
		t.Errorf("RatingValue: want nil, got %v", *item.RatingValue)
	}
}

func TestBuildPlaylistRankItem_RankedNoMSRNoSnapshot(t *testing.T) {
	t.Parallel()
	// Régression : playlist ranked vue dans match_registry mais sans MSR
	// (aucun match ranked joué) ET sans snapshot CSR (sync CSR jamais lancée).
	// Doit ressortir un placement à 0/10 avec badge unranked_0.png, pas un item vide.
	p := playlistPhaseBRow{
		playlistID:   "pl-ranked",
		playlistName: "Assassin classé",
		isRanked:     true,
		lastMatchID:  "m1",
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, map[string]playlistMSRRow{}, map[string]int{}, 10, true)

	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 10 {
		t.Errorf("MeasurementMatchesRemaining: want 10 (placement à 0 match), got %v", item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL == nil || !contains(*item.BadgeImageURL, "unranked_0") {
		t.Errorf("BadgeImageURL: want unranked_0.png, got %v", item.BadgeImageURL)
	}
	if item.RatingValue != nil {
		t.Errorf("RatingValue: want nil, got %v", *item.RatingValue)
	}
	if item.RatingType == nil || *item.RatingType != "CSR" {
		t.Errorf("RatingType: want CSR, got %v", item.RatingType)
	}
}

// ─── Phase 6 : seuil dynamique S3+ (threshold=5) ─────────────────────────

func TestBuildPlaylistRankItem_Threshold5_PlacementFromSnapshot(t *testing.T) {
	t.Parallel()
	// Seuil S3+ : 5 matchs. Snapshot remaining=2 → completed=3 → 3/5 progress.
	// Mapping image proportionnel : 3*10/5 = 6 → unranked_6.png.
	p := playlistPhaseBRow{
		playlistID: "pl-ranked", playlistName: "Ranked Arena", isRanked: true, lastMatchID: "m1",
		lastSeasonID: "CsrSeason13-1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 0, tier: "Placement", tierLabel: "Placement (2 restants)"},
	}
	snap := map[string]int{"pl-ranked": 2}

	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, snap, 5, true)

	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 2 {
		t.Errorf("MeasurementMatchesRemaining: want 2, got %v", item.MeasurementMatchesRemaining)
	}
	if item.PlacementTotal == nil || *item.PlacementTotal != 5 {
		t.Errorf("PlacementTotal: want 5, got %v", item.PlacementTotal)
	}
	if item.BadgeImageURL == nil || !contains(*item.BadgeImageURL, "unranked_6") {
		t.Errorf("BadgeImageURL: want unranked_6.png (proportional 3/5 → 6/10), got %v", item.BadgeImageURL)
	}
}

func TestBuildPlaylistRankItem_Threshold5_NoMatchYet(t *testing.T) {
	t.Parallel()
	// Joueur sur playlist ranked S13 mais aucun match joué.
	// → placement 0/5, badge unranked_0.png, placement_total=5.
	p := playlistPhaseBRow{
		playlistID: "pl-ranked", playlistName: "Assassin classé", isRanked: true, lastMatchID: "m1",
		lastSeasonID: "CsrSeason13-1",
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, map[string]playlistMSRRow{}, map[string]int{}, 5, true)

	if item.MeasurementMatchesRemaining == nil || *item.MeasurementMatchesRemaining != 5 {
		t.Errorf("MeasurementMatchesRemaining: want 5, got %v", item.MeasurementMatchesRemaining)
	}
	if item.PlacementTotal == nil || *item.PlacementTotal != 5 {
		t.Errorf("PlacementTotal: want 5, got %v", item.PlacementTotal)
	}
	if item.BadgeImageURL == nil || !contains(*item.BadgeImageURL, "unranked_0") {
		t.Errorf("BadgeImageURL: want unranked_0.png, got %v", item.BadgeImageURL)
	}
}

func TestBuildPlaylistRankItem_Threshold5_MaturedRankExposesPlacementTotal(t *testing.T) {
	t.Parallel()
	// Rang matured doit aussi exposer placement_total (pour info front).
	p := playlistPhaseBRow{
		playlistID: "pl-ranked", playlistName: "Ranked Arena", isRanked: true, lastMatchID: "m1",
		lastSeasonID: "CsrSeason13-1",
	}
	msr := map[string]playlistMSRRow{
		"m1": {ratingValue: 1450, tier: "Onyx", tierLabel: "Onyx 1450"},
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, msr, map[string]int{"pl-ranked": 0}, 5, true)

	if item.PlacementTotal == nil || *item.PlacementTotal != 5 {
		t.Errorf("PlacementTotal: want 5 (matured ranked), got %v", item.PlacementTotal)
	}
	if item.MeasurementMatchesRemaining != nil {
		t.Errorf("MeasurementMatchesRemaining: want nil (matured), got %v", *item.MeasurementMatchesRemaining)
	}
}

func TestBuildPlaylistRankItem_NonRankedNoMSR(t *testing.T) {
	t.Parallel()
	// Playlist sociale jamais classée : on ne fabrique rien.
	p := playlistPhaseBRow{
		playlistID:   "pl-social",
		playlistName: "Quick Play",
		isRanked:     false,
		lastMatchID:  "m1",
	}
	item := buildPlaylistRankItem(titlepkg.DefaultSlug, p, map[string]playlistMSRRow{}, map[string]int{}, 10, true)

	if item.MeasurementMatchesRemaining != nil {
		t.Errorf("MeasurementMatchesRemaining: want nil, got %v", *item.MeasurementMatchesRemaining)
	}
	if item.BadgeImageURL != nil {
		t.Errorf("BadgeImageURL: want nil for unranked sans rating, got %v", item.BadgeImageURL)
	}
	if item.PlaylistName != "Quick Play" {
		t.Errorf("PlaylistName: want preservé, got %q", item.PlaylistName)
	}
}

// ─── newPlacementPlaylistCSR ────────────────────────────────────────────

func TestNewPlacementPlaylistCSR_DefaultValues_Threshold5(t *testing.T) {
	t.Parallel()
	p := newPlacementPlaylistCSR(titlepkg.DefaultSlug, "pl-id", "Ranked Slayer", 5)

	if p.PlaylistID != "pl-id" {
		t.Errorf("PlaylistID: want pl-id, got %q", p.PlaylistID)
	}
	if p.PlaylistName != "Ranked Slayer" {
		t.Errorf("PlaylistName: want \"Ranked Slayer\", got %q", p.PlaylistName)
	}
	if p.Current.MeasurementMatchesRemaining != 5 {
		t.Errorf("Current.MeasurementMatchesRemaining: want 5 (threshold=5), got %d", p.Current.MeasurementMatchesRemaining)
	}
	if p.Current.PlacementTotal != 5 {
		t.Errorf("Current.PlacementTotal: want 5, got %d", p.Current.PlacementTotal)
	}
	if p.Current.Tier != "" {
		t.Errorf("Current.Tier: want empty (placement), got %q", p.Current.Tier)
	}
	if p.Current.BadgeImageURL == nil || !contains(*p.Current.BadgeImageURL, "unranked_0") {
		t.Errorf("Current.BadgeImageURL: want unranked_0.png, got %v", p.Current.BadgeImageURL)
	}
}

func TestNewPlacementPlaylistCSR_LegacyThreshold10(t *testing.T) {
	t.Parallel()
	p := newPlacementPlaylistCSR(titlepkg.DefaultSlug, "pl-id", "Ranked Doubles", 10)

	if p.Current.MeasurementMatchesRemaining != 10 {
		t.Errorf("Current.MeasurementMatchesRemaining: want 10 (legacy threshold), got %d", p.Current.MeasurementMatchesRemaining)
	}
	if p.Current.PlacementTotal != 10 {
		t.Errorf("Current.PlacementTotal: want 10, got %d", p.Current.PlacementTotal)
	}
}

func TestNewPlacementPlaylistCSR_ZeroThresholdFallsBackToDefault(t *testing.T) {
	t.Parallel()
	p := newPlacementPlaylistCSR(titlepkg.DefaultSlug, "pl-id", "X", 0)

	if p.Current.MeasurementMatchesRemaining != CSRPlacementThresholdDefault {
		t.Errorf("want fallback to default (%d), got %d", CSRPlacementThresholdDefault, p.Current.MeasurementMatchesRemaining)
	}
}

// contains : helper substring, évite d'importer strings dans chaque test.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
