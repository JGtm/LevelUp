package main

// main_test.go — LE TRI DES ARTEFACTS EN CATEGORIES, ET LA GARDE QUI EMPECHE D'ECRASER.
//
// Ce qui est eprouve : chaque artefact tombe dans EXACTEMENT une categorie (c'est ce qui rend
// le total du compte rendu verifiable), un refus du detecteur n'entre JAMAIS dans les
// reparations (on ne degrade pas un T0 existant), et une ligne deja `film_movement` a la meme
// valeur n'est pas reecrite.

import (
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
)

var refStart = time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)

func ligne(t0 *int64, qualite string) ligneRegistre {
	l := ligneRegistre{startUTC: refStart, qualite: qualite}
	if t0 != nil {
		l.realStart = sql.NullTime{Valid: true, Time: refStart.Add(time.Duration(*t0) * time.Millisecond)}
	}
	return l
}

func ptr(v int64) *int64 { return &v }

func TestConfronter_Categories(t *testing.T) {
	verdicts := []verdictArtefact{
		// (1) mesure + ligne au T0 API degenere -> reparation
		{fichier: "a.json", matchID: "ma", detecte: true, t0FilmMs: 26304},
		// (2) refus du detecteur -> aucune ecriture
		{fichier: "b.json", matchID: "mb", raison: "burstTooSmall"},
		// (3) artefact inexploitable -> aucune ecriture
		{fichier: "c.json", raison: raisonSansMatchID},
		// (4) mesure mais aucune ligne au registre
		{fichier: "d.json", matchID: "md", detecte: true, t0FilmMs: 30000},
		// (5) deja film_movement a la MEME valeur -> inchange
		{fichier: "e.json", matchID: "me", detecte: true, t0FilmMs: 31862},
		// (6) deja film_movement mais valeur DIFFERENTE -> reparation (la garde protege
		//     l'identite, pas la qualite)
		{fichier: "f.json", matchID: "mf", detecte: true, t0FilmMs: 31862},
	}
	registre := map[string]ligneRegistre{
		"ma": ligne(ptr(1), string(timeline.T0QualityOK)),
		"me": ligne(ptr(31862), string(timeline.T0QualityFilmMovement)),
		"mf": ligne(ptr(26304), string(timeline.T0QualityFilmMovement)),
	}

	b := confronter(verdicts, registre)

	if len(b.reparations) != 2 {
		t.Fatalf("reparations = %d, attendu 2 (ma et mf)", len(b.reparations))
	}
	if b.reparations[0].matchID != "ma" || b.reparations[1].matchID != "mf" {
		t.Errorf("reparations = %q/%q, attendu ma/mf",
			b.reparations[0].matchID, b.reparations[1].matchID)
	}
	if b.refus["burstTooSmall"] != 1 || b.refus[raisonSansMatchID] != 1 {
		t.Errorf("refus = %v, attendu un de chaque", b.refus)
	}
	if len(b.sansLigne) != 1 || b.sansLigne[0] != "d.json" {
		t.Errorf("sansLigne = %v, attendu [d.json]", b.sansLigne)
	}
	if b.inchanges != 1 {
		t.Errorf("inchanges = %d, attendu 1", b.inchanges)
	}
	// LA PROPRIETE QUI REND LE COMPTE RENDU VERIFIABLE : les categories partitionnent le corpus.
	somme := len(b.reparations) + totalRefus(b.refus) + len(b.sansLigne) + b.inchanges
	if somme != b.total || b.total != len(verdicts) {
		t.Errorf("partition rompue : %d categorise pour %d artefacts", somme, b.total)
	}
}

// TestConfronter_AncienNULL : un match sans `real_start_time` se repare aussi, et le compte
// rendu doit pouvoir dire que l'ancien etait ABSENT — pas zero.
func TestConfronter_AncienNULL(t *testing.T) {
	b := confronter(
		[]verdictArtefact{{fichier: "a.json", matchID: "ma", detecte: true, t0FilmMs: 26304}},
		map[string]ligneRegistre{"ma": ligne(nil, "")},
	)
	if len(b.reparations) != 1 {
		t.Fatalf("reparations = %d, attendu 1", len(b.reparations))
	}
	r := b.reparations[0]
	if r.ancienMs.Valid {
		t.Errorf("ancienMs = %d, attendu invalide (real_start_time NULL)", r.ancienMs.Int64)
	}
	if attendu := refStart.Add(26304 * time.Millisecond); !r.nouveau.Equal(attendu) {
		t.Errorf("nouveau = %v, attendu %v", r.nouveau, attendu)
	}
}
