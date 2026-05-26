// Package sync — csr_writes_test.go : tests unitaires pour l'extraction CSR
// depuis le payload skill. Les tests d'UpsertCSRRow vivent dans le fichier
// _integration séparé (build tag integration, ouvre une DuckDB en mémoire).
package sync

import (
	"testing"
	"time"
)

func makeRankedRegistry(matchID string) *MatchRegistryRow {
	return &MatchRegistryRow{
		MatchID:   matchID,
		StartTime: time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
		IsRanked:  true,
	}
}

func makeSkillWithRankRecap(preValue, postValue float64, tier string, subTier, measurementRemaining int) *MatchSkillData {
	return &MatchSkillData{
		XUID: "xuid_test",
		PreMatchCSR: &CSRRankSnapshot{
			Value:                       preValue,
			Tier:                        tier,
			SubTier:                     subTier,
			MeasurementMatchesRemaining: 0,
		},
		PostMatchCSR: &CSRRankSnapshot{
			Value:                       postValue,
			Tier:                        tier,
			SubTier:                     subTier,
			MeasurementMatchesRemaining: measurementRemaining,
		},
	}
}

func TestExtractCSRRowIfRanked_RankedStable(t *testing.T) {
	reg := makeRankedRegistry("m1")
	skill := makeSkillWithRankRecap(1247, 1259, "Gold", 5, 0)
	skill.PreMatchCSR.SubTier = 4 // pre = Gold IV, post = Gold V

	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row for ranked stable match")
	}
	if row.MatchID != "m1" {
		t.Errorf("MatchID: want m1, got %q", row.MatchID)
	}
	if row.RatingValue == nil || *row.RatingValue != 1259 {
		t.Errorf("RatingValue: want 1259, got %v", row.RatingValue)
	}
	if row.Tier != "Gold" {
		t.Errorf("Tier: want Gold, got %q", row.Tier)
	}
	if row.TierFR != "Or" {
		t.Errorf("TierFR: want Or, got %q", row.TierFR)
	}
	if row.SubTier != 5 {
		t.Errorf("SubTier: want 5, got %d", row.SubTier)
	}
	if row.TierLabel != "Or V" {
		t.Errorf("TierLabel: want %q, got %q", "Or V", row.TierLabel)
	}
	if row.RatingDelta == nil || *row.RatingDelta != 12 {
		t.Errorf("RatingDelta: want +12, got %v", row.RatingDelta)
	}
	if row.PlaylistGroup != "ranked" {
		t.Errorf("PlaylistGroup: want ranked, got %q", row.PlaylistGroup)
	}
	if row.MeasurementMatchesRemaining != 0 {
		t.Errorf("MeasurementMatchesRemaining: want 0 for stable, got %d", row.MeasurementMatchesRemaining)
	}
}

func TestExtractCSRRowIfRanked_PlacementSingular(t *testing.T) {
	reg := makeRankedRegistry("m_placement_1")
	// Placement final (dernier match avant le rang final) : 1 match restant.
	skill := makeSkillWithRankRecap(0, 0, "", 0, 1)

	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row for placement match")
	}
	// 2026-05-20 : RatingValue=0.0 en placement (au lieu de nil) pour respecter
	// la contrainte NOT NULL du schéma match_skill_rank.rating_value. Le caller
	// distingue placement vs rating réel via MeasurementMatchesRemaining > 0.
	if row.RatingValue == nil || *row.RatingValue != 0.0 {
		t.Errorf("RatingValue: want 0.0 (placement, NOT NULL constraint), got %v", row.RatingValue)
	}
	if row.RatingDelta != nil {
		t.Errorf("RatingDelta: want nil (placement), got %v", *row.RatingDelta)
	}
	if row.Tier != "Placement" {
		t.Errorf("Tier: want Placement, got %q", row.Tier)
	}
	if row.TierFR != "Placement" {
		t.Errorf("TierFR: want Placement, got %q", row.TierFR)
	}
	if row.TierLabel != "Placement (1 restant)" {
		t.Errorf("TierLabel: want %q, got %q", "Placement (1 restant)", row.TierLabel)
	}
	if row.MeasurementMatchesRemaining != 1 {
		t.Errorf("MeasurementMatchesRemaining: want 1, got %d", row.MeasurementMatchesRemaining)
	}
}

func TestExtractCSRRowIfRanked_PlacementPlural(t *testing.T) {
	reg := makeRankedRegistry("m_placement_4")
	skill := makeSkillWithRankRecap(0, 0, "", 0, 4)

	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row for placement match")
	}
	if row.TierLabel != "Placement (4 restants)" {
		t.Errorf("TierLabel: want pluriel %q, got %q", "Placement (4 restants)", row.TierLabel)
	}
}

func TestExtractCSRRowIfRanked_OnyxNoSubTier(t *testing.T) {
	reg := makeRankedRegistry("m_onyx")
	// Onyx : pas de sous-tier (SubTier=0), label = "Onyx <value>"
	skill := makeSkillWithRankRecap(1820, 1850, "Onyx", 0, 0)

	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row")
	}
	if row.TierLabel != "Onyx 1850" {
		t.Errorf("TierLabel: want %q, got %q", "Onyx 1850", row.TierLabel)
	}
	if row.TierFR != "Onyx" {
		t.Errorf("TierFR for Onyx should remain 'Onyx', got %q", row.TierFR)
	}
	if row.RatingDelta == nil || *row.RatingDelta != 30 {
		t.Errorf("RatingDelta: want +30, got %v", row.RatingDelta)
	}
}

func TestExtractCSRRowIfRanked_NotRanked(t *testing.T) {
	reg := &MatchRegistryRow{MatchID: "m_social", IsRanked: false}
	skill := makeSkillWithRankRecap(1200, 1210, "Silver", 3, 0)

	row := ExtractCSRRowIfRanked(reg, skill)
	if row != nil {
		t.Fatalf("expected nil for non-ranked match, got %+v", row)
	}
}

func TestExtractCSRRowIfRanked_NoPostMatchCSR(t *testing.T) {
	// Cas où le sync a classifié le match comme ranked côté registry mais
	// l'endpoint skill a renvoyé un payload sans RankRecap (résultat
	// partiel API, classification incohérente, etc.). On préfère renvoyer
	// nil que d'insérer une row CSR vide.
	reg := makeRankedRegistry("m_no_recap")
	skill := &MatchSkillData{XUID: "xuid_test"}

	row := ExtractCSRRowIfRanked(reg, skill)
	if row != nil {
		t.Fatalf("expected nil when PostMatchCSR is absent, got %+v", row)
	}
}

func TestExtractCSRRowIfRanked_TruncatedPayloadIsRejected(t *testing.T) {
	// Régression garde-fou : si l'API Halo renvoie un payload PostMatchCsr
	// avec Tier="" ET MeasurementMatchesRemaining=0 (signature d'un payload
	// tronqué ou drift API ponctuel), on doit retourner nil pour ne pas
	// écraser une CSR valide pré-existante par un placeholder "Placement (0)".
	// Particulièrement critique pour `--csr --force` qui re-fetche tout.
	reg := makeRankedRegistry("m_truncated")
	skill := &MatchSkillData{
		XUID: "xuid_test",
		PostMatchCSR: &CSRRankSnapshot{
			Value:                       0,
			Tier:                        "",
			SubTier:                     0,
			MeasurementMatchesRemaining: 0,
		},
	}
	row := ExtractCSRRowIfRanked(reg, skill)
	if row != nil {
		t.Fatalf("expected nil for truncated payload (Tier=\"\" + Measurement=0), got %+v", row)
	}
}

func TestExtractCSRRowIfRanked_LegitimatePlacementStillAccepted(t *testing.T) {
	// Sanity : le garde-fou anti-tronqué ne doit pas casser les placements
	// légitimes (Tier="" MAIS MeasurementMatchesRemaining > 0).
	reg := makeRankedRegistry("m_placement_valid")
	skill := makeSkillWithRankRecap(0, 0, "", 0, 7)
	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row for legitimate placement match (Measurement=7)")
	}
	if row.Tier != "Placement" {
		t.Errorf("Tier: want Placement, got %q", row.Tier)
	}
	if row.MeasurementMatchesRemaining != 7 {
		t.Errorf("MeasurementMatchesRemaining: want 7, got %d", row.MeasurementMatchesRemaining)
	}
}

func TestExtractCSRRowIfRanked_NilSkillData(t *testing.T) {
	reg := makeRankedRegistry("m_skill_missing")
	row := ExtractCSRRowIfRanked(reg, nil)
	if row != nil {
		t.Fatalf("expected nil when skill data is nil (skill API failure on ranked match), got %+v", row)
	}
}

func TestTranslateTierFR_UnknownTier(t *testing.T) {
	// Forward-compat : si Microsoft ajoute un nouveau tier (ex: "Champion"
	// au-dessus d'Onyx), translateTierFR doit retourner la valeur inchangée
	// plutôt que d'écraser par vide.
	cases := map[string]string{
		"Bronze":   "Bronze",
		"Silver":   "Argent",
		"Gold":     "Or",
		"Platinum": "Platine",
		"Diamond":  "Diamant",
		"Onyx":     "Onyx",
		"Champion": "Champion", // inconnu → passthrough
		"":         "",         // vide → vide (placement géré ailleurs)
	}
	for in, want := range cases {
		if got := translateTierFR(in); got != want {
			t.Errorf("translateTierFR(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatCSRTierLabel_DiamondWithSubTier(t *testing.T) {
	// Cas standard : Diamond III avec sub-tier — chiffres romains
	got := formatCSRTierLabel("Diamond", "Diamant", 3, 1650, 0)
	if got != "Diamant III" {
		t.Errorf("Diamond III: got %q, want %q", got, "Diamant III")
	}
}

func TestFormatCSRTierLabel_SubTierZeroNotOnyx(t *testing.T) {
	// Edge case : un tier non-Onyx avec SubTier=0 (peu probable mais
	// possible si l'API renvoie un payload partiel). Doit retourner juste
	// le tier FR sans suffixe numérique.
	got := formatCSRTierLabel("Gold", "Or", 0, 1300, 0)
	if got != "Or" {
		t.Errorf("Gold no-subtier: got %q, want %q", got, "Or")
	}
}

func TestExtractCSRRowIfRanked_DeltaNilWhenNoPreMatchCSR(t *testing.T) {
	// Edge case : PostMatchCSR présent mais pas PreMatchCSR (API tronquée).
	// On insère quand même la row CSR avec RatingDelta=nil.
	reg := makeRankedRegistry("m_no_pre")
	skill := &MatchSkillData{
		XUID:         "xuid_test",
		PreMatchCSR:  nil,
		PostMatchCSR: &CSRRankSnapshot{Value: 1500, Tier: "Platinum", SubTier: 1},
	}
	row := ExtractCSRRowIfRanked(reg, skill)
	if row == nil {
		t.Fatal("expected non-nil row even without PreMatchCSR")
	}
	if row.RatingValue == nil || *row.RatingValue != 1500 {
		t.Errorf("RatingValue: want 1500, got %v", row.RatingValue)
	}
	if row.RatingDelta != nil {
		t.Errorf("RatingDelta: want nil when no PreMatchCSR, got %v", *row.RatingDelta)
	}
	if row.TierFR != "Platine" {
		t.Errorf("TierFR: want Platine, got %q", row.TierFR)
	}
}
