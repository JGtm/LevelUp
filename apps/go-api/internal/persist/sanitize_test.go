// Tests pour SanitizeBatch (Phase 4 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24).
//
// Valide :
//   - Tous les champs *float64 NaN/Inf → nil
//   - Tous les champs float64 (non-nullable) NaN/Inf → 0.0
//   - Batches nil ou vides : pas de panic
//   - Idempotence : 2e appel sans effet
//   - Apres SanitizeBatch, json.Marshal du batch reussit toujours
package persist

import (
	"encoding/json"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// fpPtr alloue un *float64 pointant sur v (helper test).
func fpPtr(v float64) *float64 { return &v }

func TestSanitizeBatch_NilBatch_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SanitizeBatch(nil) doit etre no-op, panic = %v", r)
		}
	}()
	SanitizeBatch(nil)
}

func TestSanitizeBatch_EmptyBatch_NoPanic(t *testing.T) {
	batch := &MatchBatch{BatchID: "empty"}
	SanitizeBatch(batch)
	if batch.BatchID != "empty" {
		t.Errorf("SanitizeBatch a modifie BatchID")
	}
}

func TestSanitizeBatch_ParticipantNaN_BecomesNil(t *testing.T) {
	batch := &MatchBatch{
		BatchID: "test",
		Shared: SharedBatch{
			Participants: []domain.MatchParticipantRow{
				{
					MatchID:        "m1",
					XUID:           "1",
					KDA:            fpPtr(math.NaN()),
					Accuracy:       fpPtr(math.NaN()),
					KillsExpected:  fpPtr(math.Inf(1)),
					DeathsExpected: fpPtr(math.Inf(-1)),
					TeamMMR:        fpPtr(1500.0), // valide, doit etre preserve
				},
			},
		},
	}
	SanitizeBatch(batch)

	p := batch.Shared.Participants[0]
	if p.KDA != nil {
		t.Errorf("KDA = %v, want nil (NaN sanitize)", *p.KDA)
	}
	if p.Accuracy != nil {
		t.Errorf("Accuracy = %v, want nil", *p.Accuracy)
	}
	if p.KillsExpected != nil {
		t.Errorf("KillsExpected = %v, want nil (+Inf sanitize)", *p.KillsExpected)
	}
	if p.DeathsExpected != nil {
		t.Errorf("DeathsExpected = %v, want nil (-Inf sanitize)", *p.DeathsExpected)
	}
	if p.TeamMMR == nil || *p.TeamMMR != 1500.0 {
		t.Errorf("TeamMMR preserve, got %v", p.TeamMMR)
	}
}

func TestSanitizeBatch_EnrichmentNaN_BecomesNil(t *testing.T) {
	batch := &MatchBatch{
		BatchID: "test",
		PlayerData: PlayerBatch{
			Enrichment: &EnrichmentRow{
				MatchID:            "m1",
				PerformanceScore:   fpPtr(math.NaN()),
				EngagementScore:    fpPtr(72.5), // valide
				EngagementPaceTeam: fpPtr(math.Inf(1)),
			},
		},
	}
	SanitizeBatch(batch)

	e := batch.PlayerData.Enrichment
	if e.PerformanceScore != nil {
		t.Errorf("PerformanceScore = %v, want nil", *e.PerformanceScore)
	}
	if e.EngagementScore == nil || *e.EngagementScore != 72.5 {
		t.Errorf("EngagementScore preserve, got %v", e.EngagementScore)
	}
	if e.EngagementPaceTeam != nil {
		t.Errorf("EngagementPaceTeam = %v, want nil", *e.EngagementPaceTeam)
	}
}

func TestSanitizeBatch_LUSRComponentsNaN_BecomesZero(t *testing.T) {
	batch := &MatchBatch{
		BatchID: "test",
		PlayerData: PlayerBatch{
			LUSRComponents: []LUSRComponentInsert{
				{MatchID: "m1", ComponentName: "kills", Value: math.NaN(), Weight: 0.31},
				{MatchID: "m1", ComponentName: "deaths", Value: 0.5, Weight: math.Inf(1)},
			},
		},
	}
	SanitizeBatch(batch)

	c0 := batch.PlayerData.LUSRComponents[0]
	if c0.Value != 0.0 {
		t.Errorf("LUSRComponent[0].Value = %v, want 0 (NaN→0)", c0.Value)
	}
	if c0.Weight != 0.31 {
		t.Errorf("LUSRComponent[0].Weight preserve, got %v", c0.Weight)
	}
	c1 := batch.PlayerData.LUSRComponents[1]
	if c1.Weight != 0.0 {
		t.Errorf("LUSRComponent[1].Weight = %v, want 0 (+Inf→0)", c1.Weight)
	}
}

func TestSanitizeBatch_MarshalSucceedsAfterSanitize(t *testing.T) {
	// Reproduit le bug observe en prod : batch avec champ NaN → json.Marshal
	// echec. Apres SanitizeBatch, le marshal reussit.
	batch := &MatchBatch{
		BatchID: "regression-prod",
		Player:  "Chocoboflor",
		Shared: SharedBatch{
			Participants: []domain.MatchParticipantRow{
				{
					MatchID:  "508bd2fb-106f-4698-b21e-5e1502e69ee9",
					XUID:     "2535469190789936",
					Kills:    intPtr(0),
					Deaths:   intPtr(0), // deaths=0 → KDA=NaN
					KDA:      fpPtr(math.NaN()),
					Accuracy: fpPtr(math.NaN()),
				},
			},
		},
	}

	// Pre-sanitize : Marshal doit echouer.
	if _, err := json.Marshal(batch); err == nil {
		t.Skip("Go a corrige json.Marshal(NaN) — sentinelle obsolete (sanitize plus necessaire ?)")
	}

	// Apres sanitize : succes garanti.
	SanitizeBatch(batch)
	out, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("Marshal apres SanitizeBatch doit reussir, got: %v", err)
	}
	if len(out) < 50 {
		t.Errorf("Marshal produit %d bytes, suspicious", len(out))
	}
}

func TestSanitizeBatch_Idempotent(t *testing.T) {
	// 2e appel sur batch deja sanitize → no effet additionnel.
	batch := &MatchBatch{
		BatchID: "test",
		Shared: SharedBatch{
			Participants: []domain.MatchParticipantRow{
				{MatchID: "m1", XUID: "1", KDA: fpPtr(math.NaN())},
			},
		},
	}
	SanitizeBatch(batch)
	// 1er appel : KDA → nil
	if batch.Shared.Participants[0].KDA != nil {
		t.Fatalf("1er appel doit nil-ifier KDA NaN")
	}
	// 2eme appel : pas de regression
	SanitizeBatch(batch)
	if batch.Shared.Participants[0].KDA != nil {
		t.Errorf("2eme appel doit etre no-op")
	}
}

func TestSanitizeBatch_MatchRegistryIntensity_NaN(t *testing.T) {
	batch := &MatchBatch{
		BatchID: "test",
		Shared: SharedBatch{
			Match: &domain.MatchRegistryRow{
				MatchID:        "m1",
				MatchIntensity: fpPtr(math.NaN()),
			},
		},
	}
	SanitizeBatch(batch)
	if batch.Shared.Match.MatchIntensity != nil {
		t.Errorf("MatchIntensity NaN doit devenir nil")
	}
}

// intPtr est defini ailleurs dans le package — au cas ou.
func intPtr(v int) *int { return &v }
