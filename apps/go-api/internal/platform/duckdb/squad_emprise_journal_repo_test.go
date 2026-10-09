//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/squademprise"
)

// TestSquadEmpriseRepo_Journal_DernierePassePubliableParTueurEtArme — la vue `_latest` seule (la
// passe supplantée ne compte pas), par (tueur, clé d'arme) ; un bot (tueur NULL) et une source sans
// clé de registre marquent la passe lue sans être comptés.
func TestSquadEmpriseRepo_Journal_DernierePassePubliableParTueurEtArme(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, "A", kscTagRifle, 1_000) // passe supplantée
	for _, k := range []killEventFixture{
		{pass: kscDecodeV2, publishable: true, killerXUID: "A", tag: kscTagRifle, timeMS: 2_000},
		{pass: kscDecodeV2, publishable: true, killerXUID: "A", tag: kscTagRifle, timeMS: 3_000},
		{pass: kscDecodeV2, publishable: true, killerXUID: "B", tag: kscTagSidearm, timeMS: 4_000},
		{pass: kscDecodeV2, publishable: true, tag: kscTagRifle, timeMS: 5_000},                    // bot
		{pass: kscDecodeV2, publishable: true, killerXUID: "B", tag: kscTagInconnu, timeMS: 6_000}, // sans clé
	} {
		insertKillEvent(t, pdb, k)
	}
	got, err := NewSquadEmpriseRepo(pdb, fakeKillSourceClassifier{}).LoadJournalWeaponKills(context.Background(), []string{kscMatchID, "autre"})
	if err != nil {
		t.Fatalf("LoadJournalWeaponKills: %v", err)
	}
	if !got.Read[kscMatchID] || got.Read["autre"] || len(got.Read) != 1 {
		t.Errorf("matchs lus = %v, attendu le seul %s", got.Read, kscMatchID)
	}
	want := map[squademprise.JournalKillRow]bool{
		{MatchID: kscMatchID, XUID: "A", WeaponKey: "hinf_br75", Kills: 2}:     true,
		{MatchID: kscMatchID, XUID: "B", WeaponKey: "hinf_sidekick", Kills: 1}: true,
	}
	if len(got.Rows) != len(want) {
		t.Fatalf("lignes = %+v, attendu %v", got.Rows, want)
	}
	for _, r := range got.Rows {
		if !want[r] {
			t.Errorf("ligne inattendue %+v", r)
		}
	}
}

// Une passe non publiable ligne à ligne n'est pas lue : l'Emprise retombe sur la feuille de match.
func TestSquadEmpriseRepo_Journal_PasseNonPubliable(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, false, "A", kscTagRifle, 1_000)
	got, err := NewSquadEmpriseRepo(pdb, fakeKillSourceClassifier{}).LoadJournalWeaponKills(context.Background(), []string{kscMatchID})
	if err != nil {
		t.Fatalf("LoadJournalWeaponKills: %v", err)
	}
	if len(got.Read) != 0 || len(got.Rows) != 0 {
		t.Errorf("lecture = %+v, attendu aucun match lu", got)
	}
}

// Sans classificateur (titre sans `film.kill_source`) ou sans match : rien n'est lu, sans erreur.
func TestSquadEmpriseRepo_Journal_SansClassificateurNiMatch(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, "A", kscTagRifle, 1_000)
	for nom, lire := range map[string]func() (squademprise.JournalRead, error){
		"sans classificateur": func() (squademprise.JournalRead, error) {
			return NewSquadEmpriseRepo(pdb, nil).LoadJournalWeaponKills(context.Background(), []string{kscMatchID})
		},
		"liste vide": func() (squademprise.JournalRead, error) {
			return NewSquadEmpriseRepo(pdb, fakeKillSourceClassifier{}).LoadJournalWeaponKills(context.Background(), nil)
		},
	} {
		got, err := lire()
		if err != nil || len(got.Read) != 0 || len(got.Rows) != 0 {
			t.Errorf("%s : lecture = %+v, err = %v ; attendu vide, sans erreur", nom, got, err)
		}
	}
}
