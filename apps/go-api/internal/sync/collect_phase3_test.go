//go:build cgo

// Package sync — collect_phase3_test.go : Phase 3, bits de complétude posés au
// COLLECT (INSERT-only) sur le chemin batch. buildBatchFromFetchedMatch doit
// agréger backfill_completed (participants + skill) sur la registry row et les
// backfill_bits skill sur les participants, à parité du legacy insertFetchedMatch
// (MarkParticipantsDone / MarkSkillLoaded), sans UPDATE post-persist.
package sync

import (
	"errors"
	"testing"
	"time"
)

func TestBuildBatch_Phase3_BackfillBits(t *testing.T) {
	mmr := 1500.0
	fm := &fetchedMatch{
		MatchID:  "m-bits",
		Registry: &MatchRegistryRow{MatchID: "m-bits", StartTime: time.Now().UTC()},
		Participants: []ParticipantRow{
			{MatchID: "m-bits", XUID: "x1", TeamMMR: &mmr},
		},
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "GT", "x1")
	if err != nil {
		t.Fatalf("buildBatchFromFetchedMatch: %v", err)
	}
	if batch.Shared.Match == nil || batch.Shared.Match.BackfillCompleted == nil {
		t.Fatal("BackfillCompleted non posé sur la registry row")
	}
	bits := *batch.Shared.Match.BackfillCompleted
	if bits&backfillFlagParticipants == 0 {
		t.Errorf("bit participants manquant (bits=%d)", bits)
	}
	if bits&backfillFlagSkill == 0 {
		t.Errorf("bit skill manquant (bits=%d)", bits)
	}
	if len(batch.Shared.Participants) != 1 ||
		batch.Shared.Participants[0].BackfillBits == nil ||
		*batch.Shared.Participants[0].BackfillBits&skillBitsCombined == 0 {
		t.Error("backfill_bits skill non posé sur le participant")
	}
}

// SkillError → pas de bit skill (les colonnes skill restent NULL, le skill heal
// les complétera), mais le bit participants reste posé.
func TestBuildBatch_Phase3_NoSkillBitsOnSkillError(t *testing.T) {
	mmr := 1500.0
	fm := &fetchedMatch{
		MatchID:    "m-skillerr",
		Registry:   &MatchRegistryRow{MatchID: "m-skillerr", StartTime: time.Now().UTC()},
		SkillError: errors.New("GetMatchSkill: API down"),
		Participants: []ParticipantRow{
			{MatchID: "m-skillerr", XUID: "x1", TeamMMR: &mmr},
		},
	}
	batch, err := buildBatchFromFetchedMatch(fm, "halo_infinite", "GT", "x1")
	if err != nil {
		t.Fatalf("buildBatchFromFetchedMatch: %v", err)
	}
	var bits int64
	if batch.Shared.Match != nil && batch.Shared.Match.BackfillCompleted != nil {
		bits = *batch.Shared.Match.BackfillCompleted
	}
	if bits&backfillFlagSkill != 0 {
		t.Errorf("bit skill ne doit PAS être posé sur SkillError (bits=%d)", bits)
	}
	if bits&backfillFlagParticipants == 0 {
		t.Errorf("bit participants doit être posé même sur SkillError (bits=%d)", bits)
	}
	if len(batch.Shared.Participants) == 1 && batch.Shared.Participants[0].BackfillBits != nil &&
		*batch.Shared.Participants[0].BackfillBits&skillBitsCombined != 0 {
		t.Error("backfill_bits skill ne doit PAS être posé sur SkillError")
	}
}
