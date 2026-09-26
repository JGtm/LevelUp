package main

// main_test.go — LA GARDE QUI EMPECHE CE BACKFILL D'ANNULER LA REPARATION PAR LE FILM.
//
// Le T0 de ce binaire est ESTIME des `first_joined_time` de l'API. Depuis le 2026-09-02 une
// partie du corpus porte un T0 MESURE dans le film (`t0_quality = film_movement`), plus juste
// que l'estimation. Sans cette garde, une simple passe de rattrapage de ce binaire remettrait
// tout le corpus sur l'estimation — silencieusement, et sans qu'aucun test ne rougisse.

import (
	"testing"

	"levelup/go-api/internal/analysis/timeline"
)

func TestEcarterLesFilms(t *testing.T) {
	results := []result{
		{matchID: "m1", quality: timeline.T0QualityOK},
		{matchID: "m2", quality: timeline.T0QualityNoData},
		{matchID: "m3", quality: timeline.T0QualityOK},
	}

	gardes, proteges := ecarterLesFilms(results, map[string]bool{"m2": true})
	if proteges != 1 {
		t.Fatalf("proteges = %d, attendu 1", proteges)
	}
	if len(gardes) != 2 || gardes[0].matchID != "m1" || gardes[1].matchID != "m3" {
		t.Fatalf("gardes = %v, attendu m1 puis m3 (ordre preserve)", gardes)
	}
}

// TestEcarterLesFilms_AucunFilm : sans aucune ligne mesuree, la liste passe telle quelle —
// le comportement d'avant la garde.
func TestEcarterLesFilms_AucunFilm(t *testing.T) {
	results := []result{{matchID: "m1", quality: timeline.T0QualityOK}}
	gardes, proteges := ecarterLesFilms(results, map[string]bool{})
	if proteges != 0 || len(gardes) != 1 {
		t.Fatalf("gardes = %d / proteges = %d, attendu 1 / 0", len(gardes), proteges)
	}
}

// TestEcarterLesFilms_ToutMesure : un corpus entierement repare par le film ne laisse RIEN a
// ecrire — le commit devient un no-op, ce qui est exactement le but.
func TestEcarterLesFilms_ToutMesure(t *testing.T) {
	results := []result{{matchID: "m1"}, {matchID: "m2"}}
	gardes, proteges := ecarterLesFilms(results, map[string]bool{"m1": true, "m2": true})
	if len(gardes) != 0 || proteges != 2 {
		t.Fatalf("gardes = %d / proteges = %d, attendu 0 / 2", len(gardes), proteges)
	}
}
