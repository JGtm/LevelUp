package duckdb

// tactical_repo_zone_test.go — ce que les lectures de positions rendent EN PLUS pour le détail
// d'une zone (nom en jeu, mini-tuile « Rejeu ») : la hauteur de chaque face, les noms du
// kill-feed et la source de dégât brute du film. Une hauteur ou une source absente reste ABSENTE
// (pointeur nil, chaîne vide) — jamais un zéro, qui se lirait comme une mesure.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
)

// tacKillNomme pose une mort publiable avec ses gamertags et sa source de dégât (tag et catégorie
// nil = non mesurée).
func tacKillNomme(t *testing.T, pdb *PlayerDB, matchID, killerXUID, killerGT, victimXUID, victimGT string,
	timeMS int, tag, categorie any) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms,
		 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
		 feed_present, assist_known, read_path, read_origin, source_tag, source_category)
		VALUES (?, 'pass_v1', 'rev_test', TRUE, ?, ?, ?, ?, ?, TRUE, FALSE, ?, 'credit-concordant', ?, ?)`,
		matchID, timeMS, victimGT, victimXUID, killerGT, killerXUID, killscope.ReadPathFilmWalk, tag, categorie)
}

// tacPosZ pose une position monde avec ses hauteurs (nil = non mesurée).
func tacPosZ(t *testing.T, pdb *PlayerDB, matchID, killerXUID string, timeMS int, kz, vz any) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO kill_positions
		(match_id, decode_pass, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
		VALUES (?, 'pass_test', ?, ?, 1.0, 1.0, ?, 2.0, 2.0, ?)`,
		matchID, killerXUID, timeMS, kz, vz)
}

func seedZoneCorpus(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatch(t, pdb, "z1", tacCarteA, base)
	tacParticipant(t, pdb, "z1", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "z1", tacXUIDAdv, 1, domain.OutcomeLoss)
	// t=1000 : je tue l'adversaire ; hauteur du tueur seule mesurée ; source mesurée.
	tacKillNomme(t, pdb, "z1", tacXUIDMoi, "MoiGT", tacXUIDAdv, "AdvGT", 1000, uint32(0x64ab85c4), "Headshot")
	tacPosZ(t, pdb, "z1", tacXUIDMoi, 1000, 1.5, nil)
	tacContexte(t, pdb, "z1", tacXUIDAdv, 1000, 6.0)
	// t=2000 : l'adversaire me tue ; deux hauteurs ; source NON mesurée.
	tacKillNomme(t, pdb, "z1", tacXUIDAdv, "AdvGT", tacXUIDMoi, "MoiGT", 2000, nil, nil)
	tacPosZ(t, pdb, "z1", tacXUIDAdv, 2000, 3.25, 2.75)
	tacContexte(t, pdb, "z1", tacXUIDMoi, 2000, nil)
}

func pointA(t *testing.T, pts []domain.TacticalKillPosition, timeMS int64) domain.TacticalKillPosition {
	t.Helper()
	for _, p := range pts {
		if p.TimeMs == timeMS {
			return p
		}
	}
	t.Fatalf("aucun point à t=%d : %+v", timeMS, pts)
	return domain.TacticalKillPosition{}
}

func TestTacticalRepo_KillPositions_HauteursNomsEtSource(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedZoneCorpus(t, pdb)
	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	a := pointA(t, got.Points, 1000)
	if a.KillerZ == nil || *a.KillerZ != 1.5 {
		t.Errorf("KillerZ = %v, want 1.5", a.KillerZ)
	}
	if a.VictimZ != nil {
		t.Errorf("VictimZ = %v, want nil (non mesurée, jamais 0)", *a.VictimZ)
	}
	if a.KillerGamertag != "MoiGT" || a.VictimGamertag != "AdvGT" {
		t.Errorf("gamertags = %q / %q, want MoiGT / AdvGT", a.KillerGamertag, a.VictimGamertag)
	}
	if a.SourceTag == nil || *a.SourceTag != 0x64ab85c4 || a.SourceCategory != "Headshot" {
		t.Errorf("source = %v / %q, want 0x64ab85c4 / Headshot", a.SourceTag, a.SourceCategory)
	}
	b := pointA(t, got.Points, 2000)
	if b.KillerZ == nil || b.VictimZ == nil || *b.KillerZ != 3.25 || *b.VictimZ != 2.75 {
		t.Errorf("hauteurs = %v / %v, want 3.25 / 2.75", b.KillerZ, b.VictimZ)
	}
	if b.SourceTag != nil || b.SourceCategory != "" {
		t.Errorf("source non mesurée servie %v / %q, want nil / vide", b.SourceTag, b.SourceCategory)
	}
}

func TestTacticalRepo_MortsAvecContexte_HauteurNomEtSource(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedZoneCorpus(t, pdb)
	got, err := NewTacticalRepo(pdb).MortsAvecContexte(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("MortsAvecContexte: %v", err)
	}
	if len(got.Morts) != 2 {
		t.Fatalf("morts = %+v, want 2", got.Morts)
	}
	for _, m := range got.Morts {
		switch m.TimeMs {
		case 1000:
			if m.Z != nil {
				t.Errorf("t=1000 : Z = %v, want nil (hauteur de la victime non mesurée)", *m.Z)
			}
			if m.KillerGamertag != "MoiGT" || m.SourceTag == nil || *m.SourceTag != 0x64ab85c4 || m.SourceCategory != "Headshot" {
				t.Errorf("t=1000 : %+v, want tueur MoiGT, source 0x64ab85c4 / Headshot", m)
			}
		case 2000:
			if m.Z == nil || *m.Z != 2.75 {
				t.Errorf("t=2000 : Z = %v, want 2.75 (hauteur de la VICTIME)", m.Z)
			}
			if m.KillerGamertag != "AdvGT" || m.SourceTag != nil || m.SourceCategory != "" {
				t.Errorf("t=2000 : %+v, want tueur AdvGT, source absente", m)
			}
		default:
			t.Errorf("mort inattendue : %+v", m)
		}
	}
}
